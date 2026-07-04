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

// Run executes the x127 CLI. args is os.Args[1:]; later tasks add
// subcommand cases to the switch below.
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
