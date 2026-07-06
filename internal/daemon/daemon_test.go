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
