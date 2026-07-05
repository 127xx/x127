// Package web はビルド済み Svelte SPA(web/dist)をバイナリに埋め込む。
package web

import "embed"

//go:embed all:dist
var Dist embed.FS
