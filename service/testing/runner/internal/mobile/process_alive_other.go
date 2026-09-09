//go:build !darwin && !linux && !windows

package mobile

import "os"

func processAlive(pid int) bool {
	return pid == os.Getpid()
}
