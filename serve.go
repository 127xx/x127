package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/127xx/x127/internal/config"
	"github.com/127xx/x127/internal/daemon"
	"github.com/127xx/x127/internal/ports"
	"github.com/127xx/x127/internal/server"
)

const (
	listenAddr = "127.0.0.1:12700"
	baseURL    = "http://127.0.0.1:12700"
)

// cmdServe は serve サブコマンドの入口。X127_DAEMON=1 なら子(サーバー本体)、
// そうでなければ親(デタッチ起動)として振る舞う。
func cmdServe(stdout, stderr io.Writer) int {
	if os.Getenv("X127_DAEMON") == "1" {
		return runServer(stderr)
	}
	return spawnDaemon(stdout, stderr)
}

// spawnDaemon は親側。事前チェックの後、自プロセスを background で再 exec し、
// 子が health になるまで待ってからプロンプトを返す。
func spawnDaemon(stdout, stderr io.Writer) int {
	if _, err := config.EnsureDir(); err != nil {
		fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	pidPath, _ := config.PIDPath()
	if pid, ok := daemon.ReadPID(pidPath); ok && daemon.Alive(pid) {
		fmt.Fprintf(stderr, "x127 is already running (pid %d): %s\n", pid, baseURL)
		return 1
	}
	if err := probeListen(); err != nil {
		fmt.Fprintf(stderr, "x127: port 12700 is in use%s\n", portHolder())
		return 1
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	logPath, _ := config.LogPath()
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	defer logFile.Close()

	cmd := exec.Command(exe, "serve")
	cmd.Env = append(os.Environ(), "X127_DAEMON=1")
	cmd.Stdout, cmd.Stderr = logFile, logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(stderr, "x127: failed to start daemon: %v\n", err)
		return 1
	}

	if err := waitHealthy(3 * time.Second); err != nil {
		fmt.Fprintf(stderr, "x127: daemon did not become healthy: %v (see %s)\n", err, logPath)
		return 1
	}
	fmt.Fprintf(stdout, "x127 serving at %s (pid %d)\n", baseURL, cmd.Process.Pid)
	return 0
}

// runServer は子側。PID ファイルを持ち、HTTP サーバーを起動して
// SIGTERM/SIGINT で graceful shutdown する。
func runServer(stderr io.Writer) int {
	pidPath, _ := config.PIDPath()
	if err := daemon.WritePID(pidPath); err != nil {
		fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	defer os.Remove(pidPath)

	regPath, _ := config.RegistryPath()
	srv := &http.Server{
		Addr:    listenAddr,
		Handler: server.New(regPath, ports.Scan).Handler(),
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	fmt.Fprintf(stderr, "x127 %s listening on %s\n", version, listenAddr)
	select {
	case <-ctx.Done():
		shutdownCtx, c := context.WithTimeout(context.Background(), 3*time.Second)
		defer c()
		srv.Shutdown(shutdownCtx)
		return 0
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(stderr, "x127: %v\n", err)
			return 1
		}
		return 0
	}
}

// cmdStatus は稼働状態を表示する。プロセスが生きていて health に応答すれば 0。
func cmdStatus(stdout, stderr io.Writer) int {
	pidPath, _ := config.PIDPath()
	pid, ok := daemon.ReadPID(pidPath)
	if !ok || !daemon.Alive(pid) {
		fmt.Fprintln(stdout, "x127 is not running")
		return 1
	}
	if err := checkHealth(); err != nil {
		fmt.Fprintf(stdout, "x127 process exists (pid %d) but API is not responding: %v\n", pid, err)
		return 1
	}
	fmt.Fprintf(stdout, "x127 is running (pid %d): %s\n", pid, baseURL)
	return 0
}

// cmdStop はサーバーを停止する。PID が生きていれば SIGTERM を送って終了を待つ。
func cmdStop(stdout, stderr io.Writer) int {
	pidPath, _ := config.PIDPath()
	pid, ok := daemon.ReadPID(pidPath)
	if !ok {
		fmt.Fprintln(stdout, "x127 is not running")
		return 1
	}
	if !daemon.Alive(pid) {
		os.Remove(pidPath) // 古い PID ファイルを掃除する
		fmt.Fprintln(stdout, "x127 is not running (removed stale pid file)")
		return 1
	}
	if err := daemon.Stop(pid, 5*time.Second); err != nil {
		fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "x127 stopped (pid %d)\n", pid)
	return 0
}

// probeListen は 12700 が bind 可能かを事前確認する。
func probeListen() error {
	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	return l.Close()
}

// portHolder は 12700 を占有しているプロセスを " by <process> (pid N)" 形式で返す。
// スキャナが特定できない場合は "" を返す。
func portHolder() string {
	entries, err := ports.Scan()
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.Port == 12700 && e.Process != "" {
			return fmt.Sprintf(" by %s (pid %d)", e.Process, e.PID)
		}
	}
	return ""
}

// checkHealth は /api/health を叩いて 200 が返るかを確認する。
func checkHealth() error {
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get(baseURL + "/api/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health returned %d", resp.StatusCode)
	}
	return nil
}

// waitHealthy は timeout まで health を繰り返しポーリングし、最後のエラーを返す。
func waitHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if lastErr = checkHealth(); lastErr == nil {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return lastErr
}
