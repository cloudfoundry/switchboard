package mysql_start_mode_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGalera_MysqlStartMode(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "MysqlStartMode Suite")
}
