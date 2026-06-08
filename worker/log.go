package worker

import (
	"fmt"
	"os"
)

func (cfg Config) logf(format string, args ...any) {
	if cfg.Quiet && !cfg.Verbose {
		return
	}
	fmt.Fprintf(os.Stderr, format, args...)
}

func (cfg Config) eventf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}