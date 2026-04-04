---
name: review
description: コード品質レビューとドキュメント整合性チェックを統合的に実施するスキル。readability-reviewer、quality-reviewer、doc-reviewer サブエージェントで並列レビューし、結果に基づいて修正する。
user-invocable: true
---

セルフレビューを実施します。

## 手順

### 1. 並列レビュー

以下のサブエージェントを **並列** で呼び出す：

- `readability-reviewer`: コードの可読性レビュー（命名、関数設計、制御フロー、複雑さ等）
- `quality-reviewer`: 設計・実装パターンレビュー（インターフェース設計、エラーハンドリング、テスト戦略）
- `doc-reviewer`: ドキュメント整合性チェックとMarkdownリンク検証

### 2. レビュー結果に基づく修正

各サブエージェントの報告を確認し、必要な修正を行う。

### 3. コミット

修正を行った場合はコミットする。
