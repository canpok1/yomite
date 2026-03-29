---
name: fix-pr-local
description: PR のCI監視、レビュー対応、マージを自動化するスキル。auto-mergeを活用する。
allowed-tools: Bash, Read, Grep, Glob, Write, Edit
user-invocable: true
argument-hint: "<pr_number>"
---

PRのCI、レビュー対応、マージを自動化します。

## メインループ

以下のループを繰り返す：

### Step 1: fix-pr.sh の実行

`.claude/skills/fix-pr-local/scripts/fix-pr.sh <PR番号>` を実行する。

### Step 2: 終了コードに応じた対応

- **Exit 0**: マージ完了。ループを終了する。
- **Exit 1**: スクリプトエラー。コンフリクトがあれば解決し、ユーザーに通知する。
- **Exit 10**: CI失敗。ログを分析してコード修正を行う。
- **Exit 20**: 未解決のレビュースレッドがある。GraphQLで全スレッドを取得し、以下の手順で対応する。
    1. 指摘箇所のコードを読み、`# NOTE:` 等の設計意図コメントがあるか確認する
    2. 設計意図コメントがあれば、その内容を引用してレビュースレッドに返信し、スレッドを解決する
    3. 設計意図コメントがなければ、通常通りコード修正で対応する
- **Exit 30**: 承認待ち。CIが通りスレッドも解決済みだが、承認がない。
  - CodeRabbitのレート制限コメントがある場合は `handle-coderabbit-rate-limit.sh <PR_NUMBER> <WAIT_SECONDS>` を実行する。
  - それ以外は、コメント者（リポジトリオーナー除く）に承認をリクエストする。

## 重要な制約

- ループの各反復前に `sleep 10` を実行し、CIの早期完了検出を防ぐ
- コンフリクト解決時はコミットメッセージにIssue番号を含める
- マージ後のクリーンアップは本スキルの責務外
