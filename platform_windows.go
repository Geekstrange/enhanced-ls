//go:build windows

package main

import "io/fs"

const detectExecutableByExtension = true

func getFileOwnerGroup(info fs.FileInfo) (string, string) {
	return currentUser, currentUser
}

func getLinkCount(info fs.FileInfo) uint64 {
	if info.IsDir() {
		return 2
	}
	return 1
}

// getBlockCount approximates on-disk usage in 1K blocks; Windows stat does
// not expose block counts, so ceil(size/1024) is the best available value.
func getBlockCount(info fs.FileInfo) uint64 {
	return (uint64(info.Size()) + 1023) / 1024
}

func checkExecutable(info fs.FileInfo) bool {
	return false
}
