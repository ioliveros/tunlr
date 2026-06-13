package seed

import (
	"log"

	"github.com/ioliveros/tunlr/internal/model"
	"github.com/ioliveros/tunlr/internal/repository"
)

// SeedDefaults populates the database with the hosts and forwards from the
// original dev-tunnel.sh, but only when no hosts exist yet (first run).
func SeedDefaults(repo *repository.HostRepository) error {
	count, err := repo.CountHosts()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	for i := range defaultHosts() {
		h := defaultHosts()[i]
		if err := repo.CreateHost(&h); err != nil {
			return err
		}
	}
	log.Println("seeded default tunnel hosts from dev-tunnel.sh")
	return nil
}

// defaultHosts mirrors the SSH forwards defined in dev-tunnel.sh.
func defaultHosts() []model.Host {
	return []model.Host{
		{
			Name:          "GDT (dev)",
			Hostname:      "gdt.truckbase.ai",
			User:          "gcp",
			Port:          22,
			AuthMethod:    model.AuthAgent,
			HostKeyPolicy: model.HostKeyAcceptNew,
			Forwards: []model.Forward{
				{Label: "MongoDB", LocalPort: 27015, RemoteHost: "10.128.0.15", RemotePort: 27017, Enabled: true},
				{Label: "MongoDB (dev)", LocalPort: 27016, RemoteHost: "10.128.0.16", RemotePort: 27017, Enabled: true},
				{Label: "TimescaleDB (dev)", LocalPort: 5437, RemoteHost: "10.196.0.84", RemotePort: 5432, Enabled: true},
				{Label: "Neo4j Bolt (dev)", LocalPort: 7687, RemoteHost: "10.196.0.21", RemotePort: 7687, Enabled: true},
				{Label: "Neo4j HTTP (dev)", LocalPort: 7474, RemoteHost: "10.196.0.21", RemotePort: 7474, Enabled: true},
				{Label: "TMS (dev)", LocalPort: 5422, RemoteHost: "10.219.0.37", RemotePort: 5432, Enabled: true},
			},
		},
		{
			Name:          "GPT (prod)",
			Hostname:      "gpt.truckbase.ai",
			User:          "gcp",
			Port:          22,
			AuthMethod:    model.AuthAgent,
			HostKeyPolicy: model.HostKeyAcceptNew,
			Forwards: []model.Forward{
				{Label: "Truckbase (prod)", LocalPort: 5431, RemoteHost: "10.10.208.240", RemotePort: 5432, Enabled: true},
			},
		},
	}
}
