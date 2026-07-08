package main

import (
	_ "embed"
	"strings"
)

var versionFile string

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func init() {
	if Version == "dev" {
		if v := strings.TrimSpace(versionFile); v != "" {
			Version = v
		}
	}
}
