//go:build !darwin && !windows

package capture

import "errors"

func Trusted(bool) bool { return true }
func Run(Handler) error {
	return errors.New("server mode (capturing input) is only implemented on macOS for now")
}
func Freeze(bool)           {}
func Warp(float64, float64) {}
func ScreenBounds() Bounds  { return Bounds{} }
