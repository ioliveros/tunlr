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
	// KeyPath optionally pins the SSH private key used to connect. Empty means
	// use ssh-agent / default identities.
	KeyPath string `json:"keyPath"`
}

// SSHKey is a private key discovered in ~/.ssh, offered to the user to pick
// which identity a connection should authenticate with.
type SSHKey struct {
	Name string `json:"name"`
	Path string `json:"path"`
}
