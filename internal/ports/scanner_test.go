package ports

import (
	"net"
	"os"
	"testing"
)

func TestScanFindsOwnListener(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port

	entries, err := Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	for _, e := range entries {
		if e.Port == port && e.PID == int32(os.Getpid()) {
			if e.Proto != "tcp" {
				t.Fatalf("Proto = %q, want tcp", e.Proto)
			}
			if e.Process == "" {
				t.Fatal("Process is empty for own listener")
			}
			return
		}
	}
	t.Fatalf("own listener on port %d not found in %d entries", port, len(entries))
}

func TestScanSortedByPort(t *testing.T) {
	entries, err := Scan()
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Port > entries[i].Port {
			t.Fatalf("entries not sorted at index %d", i)
		}
	}
}
