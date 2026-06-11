//go:build windows

package util

import (
	"os"
	"syscall"
	"time"
)

func GetFileTimestamps(info os.FileInfo) FileTimestamps {
	stat := info.Sys().(*syscall.Win32FileAttributeData)
	return FileTimestamps{
		Mtime: time.Unix(0, stat.LastWriteTime.Nanoseconds()),
		Ctime: time.Unix(0, stat.LastWriteTime.Nanoseconds()), // Windows has no true ctime, mimic mtime to avoid false drift alerts
		Btime: time.Unix(0, stat.CreationTime.Nanoseconds()),
	}
}
