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
		_, _ = fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	pidPath, err := config.PIDPath()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	if pid, ok := daemon.ReadPID(pidPath); ok && daemon.Alive(pid) && daemon.Owned(pid) {
		_, _ = fmt.Fprintf(stderr, "x127 is already running (pid %d): %s\n", pid, baseURL)
		return 1
	}
	if err := probeListen(); err != nil {
		_, _ = fmt.Fprintf(stderr, "x127: port 12700 is in use%s\n", portHolder())
		return 1
	}

	exe, err := os.Executable()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	logPath, err := config.LogPath()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	defer func() { _ = logFile.Close() }()

	// context.Background() で起動する。親の終了に子を連動させない(デタッチ維持)ため、
	// キャンセル可能な context は渡さない。
	cmd := exec.CommandContext(context.Background(), exe, "serve")
	cmd.Env = append(os.Environ(), "X127_DAEMON=1")
	cmd.Stdout, cmd.Stderr = logFile, logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		_, _ = fmt.Fprintf(stderr, "x127: failed to start daemon: %v\n", err)
		return 1
	}

	if err := waitHealthy(3 * time.Second); err != nil {
		// health にならなかった子を残すとポート/PID を掴んだままになり、
		// 次回の serve が already running / port in use で失敗するため確実に停止する。
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_ = cmd.Wait()
		_, _ = fmt.Fprintf(stderr, "x127: daemon did not become healthy: %v (see %s)\n", err, logPath)
		return 1
	}

	// health は共有エンドポイントのため、同時 serve のレースに負けた親も勝者の health を
	// 見て成功と誤認しうる。PID ファイルの所有者が自分の子かを照合して取り違えを防ぐ。
	if filePid, ok := daemon.ReadPID(pidPath); !ok || filePid != cmd.Process.Pid {
		// 別インスタンスが先に bind した。自分の子は bind 失敗で終了済みだが念のため後始末する。
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_ = cmd.Wait()
		if ok {
			_, _ = fmt.Fprintf(stderr, "x127 is already running (pid %d): %s\n", filePid, baseURL)
		} else {
			_, _ = fmt.Fprintf(stderr, "x127: another instance won the startup race\n")
		}
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "x127 serving at %s (pid %d)\n", baseURL, cmd.Process.Pid)
	return 0
}

// runServer は子側。PID ファイルを持ち、HTTP サーバーを起動して
// SIGTERM/SIGINT で graceful shutdown する。
func runServer(stderr io.Writer) int {
	pidPath, err := config.PIDPath()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	regPath, err := config.RegistryPath()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	// タイムアウトを設定して遅延接続によるリソース枯渇(Slowloris 等)を防ぐ。
	// 127.0.0.1 固定バインドだが安価な堅牢化として明示する。
	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           server.New(regPath, ports.Scan).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 先に bind し、成功した本物のサーバーだけが PID ファイルを所有する。
	// これにより同時 serve で敗者(bind 失敗側)が勝者の PID ファイルを消す競合を防ぐ。
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", listenAddr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	if err := daemon.WritePID(pidPath); err != nil {
		_, _ = fmt.Fprintf(stderr, "x127: %v\n", err)
		_ = ln.Close()
		return 1
	}
	// Shutdown は listener を閉じてから runServer が return するため、その間に別 daemon が
	// 即時再起動して PID を書くことがある。自分が所有者のときだけ削除し、新 daemon の
	// PID ファイルを誤って消さないようにする。
	ownPID := os.Getpid()
	defer func() {
		if pid, ok := daemon.ReadPID(pidPath); ok && pid == ownPID {
			_ = os.Remove(pidPath)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()

	_, _ = fmt.Fprintf(stderr, "x127 %s listening on %s\n", version, listenAddr)
	select {
	case <-ctx.Done():
		shutdownCtx, c := context.WithTimeout(context.Background(), 3*time.Second)
		defer c()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			_, _ = fmt.Fprintf(stderr, "x127: graceful shutdown failed: %v\n", err)
			return 1
		}
		return 0
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			_, _ = fmt.Fprintf(stderr, "x127: %v\n", err)
			return 1
		}
		return 0
	}
}

// cmdStatus は稼働状態を表示する。プロセスが生きていて health に応答すれば 0。
func cmdStatus(stdout, stderr io.Writer) int {
	pidPath, err := config.PIDPath()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	pid, ok := daemon.ReadPID(pidPath)
	if !ok || !daemon.Alive(pid) || !daemon.Owned(pid) {
		_, _ = fmt.Fprintln(stdout, "x127 is not running")
		return 1
	}
	if err := checkHealth(); err != nil {
		_, _ = fmt.Fprintf(stdout, "x127 process exists (pid %d) but API is not responding: %v\n", pid, err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "x127 is running (pid %d): %s\n", pid, baseURL)
	return 0
}

// cmdStop はサーバーを停止する。PID が生きていれば SIGTERM を送って終了を待つ。
func cmdStop(stdout, stderr io.Writer) int {
	pidPath, err := config.PIDPath()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	pid, ok := daemon.ReadPID(pidPath)
	if !ok {
		_, _ = fmt.Fprintln(stdout, "x127 is not running")
		return 1
	}
	if !daemon.Alive(pid) {
		_ = os.Remove(pidPath) // 死んでいる PID ファイルは stale として掃除する
		_, _ = fmt.Fprintln(stdout, "x127 is not running (removed stale pid file)")
		return 1
	}
	if !daemon.Owned(pid) {
		// 生存しているが x127 と確認できない(PID 再利用による別プロセス、または
		// プロセス情報の一時的な取得失敗)。誤って他プロセスへ SIGTERM を送らず、
		// 稼働中の可能性を残して PID ファイルも消さない。
		_, _ = fmt.Fprintf(stderr, "x127: pid %d が x127 か確認できないため停止を中止しました\n", pid)
		return 1
	}
	if err := daemon.Stop(pid, 5*time.Second); err != nil {
		_, _ = fmt.Fprintf(stderr, "x127: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "x127 stopped (pid %d)\n", pid)
	return 0
}

// probeListen は 12700 が bind 可能かを事前確認する。
func probeListen() error {
	var lc net.ListenConfig
	l, err := lc.Listen(context.Background(), "tcp", listenAddr)
	if err != nil {
		return err
	}
	return l.Close()
}

// portHolder は 12700 を占有しているプロセスを " by <process> (pid N)" 形式で返す。
// プロセス名が取れない場合(macOS の他ユーザープロセス等)は pid だけでも
// " (pid N)" を返す。全く特定できない場合のみ "" を返す。
func portHolder() string {
	entries, err := ports.Scan()
	if err != nil {
		return ""
	}
	var pidOnly int32
	for _, e := range entries {
		if e.Port != 12700 {
			continue
		}
		// プロセス名まで取れたエントリを最優先で返す
		if e.Process != "" {
			return fmt.Sprintf(" by %s (pid %d)", e.Process, e.PID)
		}
		// 名前は不明だが pid は取れた場合はフォールバックとして控える
		if pidOnly == 0 && e.PID > 0 {
			pidOnly = e.PID
		}
	}
	if pidOnly > 0 {
		return fmt.Sprintf(" (pid %d)", pidOnly)
	}
	return ""
}

// checkHealth は /api/health を叩いて 200 が返るかを確認する。
func checkHealth() error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/api/health", nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
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
