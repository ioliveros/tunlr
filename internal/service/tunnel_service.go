package service

import (
	"fmt"
	"os/user"
	"strconv"
	"strings"

	"github.com/ioliveros/tunlr/internal/dto"
	"github.com/ioliveros/tunlr/internal/model"
	"github.com/ioliveros/tunlr/internal/repository"
)

// TunnelService coordinates persistence of tunnel configuration. In a later
// iteration it also drives the SSH tunnel engine (connect/disconnect/status).
type TunnelService struct {
	hosts *repository.HostRepository
}

func NewTunnelService(hosts *repository.HostRepository) *TunnelService {
	return &TunnelService{hosts: hosts}
}

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

func (s *TunnelService) DeleteHost(id uint) error { return s.hosts.DeleteHost(id) }

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
		return s.hosts.DeleteHost(fwd.HostID)
	}
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

	host, err := s.hosts.FindHost(sshUser, hostname, port)
	if err != nil {
		return nil, err
	}
	if host == nil {
		host = &model.Host{
			Name:          hostname,
			Hostname:      hostname,
			User:          sshUser,
			Port:          port,
			AuthMethod:    model.AuthAgent,
			HostKeyPolicy: model.HostKeyAcceptNew,
		}
		if err := s.hosts.CreateHost(host); err != nil {
			return nil, err
		}
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
	return s.hosts.GetHost(host.ID)
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
