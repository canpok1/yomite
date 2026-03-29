# スキル開発ガイド

SKILL.mdファイルの作成・修正時の標準ルール。

## スキル参照の正式表記

- **プラグインスキル**: `{namespace}:{skill-name}` 形式
  - 例: `commit-commands:commit-push-pr`
- **ローカルスキル**: `/skill-name` 形式
  - 例: `/monologue`, `/review`

> 省略形を使うとレビュアーが正式名称と一致しないと判断し、CHANGESが発生する。必ず正式名称を使うこと。

## 正式名称の確認元

### プラグインスキル
- `namespace`: プラグインの `.claude-plugin/plugin.json` 内の `name` フィールド
- `skill-name`: プラグインの `skills/{skill-name}/SKILL.md` のディレクトリ名、またはそのSKILL.md内の `name` フィールド

### ローカルスキル
- `.claude/skills/{skill-name}/SKILL.md` の `name` フィールドで確認
- スラッシュコマンド形式（`/skill-name`）で参照

## SKILL.mdの基本構造

```yaml
---
name: skill-name
description: スキルの説明
allowed-tools: Bash, Read, Grep, Glob, Write, Edit
user-invocable: true
argument-hint: "[引数の説明]"
---
```

フロントマターの後に、スキルの詳細な手順・制約を本文として記載する。
