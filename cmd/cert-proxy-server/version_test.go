package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.schlittermann.de/heiko/cert-proxy/internal/program"
)

func TestVersionLine(t *testing.T) {
	origName, origVersion, origPath := program.Name, program.Version, program.Path
	t.Cleanup(func() { program.Name, program.Version, program.Path = origName, origVersion, origPath })

	program.Name = "cert-proxy-server"
	program.Version = "v1.2.3"
	program.Path = "/usr/bin/cert-proxy-server"

	got := versionLine()
	assert.Equal(t, []string{"cert-proxy-server", "v1.2.3", "/usr/bin/cert-proxy-server"}, strings.Fields(got))
}
