package updater

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func supported() bool { return true }

// apply replaces the running Crossdesk.app with the one in the zip.
func apply(archive string) (func() error, error) {
	exe, err := executable()
	if err != nil {
		return nil, err
	}
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe))) // .../Crossdesk.app/Contents/MacOS/crossdesk
	if !strings.HasSuffix(bundle, ".app") {
		return nil, errors.New("Crossdesk không chạy từ một ứng dụng .app — hãy tải bản mới thủ công")
	}
	if strings.Contains(bundle, "/AppTranslocation/") {
		return nil, errors.New("hãy kéo Crossdesk vào thư mục Applications rồi mở lại, sau đó mới cập nhật được")
	}
	parent := filepath.Dir(bundle)
	tmp, err := os.MkdirTemp(parent, ".crossdesk-update-")
	if err != nil {
		return nil, fmt.Errorf("%w vào %s", errNotWritable, parent)
	}
	defer os.RemoveAll(tmp)
	if err := unzipTo(archive, tmp); err != nil {
		return nil, err
	}
	fresh := filepath.Join(tmp, "Crossdesk.app")
	if _, err := os.Stat(filepath.Join(fresh, "Contents", "MacOS", "crossdesk")); err != nil {
		return nil, errors.New("gói cập nhật không đúng định dạng")
	}
	old := bundle + ".old"
	os.RemoveAll(old)
	if err := os.Rename(bundle, old); err != nil {
		return nil, err
	}
	if err := os.Rename(fresh, bundle); err != nil {
		os.Rename(old, bundle) // roll back
		return nil, err
	}
	os.RemoveAll(old) // the running binary stays mapped; deleting it is fine
	return func() error {
		cmd := exec.Command(filepath.Join(bundle, "Contents", "MacOS", "crossdesk"))
		cmd.Env = restartEnv()
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		return cmd.Start()
	}, nil
}

// Cleanup removes leftovers of a previous update.
func Cleanup() {
	if exe, err := executable(); err == nil {
		bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
		if strings.HasSuffix(bundle, ".app") {
			os.RemoveAll(bundle + ".old")
		}
	}
}
