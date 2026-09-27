//go:build !linux

package main

func availableBytes(path string) (uint64, error) {
	return 0, nil
}
