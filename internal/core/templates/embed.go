package templates

import "embed"

// FS はテンプレートファイルを埋め込んだファイルシステム。
//
//go:embed *.tmpl
var FS embed.FS
