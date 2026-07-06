// Package daemon はデタッチ起動した x127 サーバーの PID ファイル管理と
// プロセスのライフサイクル制御を担う。Unix 系(macOS/Linux)専用。
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v4/process"
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

// Owned は pid が x127 自身のプロセスかを実行ファイル名で照合して判定する。
// PID の再利用で別プロセスを x127 と誤認し、稼働中扱いや SIGTERM 送信をしないための確認に使う。
// プロセス情報が取得できない(macOS の他ユーザープロセス等)場合は安全側に倒して false を返す。
func Owned(pid int) bool {
	self, err := os.Executable()
	if err != nil {
		return false
	}
	want := filepath.Base(self)

	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return false
	}
	// 実行ファイルのフルパスが取れれば basename で照合する
	if exe, err := p.Exe(); err == nil && exe != "" {
		return filepath.Base(exe) == want
	}
	// フルパスが取れない環境では実行ファイル名で照合する
	if name, err := p.Name(); err == nil && name != "" {
		return name == want
	}
	return false
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
