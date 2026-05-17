//go:build !windows

package main

import (
	"fmt"
	"io/fs"
	"os/user"
	"syscall"
)

const detectExecutableByExtension = false

func getFileOwnerGroup(info fs.FileInfo) (string, string) {
	sys := info.Sys()
	stat, ok := sys.(*syscall.Stat_t)
	if !ok {
		return currentUser, currentUser
	}
	uid := fmt.Sprint(stat.Uid)
	gid := fmt.Sprint(stat.Gid)
	if u, err := user.LookupId(uid); err == nil {
		uid = u.Username
	}
	if g, err := user.LookupGroupId(gid); err == nil {
		gid = g.Name
	}
	return uid, gid
}

func getLinkCount(info fs.FileInfo) uint64 {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return uint64(stat.Nlink)
	}
	if info.IsDir() {
		return 2
	}
	return 1
}

func checkExecutable(info fs.FileInfo) bool {
	return info.Mode()&0111 != 0
}
