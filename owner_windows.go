//go:build windows

package main

import "io/fs"

func getFileOwnerGroup(info fs.FileInfo) (string, string) {
	return currentUser, currentUser
}
