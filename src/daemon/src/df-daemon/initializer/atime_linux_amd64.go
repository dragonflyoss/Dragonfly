package initializer

import "syscall"

// Atime returns the last access time in seconds
func Atime(stat *syscall.Stat_t) int64 {
	return stat.Atim.Sec
}
