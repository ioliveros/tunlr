package service

import (
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

func (s *TunnelService) DeleteForward(id uint) error { return s.hosts.DeleteForward(id) }
