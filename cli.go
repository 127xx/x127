package main

import (
	"fmt"
	"io"

	"github.com/127xx/x127/internal/version"
)

const usage = `usage: x127 <command>

commands:
  serve    バックグラウンドでサーバーを起動 (http://127.0.0.1:12700)
  status   稼働状態を表示
  stop     サーバーを停止
  version  バージョンを表示
`

// Run は x127 CLI を実行する。args は os.Args[1:]。以降のタスクで
// 下の switch にサブコマンドの case を追加していく。
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		// CLI 表示の書き込み失敗は対処できないためエラーを明示的に無視する
		_, _ = fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "version", "--version":
		_, _ = fmt.Fprintf(stdout, "x127 %s\n", version.Version)
		return 0
	case "serve":
		return cmdServe(stdout, stderr)
	case "status":
		return cmdStatus(stdout, stderr)
	case "stop":
		return cmdStop(stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "x127: unknown command %q\n\n", args[0])
		_, _ = fmt.Fprint(stderr, usage)
		return 2
	}
}
