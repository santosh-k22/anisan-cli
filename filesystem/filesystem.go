// Package filesystem provides a virtualized abstraction layer for all filesystem operations.
//
// It utilizes the afero library to allow seamless switching between OS-level and in-memory filesystem backends.
package filesystem

import "github.com/spf13/afero"

var backend = afero.Afero{Fs: afero.NewOsFs()}

// API returns the active afero.Afero instance for filesystem interaction.
func API() afero.Afero {
	return backend
}

// SetOsFs restores the filesystem backend to the native operating system implementation.
func SetOsFs() {
	backend = afero.Afero{Fs: afero.NewOsFs()}
}

// SetMemMapFs initializes a volatile in-memory filesystem backend for unit testing and CI environments.
func SetMemMapFs() {
	backend = afero.Afero{Fs: afero.NewMemMapFs()}
}
