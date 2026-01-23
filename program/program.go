// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package program

import (
	"os"
	"path/filepath"
)

var (
	Version string = `*unknown*` // overridden by the linker
	Name    string = filepath.Base(os.Args[0])
	Path    string = func() string {
		if p, err := filepath.Abs(os.Args[0]); err != nil {
			panic(err)
		} else {
			return p
		}
	}()
)
