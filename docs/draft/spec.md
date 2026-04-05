# AI読者シミュレーター「ヨミテ (Yomite)」外部仕様書

## 1. プロジェクト概要
* **コンセプト:** AIが執筆者の「身代わりの読み手」となり、文章を読み進める際の脳内プロセス（疑問・納得・読み返し）をシミュレートする。
* **コア価値:** **AI読者**がどこで迷い、どこを読み直したかを可視化することで、執筆者が自発的に文章を「デバッグ」できる環境を提供する。
* **開発指針:** Ollama特化(MVP)、設定と人格の完全分離、ローカル完結、単一バイナリ配布。

## 2. 技術スタック
* **Framework:** **Wails v2/v3** (Go + React/TypeScript)
* **LLM Integration:** **Ollama** (ローカルLLM、MVP対応)
    * ユーザーが事前にOllamaをインストールし、使用モデルをpull済みであることを前提とする。
    * 将来的なマルチモデル展開を見据えたインターフェース設計。
* **Repository:** `yomite`
* **Command:** `yomite`

---

## 3. 設定ファイル構成 (`config.json`)
接続情報 (`providers`) と人格情報 (`personas`) をIDキーによるMap形式で管理。メンテナンス性と取得速度を両立。

### 設定ファイルの探索と読み込み

1. **カレントディレクトリの `yomite.json`** — 優先（プロジェクト固有の設定）
2. **`~/.config/yomite/config.json`** — フォールバック（ユーザーグローバル設定）

* 両方存在する場合はグローバル設定をベースにローカル設定で上書き（マージ）する。
* どちらも存在しない場合はエラーとする（将来的にGUIでは画面上で設定可能にする）。
* `default_provider` / `default_persona` が `providers` / `personas` に存在しないIDを指している場合はバリデーションエラーとする。

### 設定例

```json
{
  "log": {
    "level": "info",
    "path": "/tmp/yomite.log"
  },
  "default_provider": "local_ollama",
  "default_persona": "beginner",
  "providers": {
    "local_ollama": {
      "type": "ollama",
      "model": "gemma2",
      "origin": "http://localhost:11434"
    }
  },
  "personas": {
    "beginner": {
      "display_name": "初学者",
      "system_prompt": "あなたは知識レベルが『初学者』の読者として振る舞ってください。専門用語には敏感に反応し、不明点があれば前の文に戻ってください。",
      "memory_capacity": 200,
      "max_steps": 100
    },
    "expert": {
      "display_name": "専門家",
      "system_prompt": "あなたは該当分野の『専門家』として振る舞ってください。論理の飛躍や根拠の薄い記述を厳しくチェックします。",
      "memory_capacity": 500,
      "max_steps": 60
    }
  }
}
```

### 設定フィールド

| フィールド | 説明 |
|---|---|
| `log.level` | ログレベル（`"debug"`, `"info"`, `"warn"`）。上位は下位を包含する |
| `log.path` | ログ出力先ファイルパス（必須） |
| `default_provider` | CLIオプション等で未指定時に使用するプロバイダID |
| `default_persona` | CLIオプション等で未指定時に使用するペルソナID |
| `providers.*.type` | プロバイダ種別（MVP では `"ollama"` のみ） |
| `providers.*.model` | 使用するモデル名（ユーザーが設定必須） |
| `providers.*.origin` | Ollamaのオリジン（デフォルト: `http://localhost:11434`） |
| `personas.*.display_name` | ペルソナの表示名 |
| `personas.*.system_prompt` | AI読者に渡すシステムプロンプト |
| `personas.*.memory_capacity` | 記憶バッファの最大文字数 |
| `personas.*.max_steps` | シミュレーションのステップ数上限（デフォルト: 文数 × 3） |

---

## 4. 主要データ構造 (Core Schema)

