//go:build !darwin && !windows && !linux

package updater

import "errors"

func supported() bool { return false }

func apply(string) (func() error, error) {
	return nil, errors.New("không hỗ trợ tự cập nhật trên hệ điều hành này")
}

func Cleanup() {}
