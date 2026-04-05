# yomite

## 開発

### CUI の動作確認

```bash
make run options="-f input.txt"
```

主なオプション:

- `-f` — 入力テキストファイルのパス（必須）
- `--config` — 設定ファイルのパスを明示指定
- `--provider` — プロバイダID指定
- `--persona` — ペルソナID指定
- `--json` — 出力をJSON形式に切替

### GUI の動作確認

```bash
make run-gui
```

起動後、ブラウザで http://localhost:34115/ にアクセスする。

設定ファイルを明示的に指定する場合:

```bash
make run-gui options="--config /path/to/yomite.json"
```