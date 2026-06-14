package tunnel

import (
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/ioliveros/tunlr/internal/model"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// AuthMethods builds the SSH auth methods for a host: an explicit key when
// configured, otherwise ssh-agent, falling back to the usual default identity
// files. This mirrors how a plain `ssh user@host` would authenticate.
func AuthMethods(host model.Host) []ssh.AuthMethod {
	if host.AuthMethod == model.AuthKey && host.KeyPath != "" {
		if signer, err := LoadKey(host.KeyPath); err == nil {
			return []ssh.AuthMethod{ssh.PublicKeys(signer)}
		}
	}

	if m := AgentAuth(); m != nil {
		return []ssh.AuthMethod{m}
	}

	if signers := DefaultSigners(); len(signers) > 0 {
		return []ssh.AuthMethod{ssh.PublicKeys(signers...)}
	}
	return nil
}

func AgentAuth() ssh.AuthMethod {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil
	}
	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers)
}

func LoadKey(path string) (ssh.Signer, error) {
	b, err := os.ReadFile(ExpandPath(path))
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(b)
}

func DefaultSigners() []ssh.Signer {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var signers []ssh.Signer
	for _, name := range []string{"id_ed25519", "id_rsa", "id_ecdsa"} {
		b, err := os.ReadFile(filepath.Join(home, ".ssh", name))
		if err != nil {
			continue
		}
		if s, err := ssh.ParsePrivateKey(b); err == nil {
			signers = append(signers, s)
		}
	}
	return signers
}

func ExpandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
