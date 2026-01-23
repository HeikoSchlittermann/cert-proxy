// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package shared

import "os"

func Mkdir(dir string) error {
	err := os.Mkdir(dir, 0777)

	if err != nil && os.IsExist(err) {
		stat, err := os.Stat(dir)
		if err != nil {
			return err
		}

		if stat.IsDir() {
			return nil
		}
	}

	return err
}
