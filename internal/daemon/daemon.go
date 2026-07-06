// Package daemon はデタッチ起動した x127 サーバーの PID ファイル管理と
// プロセスのライフサイクル制御を担う。Unix 系(macOS/Linux)専用。
package daemon

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// WritePID は自プロセスの PID を path に書き込む。
func WritePID(path string) error {
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o644)
}

// ReadPID は path から PID を読み取る。ファイルが無い・内容が不正なら ok=false を返す。
func ReadPID(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

// Alive は signal 0 を送ってプロセスの生存を確認する。
func Alive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// Stop は pid に SIGTERM を送り、timeout まで終了を待つ。
func Stop(pid int, timeout time.Duration) error {
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to signal pid %d: %w", pid, err)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !Alive(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("pid %d did not exit within %s", pid, timeout)
}