### ① Document (本文構造)
```go
type Document struct {
    ID        string     `json:"id"`
    RawText   string     `json:"raw_text"`
    Sentences []Sentence `json:"sentences"` // 文分割済みのデータ
}

type Sentence struct {
    Index   int    `json:"index"`   // 0から始まる連番
    Content string `json:"content"` // 文の文字列
}
```

#### 文分割ルール

`RawText` から `Sentences` を生成する際のルールを以下に定義する。

- **全角区切り文字（直後で分割）:** 句点（。）、全角感嘆符（！）、全角疑問符（？）、閉じ引用符（」）
  - 区切り文字は直前の文に含める。
- **半角区切り文字（直後で分割）:** ピリオド（`.`）+ 半角スペース、半角感嘆符（`!`）+ 半角スペース、半角疑問符（`?`）+ 半角スペース
  - 区切り文字は直前の文に含める。半角スペースは除去する。
- **開き引用符（直前で分割）:** 開き引用符（「）の直前で分割する。「 は新しい文の先頭になる。
  - ネストした引用符（例: `「彼は「了解」と言った」`）も同じルールを機械的に適用する。
- **段落区切り:** 連続改行（空行）は、句点等がなくても文の区切りとして扱う。
- **空文除外:** 分割後に空白のみ・空文字となった文はフィルタリングして除外する。

**例:** `彼は「了解。すぐ行く。」と言った。` → 4文:
1. `彼は`
2. `「了解。`
3. `すぐ行く。」`
4. `と言った。`

**実装箇所:** `internal/core/document.go` の `SplitSentences()` 関数

### ② SimulationStep (AI読者の思考ログ)
```go
type SimulationStep struct {
    Step        int    `json:"step"`          // 1始まりの連番
    SentenceIdx int    `json:"current_index"` // 今読んだ文の位置
    TargetIdx   *int   `json:"next_index"`   // 次に読む文の位置 (nil = 読了)
    Note        *Note  `json:"note"`         // 思考の内容 (nil = 感想なし)
    Memory      string `json:"memory"`       // このステップ時点での記憶内容
}

type Note struct {
    Type    string `json:"type"`    // "QUESTION" | "RESOLVED" | "CONFUSION"
    Content string `json:"content"`
}
```

---

## 5. 機能要件 (MVP)

### 5.1 AI読者シミュレーション

#### シミュレーション方式
プログラムがシミュレーションの流れを制御し、各ステップでAIに判断を委ねる**逐次実行方式**を採用する。

#### シミュレーションループ
```text
1. 視線位置 = 0（先頭文）
2. AIに現在の文 + コンテキストを送信（毎回独立したリクエスト）
3. Phase 1（Note）: AIが感想と次の行動を返す
4. Phase 2（Memory）: AIが記憶内容をプレーンテキストで返す
5. プログラム側で次の視線位置を計算し、SimulationStep として記録
6. 視線位置を更新し、2 に戻る
7. 終了条件を満たしたら終了
```

#### 終了条件
* AIが `next_action: "finish"` を返した場合、または最後の文で `next_action: "next"` を返した場合（読了）。
* ステップ数がペルソナの `max_steps` に達した場合（デフォルト: 文数 × 3）。

#### AIへの入力（毎ステップ）

| 情報 | 内容 |
|---|---|
| ペルソナ | `system_prompt`（システムプロンプトとして） |
| 現在の文 | 今読んでいる1文のテキスト |
| 現在の位置 | 文のインデックスと文の総数 |
| 記憶バッファ | AIが過去ステップで記憶した自由テキスト |

* 全文リストは渡さない。人間が頭から順に読み、記憶を頼りに読み進める体験を模倣する。
* 会話履歴は保持しない。毎回独立したリクエストとし、記憶バッファのみがコンテキストとなる。

#### AIからの出力

##### Phase 1: Note（感想生成）

```json
{
  "next_action": "next",
  "feeling": "この用語の定義がまだ出てきていない",
  "feeling_type": "question"
}
```

