package main

import (
	"fmt"
	"io"
)

const version = "v0.1.0-dev"

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
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "version", "--version":
		fmt.Fprintf(stdout, "x127 %s\n", version)
		return 0
	default:
		fmt.Fprintf(stderr, "x127: unknown command %q\n\n", args[0])
		fmt.Fprint(stderr, usage)
		return 2
	}
}
