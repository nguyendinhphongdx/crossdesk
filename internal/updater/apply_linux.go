package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func supported() bool { return true }

// apply replaces the binary in place (rename over a running file is fine).
func apply(archive string) (func() error, error) {
	exe, err := executable()
	if err != nil {
		return nil, err
	}
	fresh := exe + ".new"
	if err := extractFromTarGz(archive, "crossdesk", fresh); err != nil {
		if os.IsPermission(err) {
			return nil, fmt.Errorf("%w vào %s — tải gói mới và chạy lại sudo ./install.sh", errNotWritable, filepath.Dir(exe))
		}
		return nil, err
	}
	if err := os.Rename(fresh, exe); err != nil {
		os.Remove(fresh)
		return nil, err
	}
	return func() error {
		cmd := exec.Command(exe)
		cmd.Env = restartEnv()
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		return cmd.Start()
	}, nil
}

func Cleanup() {
	if exe, err := executable(); err == nil {
		os.Remove(exe + ".new")
	}
}
