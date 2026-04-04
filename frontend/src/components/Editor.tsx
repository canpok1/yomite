import { useEffect, useState } from "react";
import { ListProviders, ListPersonas } from "../../wailsjs/go/main/App";
import type { SimulationStatus } from "../hooks/useSimulation";

interface EditorProps {
  onStart: (text: string, providerID: string, personaID: string) => void;
  onStop: () => void;
  status: SimulationStatus;
}

export function Editor({ onStart, onStop, status }: EditorProps) {
  const [text, setText] = useState("");
  const [provider, setProvider] = useState("");
  const [persona, setPersona] = useState("");
  const [providers, setProviders] = useState<string[]>([]);
  const [personas, setPersonas] = useState<Record<string, string>>({});
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([ListProviders(), ListPersonas()])
      .then(([providerList, personaMap]) => {
        setProviders(providerList);
        setPersonas(personaMap);
        if (providerList.length > 0) setProvider(providerList[0]);
        const personaIds = Object.keys(personaMap);
        if (personaIds.length > 0) setPersona(personaIds[0]);
      })
      .catch((err) => {
        setLoadError(
          err instanceof Error ? err.message : String(err),
        );
      });
  }, []);

  const isRunning = status === "running";
  const canStart = !isRunning && text.trim() !== "" && provider !== "" && persona !== "";

  function handleRun() {
    if (!canStart) return;
    onStart(text, provider, persona);
  }

  return (
    <div className="flex flex-col gap-4 h-full">
      {loadError && (
        <div className="p-2 rounded bg-red-100 dark:bg-red-900 text-red-700 dark:text-red-300 text-sm">
          設定読み込みエラー: {loadError}
        </div>
      )}

      <div className="flex gap-2">
        <label className="flex items-center gap-1 text-sm text-gray-600 dark:text-gray-400">
          プロバイダ
          <select
            value={provider}
            onChange={(e) => setProvider(e.target.value)}
            disabled={isRunning}
            className="px-2 py-1 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm disabled:opacity-50"
          >
            {providers.map((p) => (
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
            disabled={isRunning}
            className="px-2 py-1 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm disabled:opacity-50"
          >
            {Object.entries(personas).map(([id, name]) => (
              <option key={id} value={id}>
                {name}
              </option>
            ))}
          </select>
        </label>
      </div>

      <textarea
        value={text}
        onChange={(e) => setText(e.target.value)}
        placeholder="文章を入力してください…"
        disabled={isRunning}
        className="flex-1 min-h-[200px] p-3 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
      />

      <div className="flex gap-2 self-end">
        {isRunning && (
          <button
            onClick={onStop}
            className="px-6 py-2 bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
          >
            停止
          </button>
        )}
        <button
          onClick={handleRun}
          disabled={!canStart}
          className="px-6 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          実行
        </button>
      </div>
    </div>
  );
}
