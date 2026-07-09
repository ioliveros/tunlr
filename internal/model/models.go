package model

import (
	"time"

	"gorm.io/gorm"
)

type AuthMethod string

const (
	AuthAgent AuthMethod = "agent"
	AuthKey AuthMethod = "key"
)

type HostKeyPolicy string

const (
	HostKeyStrict HostKeyPolicy = "strict"
	HostKeyAcceptNew HostKeyPolicy = "accept-new"
)

// Host is an SSH bastion the app dials to open one or more port forwards.
type Host struct {
	ID            uint          `gorm:"primaryKey" json:"id"`
	Name          string        `gorm:"not null" json:"name"`
	Hostname      string        `gorm:"not null" json:"hostname"`
	User          string        `gorm:"not null" json:"user"`
	Port          int           `gorm:"not null;default:22" json:"port"`
	AuthMethod    AuthMethod    `gorm:"not null;default:agent" json:"authMethod"`
	KeyPath       string        `json:"keyPath"`
	HostKeyPolicy HostKeyPolicy `gorm:"not null;default:accept-new" json:"hostKeyPolicy"`
	Forwards      []Forward     `gorm:"constraint:OnDelete:CASCADE;" json:"forwards"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

// Forward is a local-to-remote TCP port forward carried over a Host's
// SSH connection (equivalent to `ssh -L LocalPort:RemoteHost:RemotePort`).
type Forward struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	HostID     uint      `gorm:"not null;index" json:"hostId"`
	Label      string    `json:"label"`
	LocalPort  int       `gorm:"not null" json:"localPort"`
	RemoteHost string    `gorm:"not null" json:"remoteHost"`
	RemotePort int       `gorm:"not null" json:"remotePort"`
	Enabled    bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// Migrate creates or updates the schema for all models.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Host{}, &Forward{})
}
