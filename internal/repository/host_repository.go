package repository

import (
	"errors"

	"github.com/ioliveros/tunlr/internal/model"
	"gorm.io/gorm"
)

// HostRepository provides CRUD access to hosts and their forwards.
type HostRepository struct {
	db *gorm.DB
}

func NewHostRepository(db *gorm.DB) *HostRepository {
	return &HostRepository{db: db}
}

// ListHosts returns all hosts with their forwards eagerly loaded.
func (r *HostRepository) ListHosts() ([]model.Host, error) {
	var hosts []model.Host
	err := r.db.Preload("Forwards").Order("name asc").Find(&hosts).Error
	return hosts, err
}

// GetHost returns a single host with its forwards.
func (r *HostRepository) GetHost(id uint) (*model.Host, error) {
	var host model.Host
	if err := r.db.Preload("Forwards").First(&host, id).Error; err != nil {
		return nil, err
	}
	return &host, nil
}

// CreateHost inserts a host (and any forwards attached to it).
func (r *HostRepository) CreateHost(host *model.Host) error {
	return r.db.Create(host).Error
}

// UpdateHost saves changes to a host's own fields (not its forwards).
// Uses a map so zero-value fields (e.g. empty KeyPath) are written.
func (r *HostRepository) UpdateHost(host *model.Host) error {
	return r.db.Model(host).Updates(map[string]any{
		"name":            host.Name,
		"hostname":        host.Hostname,
		"user":            host.User,
		"port":            host.Port,
		"auth_method":     string(host.AuthMethod),
		"key_path":        host.KeyPath,
		"host_key_policy": string(host.HostKeyPolicy),
	}).Error
}

// DeleteHost removes a host and cascades to its forwards.
func (r *HostRepository) DeleteHost(id uint) error {
	return r.db.Select("Forwards").Delete(&model.Host{ID: id}).Error
}

// CountHosts returns the number of hosts, used to decide first-run seeding.
func (r *HostRepository) CountHosts() (int64, error) {
	var n int64
	err := r.db.Model(&model.Host{}).Count(&n).Error
	return n, err
}

// FindHost looks up a host by its SSH identity (user@hostname:port).
// It returns (nil, nil) when no matching host exists.
func (r *HostRepository) FindHost(user, hostname string, port int) (*model.Host, error) {
	var host model.Host
	err := r.db.Where("user = ? AND hostname = ? AND port = ?", user, hostname, port).First(&host).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// GetForward returns a single forward by id.
func (r *HostRepository) GetForward(id uint) (*model.Forward, error) {
	var f model.Forward
	if err := r.db.First(&f, id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

// CountForwardsForHost returns how many forwards belong to a host.
func (r *HostRepository) CountForwardsForHost(hostID uint) (int64, error) {
	var n int64
	err := r.db.Model(&model.Forward{}).Where("host_id = ?", hostID).Count(&n).Error
	return n, err
}

// CreateForward inserts a forward for an existing host.
func (r *HostRepository) CreateForward(f *model.Forward) error {
	return r.db.Create(f).Error
}

// UpdateForward saves changes to a forward.
func (r *HostRepository) UpdateForward(f *model.Forward) error {
	return r.db.Model(f).Updates(map[string]any{
		"label":       f.Label,
		"local_port":  f.LocalPort,
		"remote_host": f.RemoteHost,
		"remote_port": f.RemotePort,
		"enabled":     f.Enabled,
	}).Error
}

// DeleteForward removes a single forward.
func (r *HostRepository) DeleteForward(id uint) error {
	return r.db.Delete(&model.Forward{}, id).Error
}
