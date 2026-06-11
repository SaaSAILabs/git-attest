//go:build darwin

package util

import (
	"os"
	"syscall"
	"time"
)


func GetFileTimestamps(info os.FileInfo) FileTimestamps {
	stat := info.Sys().(*syscall.Stat_t)
	return FileTimestamps{
		Mtime: time.Unix(stat.Mtimespec.Sec, stat.Mtimespec.Nsec),
		Ctime: time.Unix(stat.Ctimespec.Sec, stat.Ctimespec.Nsec),
		Btime: time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec),
	}
}
