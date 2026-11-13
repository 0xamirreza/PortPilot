package sockets

import "os"

func openProc(path string) (*os.File, error) {
	return os.Open(path)
}
