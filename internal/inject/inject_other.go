//go:build !windows && !darwin

package inject

import "errors"

func New() (Injector, error) {
	return nil, errors.New("client mode is only implemented on Windows and macOS")
}
