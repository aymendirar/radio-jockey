package session

type SessionManagerEventType string

const (
	SessionCreated SessionManagerEventType = "created"
	SessionDeleted SessionManagerEventType = "deleted"
)

type SessionManagerEvent struct {
	Type      SessionManagerEventType
	SessionID SessionID
}

type SessionQueueEventType string

const (
	SkipTrack SessionQueueEventType = "skip"
)

type SessionQueueEvent struct {
	Type SessionQueueEventType
}
