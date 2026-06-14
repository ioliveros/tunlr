package tunnel

// State is the connection state of a host or a forward.
type State string

const (
	StateConnecting   State = "connecting"
	StateConnected    State = "connected"
	StateDisconnected State = "disconnected"
	StateError        State = "error"
	StateGivenUp      State = "given-up"
)

// HostStatus is the live state of a bastion's SSH connection.
type HostStatus struct {
	State State  `json:"state"`
	Error string `json:"error,omitempty"`
}

// ForwardStatus is the live state of a single port forward's local listener.
type ForwardStatus struct {
	State State  `json:"state"`
	Error string `json:"error,omitempty"`
}

// Status is a snapshot of all managed hosts and forwards, keyed by their
// (string-encoded) database IDs so it serializes cleanly to the frontend.
type Status struct {
	Hosts    map[string]HostStatus    `json:"hosts"`
	Forwards map[string]ForwardStatus `json:"forwards"`
}
