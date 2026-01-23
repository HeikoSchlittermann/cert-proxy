// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package secret

import (
	"io/ioutil"
	"os"
	"strings"
)

func Read(src string) (string, error) {
	var proto, value = func() (string, string) {
		x := strings.SplitN(src, `:`, 2)
		return x[0], x[1]
	}()

	switch strings.ToUpper(proto) {
	case `PASS`:
		return value, nil
	case `FILE`:
		if b, err := ioutil.ReadFile(value); err != nil {
			return ``, err
		} else {
			return strings.TrimRight(string(b), "\r\n \t"), nil
		}
	case `ENV`:
		return os.Getenv(value), nil
	default:
		panic("unhandled secret source proto: " + proto)
	}

	return ``, errors.New("xxx")
}
