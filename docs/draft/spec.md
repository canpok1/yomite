# AI読者シミュレーター「ヨミテ (Yomite)」外部仕様書

## 1. プロジェクト概要
* **コンセプト:** AIが執筆者の「身代わりの読み手」となり、文章を読み進める際の脳内プロセス（疑問・納得・読み返し）をシミュレートする。
* **コア価値:** **AI読者**がどこで迷い、どこを読み直したかを可視化することで、執筆者が自発的に文章を「デバッグ」できる環境を提供する。
* **開発指針:** Gemini API特化(MVP)、設定と人格の完全分離、ローカル完結、単一バイナリ配布。

## 2. 技術スタック
* **Framework:** **Wails v2/v3** (Go + React/TypeScript)
* **LLM Integration:** **Google Gemini API** (MVP対応)
    * 将来的なマルチモデル展開を見据えたインターフェース設計。
* **Repository:** `yomite`
* **Command:** `yomite`

---

## 3. 設定ファイル構成 (`config.json`)
接続情報 (`providers`) と人格情報 (`personas`) をIDキーによるMap形式で管理。メンテナンス性と取得速度を両立。

```json
{
  "current_provider": "gemini_std",
  "current_persona": "beginner",
  "providers": {
    "gemini_std": {
      "type": "google",
      "model": "gemini-1.5-pro",
      "api_key": "YOUR_GEMINI_API_KEY"
    }
  },
  "personas": {
    "beginner": {
      "display_name": "初学者",
      "system_prompt": "あなたは知識レベルが『初学者』の読者として振る舞ってください。専門用語には敏感に反応し、不明点があれば前の文に戻ってください。"
    },
    "expert": {
      "display_name": "専門家",
      "system_prompt": "あなたは該当分野の『専門家』として振る舞ってください。論理の飛躍や根拠の薄い記述を厳しくチェックします。"
    }
  }
}
```

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

### ② SimulationStep (AI読者の思考ログ)
```go
type SimulationStep struct {
    Step        int      `json:"step"`
    SentenceIdx int      `json:"sentence_idx"`
    Action      string   `json:"action"`       // "READ" | "BACKTRACK"
    TargetIdx   int      `json:"target_idx"`   // 戻り先（なければ-1）
    Note        Note     `json:"note"`         // 思考の内容
    MemoryLoad  float64  `json:"memory_load"`  // 0.0〜1.0（認知負荷）
}

type Note struct {
    Type    string `json:"type"`    // "QUESTION" | "RESOLVED" | "CONFUSION"
    Content string `json:"content"` 
}
```

---

## 5. 機能要件 (MVP)

### 5.1 AI読者シミュレーション
* **逐次読み解析:** Geminiに対し、選択された `system_prompt` と `Sentences` を渡し、1文ずつ読み進めた際の反応をJSONで取得。
* **バックトラック検知:** 指示語が不明瞭な際や、論理の不一致を感じた際に、過去の `SentenceIdx` へ戻る挙動を記録。

### 5.2 ユーザーインターフェース
* **ヨミテ・エディタ:** 文章入力エリア。シミュレーション後に負荷状況をヒートマップ（色の濃淡）で表示。
* **思考レイヤー:** 各文の横にAI読者の「付箋（Note）」を表示し、読み返し箇所を「矢印」で視覚化。
* **CLIモード:** `yomite run -f sample.txt` で実行し、思考ログを標準出力。

### 5.3 設定管理
* **ポータビリティ:** `config.json` はバイナリと同ディレクトリ、または `~/.config/yomite/` に配置し、ユーザーが直接編集可能とする。

---

## 6. ディレクトリ構成

```text
yomite/
├── core/                # Go: 共通ロジック
│   ├── config.go        # Map形式の設定読み込み
│   ├── document.go      # 文分割ロジック
│   └── simulator.go     # Gemini API連携・プロンプト管理
├── cmd/
│   └── yomite/          # Go: CLIインターフェース
├── frontend/            # TS: GUIインターフェース (React/Tailwind)
├── main.go              # Go: Wailsエントリポイント
└── wails.json           # Wailsビルド設定
```

