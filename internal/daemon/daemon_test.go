package daemon

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteAndReadPID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x127.pid")
	if err := WritePID(path); err != nil {
		t.Fatal(err)
	}
	pid, ok := ReadPID(path)
	if !ok || pid != os.Getpid() {
		t.Fatalf("ReadPID = %d, %v; want %d, true", pid, ok, os.Getpid())
	}
}

func TestReadPIDMissingFile(t *testing.T) {
	if _, ok := ReadPID(filepath.Join(t.TempDir(), "none.pid")); ok {
		t.Fatal("ReadPID on missing file returned ok")
	}
}

func TestAlive(t *testing.T) {
	if !Alive(os.Getpid()) {
		t.Fatal("Alive(self) = false")
	}
	if Alive(99999999) {
		t.Fatal("Alive(bogus) = true")
	}
}

func TestOwned(t *testing.T) {
	// 自プロセス(テストバイナリ)は os.Executable() と一致するため owned とみなされる
	if !Owned(os.Getpid()) {
		t.Fatal("Owned(self) = false, want true")
	}
	// 別名のプロセス(sleep)は x127 ではないので false
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	if Owned(cmd.Process.Pid) {
		t.Fatalf("Owned(sleep pid %d) = true, want false", cmd.Process.Pid)
	}
	// 存在しない PID も false
	if Owned(99999999) {
		t.Fatal("Owned(bogus) = true, want false")
	}
}

func TestExeBase(t *testing.T) {
	cases := map[string]string{
		"/usr/local/bin/x127":           "x127",
		"/usr/local/bin/x127 (deleted)": "x127", // 稼働中に削除・置換されたケース
		"x127":                          "x127",
	}
	for in, want := range cases {
		if got := exeBase(in); got != want {
			t.Errorf("exeBase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStop(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { _ = cmd.Wait() }() // ゾンビ化させないよう子プロセスを reap する
	if err := Stop(cmd.Process.Pid, 5*time.Second); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if Alive(cmd.Process.Pid) {
		t.Fatal("process still alive after Stop")
	}
}
