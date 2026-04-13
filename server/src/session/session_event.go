package session

type EventType string

const (
	SessionCreated EventType = "created"
	SessionDeleted EventType = "deleted"
)

type SessionEvent struct {
	Type      EventType
	SessionID string
}
