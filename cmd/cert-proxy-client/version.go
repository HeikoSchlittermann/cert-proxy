package main

import "go.schlittermann.de/heiko/cert-proxy/internal/program"

func versionLine() string {
	return program.Name + " " + program.Version + " " + program.Path
}
