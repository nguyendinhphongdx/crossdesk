//go:build !darwin && !windows && !linux

package clipboard

import "errors"

func read() (Content, bool) { return Content{}, false }
func write(Content) error   { return errors.New("clipboard not supported") }
func problem() string       { return "Hệ điều hành này chưa hỗ trợ chia sẻ clipboard" }
