package tunnel

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/ioliveros/tunlr/internal/model"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var knownHostsMu sync.Mutex

// HostKeyCallback verifies the server's host key against ~/.ssh/known_hosts.
// With the accept-new policy unknown hosts are trusted on first use and
// persisted; a changed key for a known host is always rejected.
func HostKeyCallback(policy model.HostKeyPolicy) (ssh.HostKeyCallback, error) {
	path, err := KnownHostsPath()
	if err != nil {
		return nil, err
	}
	base, err := knownhosts.New(path)
	if err != nil {
		return nil, err
	}
	if policy == model.HostKeyStrict {
		return base, nil
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := base(hostname, remote, key)
		if err == nil {
			return nil
		}
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
			// Unknown host: trust on first use and record it.
			return AppendKnownHost(path, hostname, remote, key)
		}
		return err
	}, nil
}

func KnownHostsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "known_hosts")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return "", err
	}
	_ = f.Close()
	return path, nil
}

func AppendKnownHost(path, hostname string, remote net.Addr, key ssh.PublicKey) error {
	knownHostsMu.Lock()
	defer knownHostsMu.Unlock()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	addrs := []string{knownhosts.Normalize(hostname)}
	if remote != nil {
		if r := knownhosts.Normalize(remote.String()); r != addrs[0] {
			addrs = append(addrs, r)
		}
	}
	_, err = fmt.Fprintln(f, knownhosts.Line(addrs, key))
	return err
}
