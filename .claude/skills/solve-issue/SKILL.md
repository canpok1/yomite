---
name: solve-issue
description: GitHub Issueの理解から実装、PR作成、マージまでを一貫して実行するスキル
allowed-tools: Skill, Agent, Bash, Read, Grep, Glob, Write, Edit
user-invocable: true
argument-hint: "<issue_number>"
---

GitHub Issueを解決します。
各ステップが完了したら、ユーザーの指示を待たずに次のステップへ自動的に進むこと。

## ワークフロー

### Step 1: Issue状態の確認

Issueがオープンであることを確認する。CLOSEDの場合は終了する。

### Step 2: Issueの理解

1. `/vox-actor-plugin:monologue` で作業開始を宣言する
2. Issueの内容を読み込み、要件を理解する
3. 作業メモファイル（`.tmp/memo/` 配下）を作成・更新する
   - ファイル名はブランチ名の `/:*?"<>|\` をハイフン `-` に変換したもの + `.md`

### Step 3: 既存実装の確認

既存のコミットやPRを確認する。すでに進行中の場合はStep 4〜10をスキップする。

### Step 4: 依存Issueの確認

依存するIssueが解決済みであることを確認する。未解決の場合はユーザーに通知する。

### Step 5: 実装

- Goコードの変更がある場合: `/tdd` スキルを使用する
- それ以外の変更: 直接編集する
- サブスキルは中断せず完了まで実行すること

### Step 6: コード品質チェック

`/simplify` を呼び出し、変更コードの再利用性・品質・効率をレビュー・修正する。

### Step 7: セルフレビュー

`/review` スキルを呼び出し、品質とドキュメントの整合性を確認する。

### Step 8: Lint/フォーマットチェック

変更したファイルタイプに応じてチェックを実行する：
- Go: `gofmt`, `golangci-lint run`
- Shell: `shellcheck`

### Step 9: 重複PRチェック

既存のPRと重複しないことを確認する。

### Step 10: リベースコンフリクトチェック

mainブランチに対してリベースを実行し、コンフリクトがあれば解決する。

### Step 11: PR作成

`commit-commands:commit-push-pr` を使用してPRを作成する。
- PR本文に `Closes #<issue_number>` を含めること

### Step 12: ローカルPR修正

`/fix-pr-local` スキルを適用し、CI、レビュー対応、マージを行う。

### Step 13: 振り返り

`/retro` スキルを実行する。

### Step 14: 完了通知

`/vox-actor-plugin:monologue` で作業完了を宣言する。

## セッション再開時のルール

- git log を確認し、前回の進捗を把握する
- 作業メモファイルを参照して状態を復元する
