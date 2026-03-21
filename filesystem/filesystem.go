package filesystem

import "github.com/spf13/afero"

var backend = afero.Afero{Fs: afero.NewOsFs()}

func API() afero.Afero {
	return backend
}

func SetOsFs() {
	backend = afero.Afero{Fs: afero.NewOsFs()}
}

func SetMemMapFs() {
	backend = afero.Afero{Fs: afero.NewMemMapFs()}
}
