---
name: assign-issues
description: open状態のIssueを優先度順に評価し、指定件数にassign-to-claudeラベルを付与するスキル
allowed-tools: Bash, Read, Grep, Glob, Agent
user-invocable: true
argument-hint: "[--count N]"
---

Issueの自動アサインを行います。

## ワークフロー

1. `/vox-actor-plugin:monologue` でアサイン開始を宣言する
2. `gh issue list` でopen状態かつ `ready` ラベル付きのIssueを取得する
3. 以下の条件に該当するIssueを除外する：
   - すでに `assign-to-claude` または `in-progress-by-claude` ラベルが付与されている
   - タイトルまたは本文に `.claude/` パスへの参照を含む
   - `.claude/` ディレクトリの変更を主目的とする（スキル/skill、ルール/rule、フック/hook、CLAUDE.md、自動化/automation）
4. `.claude/` 除外対象のIssueからは `ready` ラベルを除去し、除去をログに記録する
5. 残りのIssueを `issue-assigner` エージェントの優先度基準に従って優先順位付けする
6. 上位N件（デフォルト: 2件）に `assign-to-claude` ラベルを付与する
7. `/vox-actor-plugin:monologue` でアサイン完了を宣言する

## 出力要件

- 除外したIssue: 番号、タイトル、除外理由（ターミナルのみ）
- アサインしたIssue: 番号、タイトル、判定根拠（ターミナルのみ）

## 制約事項

- GitHub Issueへのコメント投稿（`gh issue comment`）は禁止
- 許可されるghコマンド: `gh issue list`, `gh issue edit` のみ
