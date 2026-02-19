// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

// Package program provides program self identification.
package program

import (
	"os"
	"path/filepath"
)

var (
	// Version is the version of the current program, as set by the
	// builder (linker)
	Version = `*unknown*` // overridden by the linker

	// Name is the basename of the current process
	Name = filepath.Base(os.Args[0])

	// Path is the absolute path of the running executable.
	Path = func() string {
		p, err := filepath.Abs(os.Args[0])
		if err != nil {
			panic(err)
		}

		return p
	}()
)
