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

func checkExecutable(info fs.FileInfo) bool {
	return false
}
