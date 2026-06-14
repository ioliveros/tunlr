package service

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ioliveros/tunlr/internal/dto"
	"github.com/ioliveros/tunlr/internal/model"
	"github.com/ioliveros/tunlr/internal/repository"
	"github.com/ioliveros/tunlr/internal/tunnel"
)

// Engine is the tunnel runtime the service drives. *tunnel.Manager satisfies
// it; tests inject a no-op so persistence can be exercised without dialing SSH.
type Engine interface {
	Apply(host model.Host)
	StopHost(id uint)
	StopForward(hostID, forwardID uint)
	ReconnectHost(id uint)
	Snapshot() tunnel.Status
}

// TunnelService coordinates persistence of tunnel configuration and drives the
// SSH tunnel engine (connect/disconnect/status).
type TunnelService struct {
	hosts  *repository.HostRepository
	engine Engine
}

func NewTunnelService(hosts *repository.HostRepository, engine Engine) *TunnelService {
	return &TunnelService{hosts: hosts, engine: engine}
}

// StartAll brings up tunnels for every persisted host. Call once at startup.
func (s *TunnelService) StartAll() error {
	hosts, err := s.hosts.ListHosts()
	if err != nil {
		return err
	}
	for _, h := range hosts {
		s.engine.Apply(h)
	}
	return nil
}

// Status returns the live connection status of all hosts and forwards.
func (s *TunnelService) Status() tunnel.Status { return s.engine.Snapshot() }

// ListSSHKeys returns the private keys found in ~/.ssh (those with a sibling
// .pub file), so the UI can let the user pick which identity to connect with.
func (s *TunnelService) ListSSHKeys() []dto.SSHKey {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dir := filepath.Join(home, ".ssh")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	keys := make([]dto.SSHKey, 0)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasSuffix(name, ".pub") {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, name+".pub")); err == nil {
			keys = append(keys, dto.SSHKey{Name: name, Path: filepath.Join(dir, name)})
		}
	}
	return keys
}

