# CLAUDE.md

このファイルはできるだけシンプルに保つ。詳細なルールは `.claude/rules/` に分離する。
スキルの情報は `.claude/skills/` に記載する。

## GitHub Issue/PR管理

Issue やPR を作成する際は、末尾に以下のフッターを付与すること。

```text
🤖 Generated with [Claude Code](https://claude.ai/code)
```

作成後は内容を確認すること。

## 独り言（monologue）スキル

`/vox-actor-plugin:monologue` スキルを以下のタイミングで積極的に使うこと。

- 作業開始時
- 作業終了時
- 想定外のことが起こったとき
- ユーザーに確認が必要なとき
- 作業途中の区切りの良いとき
