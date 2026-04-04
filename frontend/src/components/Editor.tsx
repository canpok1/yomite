import { useState } from "react";

const PROVIDERS = ["ollama"] as const;
const PERSONAS = ["default"] as const;

interface EditorProps {
  onLoadDocument: (text: string) => void;
}

export function Editor({ onLoadDocument }: EditorProps) {
  const [text, setText] = useState("");
  // NOTE: provider/personaはGoバインディング接続時にコールバック経由で渡す予定。
  // 現時点ではUI表示のみで実行時には未使用。
  const [provider, setProvider] = useState<string>(PROVIDERS[0]);
  const [persona, setPersona] = useState<string>(PERSONAS[0]);

  function handleRun() {
    if (!text.trim()) return;
    onLoadDocument(text);
  }

  return (
    <div className="flex flex-col gap-4 h-full">
      <div className="flex gap-2">
        <label className="flex items-center gap-1 text-sm text-gray-600 dark:text-gray-400">
          プロバイダ
          <select
            value={provider}
            onChange={(e) => setProvider(e.target.value)}
            className="px-2 py-1 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm"
          >
            {PROVIDERS.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
        </label>
        <label className="flex items-center gap-1 text-sm text-gray-600 dark:text-gray-400">
          ペルソナ
          <select
            value={persona}
            onChange={(e) => setPersona(e.target.value)}
            className="px-2 py-1 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm"
          >
            {PERSONAS.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
        </label>
      </div>

      <textarea
        value={text}
        onChange={(e) => setText(e.target.value)}
        placeholder="文章を入力してください…"
        className="flex-1 min-h-[200px] p-3 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 resize-none focus:outline-none focus:ring-2 focus:ring-blue-500"
      />

      <button
        onClick={handleRun}
        disabled={!text.trim()}
        className="px-6 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors self-end"
      >
        実行
      </button>
    </div>
  );
}