// SetHostKey pins (or clears) the SSH key a host authenticates with and
// reconnects so the new credentials take effect immediately.
func (s *TunnelService) SetHostKey(hostID uint, keyPath string) (*model.Host, error) {
	host, err := s.hosts.GetHost(hostID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(keyPath) == "" {
		host.AuthMethod = model.AuthAgent
		host.KeyPath = ""
	} else {
		host.AuthMethod = model.AuthKey
		host.KeyPath = strings.TrimSpace(keyPath)
	}
	if err := s.hosts.UpdateHost(host); err != nil {
		return nil, err
	}

	full, err := s.hosts.GetHost(hostID)
	if err != nil {
		return nil, err
	}
	s.engine.StopHost(hostID)
	s.engine.Apply(*full)
	return full, nil
}

func (s *TunnelService) ReconnectHost(id uint) { s.engine.ReconnectHost(id) }

func (s *TunnelService) ListHosts() ([]model.Host, error) { return s.hosts.ListHosts() }

func (s *TunnelService) GetHost(id uint) (*model.Host, error) { return s.hosts.GetHost(id) }

func (s *TunnelService) CreateHost(host model.Host) (*model.Host, error) {
	if err := s.hosts.CreateHost(&host); err != nil {
		return nil, err
	}
	return s.hosts.GetHost(host.ID)
}

func (s *TunnelService) UpdateHost(host model.Host) (*model.Host, error) {
	if err := s.hosts.UpdateHost(&host); err != nil {
		return nil, err
	}
	return s.hosts.GetHost(host.ID)
}

func (s *TunnelService) DeleteHost(id uint) error {
	s.engine.StopHost(id)
	return s.hosts.DeleteHost(id)
}

func (s *TunnelService) AddForward(f model.Forward) (*model.Forward, error) {
	if err := s.hosts.CreateForward(&f); err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *TunnelService) UpdateForward(f model.Forward) (*model.Forward, error) {
	if err := s.hosts.UpdateForward(&f); err != nil {
		return nil, err
	}
	// Restart the listener so changed ports take effect: drop it, then re-sync
	// the host so the engine recreates it from the updated config.
	s.engine.StopForward(f.HostID, f.ID)
	if host, err := s.hosts.GetHost(f.HostID); err == nil {
		s.engine.Apply(*host)
	}
	return &f, nil
}

// DeleteForward removes a forward and, when it was the last forward under its
// (implicitly created) bastion host, removes the now-empty host too.
func (s *TunnelService) DeleteForward(id uint) error {
	fwd, err := s.hosts.GetForward(id)
	if err != nil {
		return err
	}
	if err := s.hosts.DeleteForward(id); err != nil {
		return err
	}
	count, err := s.hosts.CountForwardsForHost(fwd.HostID)
	if err != nil {
		return err
	}
	if count == 0 {
		s.engine.StopHost(fwd.HostID)
		return s.hosts.DeleteHost(fwd.HostID)
	}
	s.engine.StopForward(fwd.HostID, id)
	return nil
}

// AddConnection creates a port forward, finding or creating the bastion host
// described by the connection's domain so that connections sharing a domain
// are grouped under one host.
func (s *TunnelService) AddConnection(in dto.ConnectionInput) (*model.Host, error) {
	sshUser, hostname, port := parseSSHTarget(in.Domain)
	if hostname == "" {
		return nil, fmt.Errorf("a domain to connect through is required")
	}
	if sshUser == "" {
		sshUser = currentUsername()
	}

	keyPath := strings.TrimSpace(in.KeyPath)

	host, err := s.hosts.FindHost(sshUser, hostname, port)
	if err != nil {
		return nil, err
	}
	keyChanged := false
	if host == nil {
		host = &model.Host{
			Name:          hostname,
			Hostname:      hostname,
			User:          sshUser,
			Port:          port,
			AuthMethod:    model.AuthAgent,
			HostKeyPolicy: model.HostKeyAcceptNew,
		}
		if keyPath != "" {
			host.AuthMethod = model.AuthKey
			host.KeyPath = keyPath
		}
		if err := s.hosts.CreateHost(host); err != nil {
			return nil, err
		}
	} else if keyPath != "" && (host.AuthMethod != model.AuthKey || host.KeyPath != keyPath) {
		host.AuthMethod = model.AuthKey
		host.KeyPath = keyPath
		if err := s.hosts.UpdateHost(host); err != nil {
			return nil, err
		}
		keyChanged = true
	}

	fwd := &model.Forward{
		HostID:     host.ID,
		Label:      strings.TrimSpace(in.ConnectionName),
		RemoteHost: strings.TrimSpace(in.Host),
		RemotePort: in.RemotePort,
		LocalPort:  in.LocalPort,
		Enabled:    true,
	}
	if err := s.hosts.CreateForward(fwd); err != nil {
		return nil, err
	}

	full, err := s.hosts.GetHost(host.ID)
	if err != nil {
		return nil, err
	}
	// If we changed an already-connected host's key, force a reconnect so the
	// new credentials apply; otherwise just bring it up / add the new forward.
	if keyChanged {
		s.engine.StopHost(full.ID)
	}
	s.engine.Apply(*full)
	return full, nil
}

// parseSSHTarget splits "[user@]hostname[:port]" into its parts. Port
// defaults to 22 when absent or unparseable.
func parseSSHTarget(s string) (sshUser, hostname string, port int) {
	port = 22
	s = strings.TrimSpace(s)
	if at := strings.LastIndex(s, "@"); at >= 0 {
		sshUser = s[:at]
		s = s[at+1:]
	}
	if colon := strings.LastIndex(s, ":"); colon >= 0 {
		if p, err := strconv.Atoi(s[colon+1:]); err == nil {
			port = p
			s = s[:colon]
		}
	}
	hostname = s
	return sshUser, hostname, port
}

func currentUsername() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return ""
}
