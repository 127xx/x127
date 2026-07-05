// Package config は x127 の設定ディレクトリとファイルパスを解決する。
// 優先順位: $X127_CONFIG_DIR > $XDG_CONFIG_HOME/x127 > ~/.config/x127。
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func Dir() (string, error) {
	// X127_CONFIG_DIR は明示的な上書き。相対パスを許すと設定パスが各コマンドの
	// CWD に依存してしまい(例: serve と stop を別ディレクトリから実行すると互いの
	// pid ファイルを見失う)、CWD 基準で暗黙に解決するのではなくエラーとして弾く。
	if d := os.Getenv("X127_CONFIG_DIR"); d != "" {
		if !filepath.IsAbs(d) {
			return "", fmt.Errorf("X127_CONFIG_DIR must be an absolute path, got %q", d)
		}
		return d, nil
	}
	// XDG 仕様では相対パスの XDG_CONFIG_HOME は不正で無視する必要があるため、
	// CWD 依存の設定パスを避けるべく ~/.config へフォールスルーする。
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" && filepath.IsAbs(x) {
		return filepath.Join(x, "x127"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "x127"), nil
}

func EnsureDir() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	// 0o700: このディレクトリは registry.json / x127.pid / x127.log を保持するため、
	// 所有ユーザー専用に保つ(他ユーザーが一覧・走査できないようにする)。
	return d, os.MkdirAll(d, 0o700)
}

func RegistryPath() (string, error) { return join("registry.json") }
func PIDPath() (string, error)      { return join("x127.pid") }
func LogPath() (string, error)      { return join("x127.log") }

func join(name string) (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, name), nil
}
