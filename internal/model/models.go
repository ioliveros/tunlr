package model

import (
	"time"

	"gorm.io/gorm"
)

// AuthMethod identifies how the SSH client authenticates to a host.
type AuthMethod string

const (
	// AuthAgent uses the running ssh-agent (SSH_AUTH_SOCK).
	AuthAgent AuthMethod = "agent"
	// AuthKey uses a private key file at Host.KeyPath.
	AuthKey AuthMethod = "key"
)

// HostKeyPolicy controls how the server's host key is verified.
type HostKeyPolicy string

const (
	// HostKeyStrict requires the key to already be in known_hosts.
	HostKeyStrict HostKeyPolicy = "strict"
	// HostKeyAcceptNew trusts unknown hosts on first connect (TOFU) but
	// still rejects a changed key for a known host.
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
