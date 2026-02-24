// Package filesystem provides a virtualized abstraction layer for all filesystem operations.
package filesystem

import (
	"io"
	"os"
)

// GacheFs adapts the afero filesystem to the gache.FileSystem interface.
// This allows the gache library to use our swappable filesystem backend.
type GacheFs struct{}

// OpenFile opens a file using the current filesystem backend.
func (GacheFs) OpenFile(name string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	return API().OpenFile(name, flag, perm)
}

// MkdirAll creates a directory using the current filesystem backend.
func (GacheFs) MkdirAll(path string, perm os.FileMode) error {
	return API().MkdirAll(path, perm)
}
