package repo

import (
	"errors"
	"os/exec"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/pkg/restic"
)

func niceAvailable() bool {
	_, err := exec.LookPath("nice")
	return err == nil
}

func ioniceAvailable() bool {
	_, err := exec.LookPath("ionice")
	return err == nil
}

// resolveCommandPrefix returns a list of restic.GenericOption that should be applied to a restic command based on the given prefix.
func resolveCommandPrefix(prefix *v1.CommandPrefix) ([]restic.GenericOption, error) {
	var opts []restic.GenericOption

	if prefix.GetCpuNice() != v1.CommandPrefix_CPU_DEFAULT {
		if !niceAvailable() {
			return nil, errors.New("nice not available, cpu_nice cannot be used")
		} else {
			switch prefix.GetCpuNice() {
			case v1.CommandPrefix_CPU_HIGH:
				opts = append(opts, restic.WithPrefixCommand("nice", "-n", "10"))
			case v1.CommandPrefix_CPU_LOW:
				opts = append(opts, restic.WithPrefixCommand("nice", "-n", "-10"))
			}
		}
	}

	if prefix.GetIoNice() != v1.CommandPrefix_IO_DEFAULT {
		if !ioniceAvailable() {
			return nil, errors.New("ionice not available, io_nice cannot be used")
		}
		switch prefix.GetIoNice() {
		case v1.CommandPrefix_IO_IDLE:
			opts = append(opts, restic.WithPrefixCommand("ionice", "-c", "3")) // idle priority, only runs when other IO is not queued.
		case v1.CommandPrefix_IO_BEST_EFFORT_LOW:
			opts = append(opts, restic.WithPrefixCommand("ionice", "-c", "2", "-n", "7")) // best effort, low priority. Default is -n 4.
		case v1.CommandPrefix_IO_BEST_EFFORT_HIGH:
			opts = append(opts, restic.WithPrefixCommand("ionice", "-c", "2", "-n", "0")) // best effort, high(er) than default priority. Default is -n 4.
		}
	}

	return opts, nil
}
