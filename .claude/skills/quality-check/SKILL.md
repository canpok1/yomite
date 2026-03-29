---
name: quality-check
description: コミットやPR作成後に設計・実装パターンをレビューしてフィードバックを提供するスキル。linterでチェックできない設計・実装パターン（インターフェース設計、エラーハンドリング、テスト戦略等）をレビューする。
allowed-tools: Bash, Read, Grep, Glob
user-invocable: true
argument-hint: "[対象コミットまたはPR番号]"
---

品質チェックを実施します。

## 対象範囲

linterでチェック不可能な項目に特化する：
- インターフェース設計
- エラーハンドリング
- テスト戦略

## スキップ対象

- コミットメッセージのフォーマット
- linterで適用可能な形式チェック（gofmt、golangci-lint等）

## 実行手順

1. **変更内容確認**: `git diff` や `git show` で変更内容を取得
2. **自動チェック実行**: 該当ファイル形式に応じてlinterを実行
   - Go: `golangci-lint run`
   - Shell: `shellcheck`
   - Markdown: リンク検証
3. **テスト実行**: `go test -v -race -coverprofile=coverage.out ./...`
4. **基準照合**: `references/quality-checklist.md` の基準と照合
5. **フィードバック生成**: 改善点と良い点を含むフィードバックを作成

## 使用方法

```
/quality-check HEAD          # 最新コミット
/quality-check abc123        # 特定コミット
/quality-check #123          # PR番号
```

## フィードバック形式

- ✅ 良い点
- ❌ 改善が必要な点
- 💡 提案

## 参考資料

- [品質チェックリスト](references/quality-checklist.md)
