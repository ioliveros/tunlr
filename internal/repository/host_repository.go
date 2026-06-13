package repository

import (
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
func (r *HostRepository) UpdateHost(host *model.Host) error {
	return r.db.Model(host).Omit("Forwards").Updates(host).Error
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
