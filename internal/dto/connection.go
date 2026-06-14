package dto

// ConnectionInput is the payload for creating a connection: a single SSH
// port forward plus, when needed, the bastion host it is tunneled through.
//
// Domain is the SSH endpoint used to connect, in the form
// "[user@]hostname[:port]" (e.g. "dev@me.ioliveros.dev"). Host/RemotePort
// are the target reachable from that bastion, forwarded to LocalPort.
type ConnectionInput struct {
	ConnectionName string `json:"connectionName"`
	Host           string `json:"host"`
	RemotePort     int    `json:"remotePort"`
	LocalPort      int    `json:"localPort"`
	Domain         string `json:"domain"`
}
