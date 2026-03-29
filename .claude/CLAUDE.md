# CLAUDE.md

このファイルはできるだけシンプルに保つ。詳細なルール��� `.claude/rules/` に分離する。
スキルの情報は `.claude/skills/` に記載する。

## GitHub Issue/PR管理

Issue やPR を作成する際は、末尾に以下のフッターを付与すること。

```text
🤖 Generated with [Claude Code](https://claude.ai/code)
```

作成後は内容を確認すること。

## コミュニケーシ���ン

作業中は `base-tools:monologue` スキルを活用して思考プ��セスを透明化すること。
特に以下の場面では必ず呼び出すこと：

- 作業開始時・終了時
- ユーザーへの確認が必要な局面
- 予期しない事態が発生した場合
