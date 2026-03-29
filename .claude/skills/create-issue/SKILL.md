---
name: create-issue
description: ユーザーとの対話を通じて仕様を整理し、GitHub Issueを作成するスキル。実装は行わない。
allowed-tools: Read, Grep, Glob, Bash, AskUserQuestion, WebSearch, WebFetch
user-invocable: true
---

GitHub Issueを作成します。
実装は行いません。Issue作成のみに専念します。

## 禁止事項

- ファイル編集（Write、Edit ツールの使用は禁止）
- git コマンドによる変更操作
- 実装作業

## ワークフロー

### 1. ヒアリング

`AskUserQuestion` を使い、以下を確認する：
- 目的: 何を実現したいか
- 背景: なぜ必要か
- 制約条件: 考慮すべき制約

### 2. 調査

コードベースを `Glob`、`Grep`、`Read` で調査し、関連する実装を把握する。

### 3. 仕様整理

会話内容を以下の形式でまとめる：
- 概要
- 背景
- やりたいこと
- 実装方針（調査結果に基づく）

### 4. 確認

`AskUserQuestion` でユーザーにドラフトを提示し、承認を得る。

### 5. Issue作成

`gh issue create` でGitHub上にIssueを作成する。
