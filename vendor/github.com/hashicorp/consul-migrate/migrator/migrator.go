package migrator

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hashicorp/raft"
	"github.com/hashicorp/raft-boltdb"
	"github.com/hashicorp/raft-mdb"
)

const (
	// Path to the raft directory
	raftDir = "raft"

	// Path to the legacy MDB data and its backup location
	mdbDir       = "mdb"
	mdbBackupDir = "mdb.backup"

	// The name of the BoltDB file in the raft path
	boltFile     = "raft.db"
	boltTempFile = "raft.db.temp"

	// Maximum log sizes for LMDB. These mirror settings in Consul
	// and are automatically set based on the runtime.
	maxLogSize32bit uint64 = 8 * 1024 * 1024 * 1024
	maxLogSize64bit uint64 = 64 * 1024 * 1024 * 1024
)

var (
	// Common error messages from migrator
	errFirstIndexZero = fmt.Errorf("No logs found (first index was 0)")
	errLastIndexZero  = fmt.Errorf("No logs found (last index was 0)")

	// stableStoreKeys are the well-known keys written to the
	// stable store, and are used internally by Raft. We hard-code
	// them here so that we can copy them explicitly.
	stableStoreKeys [][]byte = [][]byte{
		[]byte("CurrentTerm"),
		[]byte("LastVoteTerm"),
		[]byte("LastVoteCand"),
	}
)

// Migrator is used to migrate the Consul data storage format on
// servers with versions <= 0.5.0. Consul versions >= 0.5.1 use
// BoltDB internally as the store for the Raft log. During this
// transition, it is necessary to copy data out of our LMDB store
// and create a new BoltStore with the same data.
type Migrator struct {
	// Channels used to expose what's happening internally
	// during a migration.
	ProgressCh chan *ProgressUpdate

	dataDir   string                // The Consul data-dir
	mdbStore  *raftmdb.MDBStore     // The legacy MDB environment
	boltStore *raftboltdb.BoltStore // Handle for the new store

	// Calculated paths based on the data dir
	raftPath      string
	mdbPath       string
	mdbBackupPath string
	boltPath      string
	boltTempPath  string
}

// New creates a new Migrator given the path to a Consul
// data-dir. Returns the new Migrator and any error.
func New(dataDir string) (*Migrator, error) {
	// Check that the directory exists
	if _, err := os.Stat(dataDir); err != nil {
		return nil, err
	}

	// Create the struct
	m := &Migrator{
		ProgressCh: make(chan *ProgressUpdate, 128),
		dataDir:    dataDir,

		raftPath:      filepath.Join(dataDir, raftDir),
		mdbPath:       filepath.Join(dataDir, raftDir, mdbDir),
		mdbBackupPath: filepath.Join(dataDir, raftDir, mdbBackupDir),
		boltPath:      filepath.Join(dataDir, raftDir, boltFile),
		boltTempPath:  filepath.Join(dataDir, raftDir, boltTempFile),
	}

	return m, nil
}

// mdbConnect is used to open a handle on our LMDB raft backend. This
// is enough to read all of the Consul data we need to migrate.
func (m *Migrator) mdbConnect(dir string) error {
	// Calculate and set the max size
	size := maxLogSize32bit
	if runtime.GOARCH == "amd64" {
		size = maxLogSize64bit
	}

	// Open the connection
	mdb, err := raftmdb.NewMDBStoreWithSize(dir, size)
	if err != nil {
		return err
	}

	// Return the new environment
	m.mdbStore = mdb
	return nil
}

// boltConnect creates a new BoltStore to copy our data into. We can
// use the BoltStore directly because it provides simple setter
// methods, provided our keys and values are known.
func (m *Migrator) boltConnect(file string) error {
	// Connect to the new BoltStore
	store, err := raftboltdb.NewBoltStore(file)
	if err != nil {
		return err
	}

	m.boltStore = store
	return nil
}

// migrateStableStore copies values out of the origin StableStore
// and writes them into the destination. There are only a handful
// of keys we need, so we copy them explicitly.
func (m *Migrator) migrateStableStore() error {
	op := "Migrating stable store"
	m.sendProgress(op, 0, 1)

	total := len(stableStoreKeys)
	for i, key := range stableStoreKeys {
		val, err := m.mdbStore.Get(key)
		if err != nil {
			if err.Error() != "not found" {
				return fmt.Errorf("Error getting key '%s': %s", string(key), err)
			}
			m.sendProgress(op, i+1, total)
			continue
		}
		if err := m.boltStore.Set(key, val); err != nil {
			return fmt.Errorf("Error storing key '%s': %s", string(key), err)
		}
		m.sendProgress(op, i+1, total)
	}
	return nil
}

