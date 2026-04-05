# yomite

## 開発

### GUI の動作確認

```bash
make run-gui
```

起動後、ブラウザで http://localhost:34115/ にアクセスする。

設定ファイルを明示的に指定する場合:

```bash
make run-gui options="--config /path/to/yomite.json"
```