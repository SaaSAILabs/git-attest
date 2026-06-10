//go:build linux

package util

import (
	"os"
	"syscall"
	"time"
)

func GetFileTimestamps(info os.FileInfo) FileTimestamps {
	stat := info.Sys().(*syscall.Stat_t)
	return FileTimestamps{
		Mtime: time.Unix(stat.Mtim.Sec, stat.Mtim.Nsec),
		Ctime: time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec),
		Btime: time.Time{}, // birthtime unavailable on Linux
	}
}
