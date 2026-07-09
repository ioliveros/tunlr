package tunnel


type State string

const (
	StateConnecting   State = "connecting"
	StateConnected    State = "connected"
	StateDisconnected State = "disconnected"
	StateError        State = "error"
	StateGivenUp      State = "given-up"
)

type HostStatus struct {
	State State  `json:"state"`
	Error string `json:"error,omitempty"`
}

type ForwardStatus struct {
	State State  `json:"state"`
	Error string `json:"error,omitempty"`
}

type Status struct {
	Hosts    map[string]HostStatus    `json:"hosts"`
	Forwards map[string]ForwardStatus `json:"forwards"`
}
