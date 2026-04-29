// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package shared //nolint:revive // I know, it is stupid

import (
	"context"
	"fmt"
	"log/slog"
)

// Verbose emits a printf-formatted message at slog.LevelDebug on the
// process-wide slog.Default(). When the default logger is below debug
// level (the silent default) the call returns without formatting the
// arguments. Configure visibility by replacing slog.Default() with a
// handler whose Level allows Debug.
func Verbose(format string, args ...any) {
	logger := slog.Default()
	if !logger.Enabled(context.Background(), slog.LevelDebug) {
		return
	}

	logger.Debug(fmt.Sprintf(format, args...))
}
