package main

import (
	"fmt"
	"os"

	yomite "github.com/canpok1/yomite/cmd/yomite"
)

func printUsage() {
	_, _ = fmt.Fprintln(os.Stderr, "使い方: yomite <command> [options]")
	_, _ = fmt.Fprintln(os.Stderr, "")
	_, _ = fmt.Fprintln(os.Stderr, "コマンド:")
	_, _ = fmt.Fprintln(os.Stderr, "  run    シミュレーションを実行する")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		os.Exit(yomite.Run(os.Args[2:], os.Stdout, os.Stderr))
	default:
		_, _ = fmt.Fprintf(os.Stderr, "エラー: 不明なコマンド %q\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}
