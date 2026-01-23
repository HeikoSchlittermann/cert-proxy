// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package shared

import "log"

func Check(err error) {
	if err == nil {
		return
	}

	log.Fatal(err)
}
