//go:build !windows

package engine

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func identity(fi os.FileInfo, path string) string {
	s := fi.Sys().(*syscall.Stat_t)
	return fmt.Sprintf("%s:%d:%d", path, s.Dev, s.Ino)
}
func replace(from, to string) error { return os.Rename(from, to) }
func lockFile(f *os.File) error     { return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) }
func unlockFile(f *os.File)         { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }

func boundProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 2 * time.Second
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if err == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return err
	}
}