| フィールド | 型 | 説明 |
|---|---|---|
| `next_action` | string | `"next"`: 次の文へ / `"back:N"`: N文戻る / `"finish"`: 読了 |
| `feeling` | string \| null | 感想の内容。すんなり読めた場合は `null` |
| `feeling_type` | string \| null | `"question"`: 疑問 / `"resolved"`: 解消 / `"confusion"`: 混乱。`feeling` が `null` の場合は `null` |

プログラム側で `next_action` から次の視線位置（`next_index`）を計算し、`feeling` / `feeling_type` から `Note` 構造体を組み立てる。`current_index` はプログラムが管理する値をそのまま使用する。

##### Phase 2: Memory（記憶生成）

プレーンテキストで記憶内容を返す（JSON形式は不要）。

#### 記憶バッファ
* 形式: 自由テキスト（1つの文字列）。AIが記憶の構造を自由に決める。
* 更新: AIが毎ステップ記憶全体を返し、プログラム側で丸ごと上書きする。
* 上限: ペルソナごとに `memory_capacity`（最大文字数）で制限。
* AIが覚えておく内容・忘れる内容を自ら判断する。

#### 視線移動
* AIは `next_action` で次の行動を指定する。`"next"` で次の文、`"back:N"` でN文戻る。
* 既読か読み返しかはプログラム側で既読管理から判定する（AI側では判定しない）。

#### 異常系
* AIが不正なJSONや不正な `next_action` 値を返した場合のエラーハンドリングを行う。

### 5.2 ユーザーインターフェース

#### GUI
* **ヨミテ・エディタ:** 文章入力エリア。
* **付箋（Note）:** 各文の右横にAI読者の感想を表示する。`Note.Type` により色分けする。
  * `QUESTION`（疑問）: 黄色
  * `CONFUSION`（混乱）: 赤
  * `RESOLVED`（解消）: 緑
* **矢印:** 文から文への視線移動（バックトラック・先読み）を矢印で全件可視化する。
* **設定ビューア:** ヘッダー右上の歯車アイコンから開く読み取り専用パネル。プロバイダ一覧・ペルソナ一覧を表示し、デフォルト設定をハイライトする。system_promptは折りたたみ表示。

#### CLI
* **コマンド:** `yomite run -f sample.txt`
* **出力:** デフォルトは人間向けテキスト形式。`--json` フラグでJSON出力に切替可能。
* **オプション:**

| フラグ | 説明 |
|---|---|
| `-f` | 入力テキストファイルのパス |
| `--provider` | 使用するプロバイダIDを指定（未指定時は `default_provider`） |
| `--persona` | 使用するペルソナIDを指定（未指定時は `default_persona`） |
| `--json` | 出力をJSON Lines形式に切替（1ステップ1行） |
| `--config` | config.jsonのパスを明示指定 |

### 5.3 設定管理
* **探索:** カレントディレクトリの `yomite.json` を優先し、`~/.config/yomite/config.json` をフォールバックとする。詳細は「3. 設定ファイル構成」を参照。
* **ポータビリティ:** ユーザーが直接編集可能なJSON形式。GUI/CLI共通の設定ファイル。

---

## 6. ディレクトリ構成

```text
yomite/
├── cmd/
│   └── yomite/          # Go: CLIエントリポイント
├── internal/
│   ├── core/            # Go: CLI・GUI共通ロジック
│   │   ├── config.go    # Map形式の設定読み込み
│   │   ├── document.go  # 文分割ロジック
│   │   └── simulator.go # Ollama連携・プロンプト管理
│   ├── cli/             # Go: CLI固有ロジック
│   └── gui/             # Go: GUI固有ロジック (Wailsバインディング)
├── frontend/            # TS: GUIインターフェース (React/Tailwind)
├── main_gui.go          # Go: Wailsエントリポイント (ルート維持)
└── wails.json           # Wailsビルド設定
```
