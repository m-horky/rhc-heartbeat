package heartbeat

// Kind identifies why a heartbeat was collected.
type Kind string

const (
	// KindOn identifies a heartbeat reporting that the system is able to send updates.
	KindOn Kind = "on"
	// KindOff identifies that the system is not expected to be able to send updates for the foreseeable future.
	KindOff Kind = "off"
	// KindPing identifies a periodic heartbeat.
	KindPing Kind = "ping"
)

// Valid reports whether kind is one of the supported heartbeat kinds.
func (kind Kind) Valid() bool {
	switch kind {
	case KindOn, KindOff, KindPing:
		return true
	default:
		return false
	}
}
