package updater

import (
	"io"
	"os"
	"os/exec"
)

func supported() bool { return true }

// apply swaps the running .exe: Windows lets a running executable be
// renamed (not deleted), so the old one becomes .old and is removed on the
// next start.
func apply(downloaded string) (func() error, error) {
	exe, err := executable()
	if err != nil {
		return nil, err
	}
	fresh := exe + ".new"
	if err := copyFile(downloaded, fresh); err != nil {
		return nil, err
	}
	old := exe + ".old"
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		os.Remove(fresh)
		return nil, err
	}
	if err := os.Rename(fresh, exe); err != nil {
		os.Rename(old, exe)
		return nil, err
	}
	return func() error {
		cmd := exec.Command(exe)
		cmd.Env = restartEnv()
		return cmd.Start()
	}, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func Cleanup() {
	if exe, err := executable(); err == nil {
		os.Remove(exe + ".old")
		os.Remove(exe + ".new")
	}
}