// migrateLogStore is like migrateStableStore, but iterates over
// all of our Raft logs and copies them into the new BoltStore.
func (m *Migrator) migrateLogStore() error {
	op := "Migrating log store"
	m.sendProgress(op, 0, 1)

	first, err := m.mdbStore.FirstIndex()
	if err != nil {
		return err
	}
	if first == 0 {
		return errFirstIndexZero
	}

	last, err := m.mdbStore.LastIndex()
	if err != nil {
		return err
	}
	if last == 0 {
		return errLastIndexZero
	}
	total := int(last - first)

	current := 0
	for i := first; i <= last; i++ {
		log := &raft.Log{}
		if err := m.mdbStore.GetLog(i, log); err != nil {
			return err
		}
		if err := m.boltStore.StoreLog(log); err != nil {
			return err
		}
		current++
		m.sendProgress(op, current, total)
	}
	return nil
}

// activateBoltStore wraps moving the Bolt file into place after
// a data migration has finished successfully.
func (m *Migrator) activateBoltStore() error {
	op := "Moving Bolt file into place"
	m.sendProgress(op, 0, 1)

	if err := os.Rename(m.boltTempPath, m.boltPath); err != nil {
		return err
	}

	m.sendProgress(op, 1, 1)
	return nil
}

// archiveMDBStore is used to move the LMDB data directory to a backup
// location so that it is not used by Consul again.
func (m *Migrator) archiveMDBStore() error {
	op := "Archiving LMDB data"
	m.sendProgress(op, 0, 1)

	if err := os.Rename(m.mdbPath, m.mdbBackupPath); err != nil {
		os.Remove(m.boltPath)
		return err
	}

	m.sendProgress(op, 1, 1)
	return nil
}

// Migrate is the high-level function we call when we want to attempt
// to migrate all of our LMDB data into BoltDB. If an error is
// encountered, the BoltStore is nuked from disk, since it is useless.
// The migration can be attempted again, as the LMDB data should
// still be intact. Returns a bool indicating whether a migration
// was completed, and any error.
func (m *Migrator) Migrate() (bool, error) {
	// Check if we should attempt a migration
	if _, err := os.Stat(m.mdbPath); os.IsNotExist(err) {
		return false, nil
	}

	// Connect the stores
	if err := m.mdbConnect(m.raftPath); err != nil {
		return false, fmt.Errorf("Failed to connect MDB: %s", err)
	}
	defer m.mdbStore.Close()

	if err := m.boltConnect(m.boltTempPath); err != nil {
		return false, fmt.Errorf("Failed to connect BoltDB: %s", err)
	}
	defer m.boltStore.Close()

	// Ensure we clean up the temp file during failure cases
	defer os.Remove(m.boltTempPath)

	// Migrate the stable store
	if err := m.migrateStableStore(); err != nil {
		return false, fmt.Errorf("Failed to migrate stable store: %v", err)
	}

	// Migrate the log store
	if err := m.migrateLogStore(); err != nil {
		return false, fmt.Errorf("Failed to migrate log store: %v", err)
	}

	// Activate the new BoltDB file
	if err := m.activateBoltStore(); err != nil {
		return false, fmt.Errorf("Failed to activate Bolt store: %s", err)
	}

	// Move the old MDB dir to its backup location
	if err := m.archiveMDBStore(); err != nil {
		return false, fmt.Errorf("Failed to archive LMDB data: %s", err)
	}

	return true, nil
}

// sendProgress is used to send a progress update message to the progress
// channel. Sending is a non-blocking operation. It is the responsibility
// of the caller to ensure they drain the queue promptly to avoid missing
// progress update messages.
func (m *Migrator) sendProgress(op string, done, total int) {
	update := ProgressUpdate{op, (float64(done) / float64(total)) * 100}
	select {
	case m.ProgressCh <- &update:
	default:
	}
}

// ProgressUpdate is used to communicate internal progress data about
// an in-flight migration. The Op is the current operation, and the
// Progress indicates a percentage of migrations completed.
type ProgressUpdate struct {
	Op       string
	Progress float64
}
