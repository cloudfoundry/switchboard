package domain

import "fmt"

type WsrepLocalState uint
type WsrepLocalStateComment string

const (
	Joining WsrepLocalState = iota + 1 // https://splice.com/blog/iota-elegant-constants-golang/
	DonorDesynced
	Joined
	Synced

	JoiningString       = WsrepLocalStateComment("Joining")
	DonorDesyncedString = WsrepLocalStateComment("Donor/Desynced")
	JoinedString        = WsrepLocalStateComment("Joined")
	SyncedString        = WsrepLocalStateComment("Synced")
)

type DBState struct {
	WsrepLocalIndex uint
	WsrepLocalState WsrepLocalState
	ReadOnly        bool
}

func (w WsrepLocalState) Comment() WsrepLocalStateComment {
	switch w {
	case Joining:
		return JoiningString
	case DonorDesynced:
		return DonorDesyncedString
	case Joined:
		return JoinedString
	case Synced:
		return SyncedString
	default:
		return WsrepLocalStateComment(fmt.Sprintf("Unrecognized state: %d", w))
	}
}
