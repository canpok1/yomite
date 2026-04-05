import { useState } from "react";
import { Editor } from "./components/Editor";
import { SentenceList } from "./components/SentenceList";
import { ConfigPanel } from "./components/ConfigPanel";
import { useSimulation, type SimulationStatus } from "./hooks/useSimulation";
import { LoadDocument } from "../wailsjs/go/gui/App";
import type { Sentence } from "./types";

const STATUS_BADGE: Partial<
  Record<SimulationStatus, { label: string; className: string }>
> = {
  running: {
    label: "実行中…",
    className:
      "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300 animate-pulse",
  },
  completed: {
    label: "完了",
    className:
      "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300",
  },
  error: {
    label: "エラー",
    className: "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300",
  },
};

function App() {
  const [sentences, setSentences] = useState<Sentence[]>([]);
  const [showConfig, setShowConfig] = useState(false);
  const simulation = useSimulation();

  const isResultView = sentences.length > 0 && simulation.status !== "idle";

  async function handleStart(text: string, providerID: string, personaID: string) {
    const result = await LoadDocument(text);
    setSentences(result);
    simulation.start(text, providerID, personaID);
  }

  function handleStop() {
    simulation.stop();
  }

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100">
      <header className="shrink-0 px-6 py-3 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
        <h1 className="text-xl font-bold">yomite</h1>
        <button
          onClick={() => setShowConfig((prev) => !prev)}
          className={`p-1.5 rounded transition-colors ${
            showConfig
              ? "bg-gray-200 dark:bg-gray-700"
              : "hover:bg-gray-200 dark:hover:bg-gray-700"
          }`}
          title="設定"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 20 20"
            fill="currentColor"
            className="w-5 h-5"
          >
            <path
              fillRule="evenodd"
              d="M7.84 1.804A1 1 0 0 1 8.82 1h2.36a1 1 0 0 1 .98.804l.331 1.652a6.993 6.993 0 0 1 1.929 1.115l1.598-.54a1 1 0 0 1 1.186.447l1.18 2.044a1 1 0 0 1-.205 1.251l-1.267 1.113a7.047 7.047 0 0 1 0 2.228l1.267 1.113a1 1 0 0 1 .206 1.25l-1.18 2.045a1 1 0 0 1-1.187.447l-1.598-.54a6.993 6.993 0 0 1-1.929 1.115l-.33 1.652a1 1 0 0 1-.98.804H8.82a1 1 0 0 1-.98-.804l-.331-1.652a6.993 6.993 0 0 1-1.929-1.115l-1.598.54a1 1 0 0 1-1.186-.447l-1.18-2.044a1 1 0 0 1 .205-1.251l1.267-1.114a7.05 7.05 0 0 1 0-2.227L1.821 7.773a1 1 0 0 1-.206-1.25l1.18-2.045a1 1 0 0 1 1.187-.447l1.598.54A6.992 6.992 0 0 1 7.51 3.456l.33-1.652ZM10 13a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z"
              clipRule="evenodd"
            />
          </svg>
        </button>
      </header>

      {showConfig ? (
        <ConfigPanel onClose={() => setShowConfig(false)} />
      ) : isResultView ? (
        <div className="flex-1 flex flex-col min-h-0 p-4">
          <div className="flex items-center gap-3 mb-3">
            <h2 className="text-sm font-semibold text-gray-500 dark:text-gray-400">
              シミュレーション結果
            </h2>
            {STATUS_BADGE[simulation.status] && (
              <span
                className={`text-xs px-2 py-0.5 rounded ${STATUS_BADGE[simulation.status]!.className}`}
              >
                {STATUS_BADGE[simulation.status]!.label}
              </span>
            )}
            {simulation.status === "running" && (
              <button
                onClick={handleStop}
                className="ml-auto px-4 py-1 text-sm bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
              >
                停止
              </button>
            )}
          </div>

          {simulation.error && (
            <div className="mb-3 p-3 rounded bg-red-100 dark:bg-red-900 text-red-700 dark:text-red-300 text-sm">
              {simulation.error}
            </div>
          )}

          <div className="flex-1 overflow-y-auto">
            <SentenceList sentences={sentences} steps={simulation.steps} isRunning={simulation.status === "running"} />
          </div>
        </div>
      ) : (
        <div className="flex-1 flex min-h-0">
          {/* 左パネル: エディタ */}
          <section className="w-1/2 p-4 border-r border-gray-200 dark:border-gray-700 flex flex-col">
            <h2 className="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-2">
              ドキュメント
            </h2>
            <div className="flex-1 flex flex-col min-h-0">
              <Editor
                onStart={handleStart}
                onStop={handleStop}
                status={simulation.status}
              />
            </div>
          </section>

          {/* 右パネル: プレースホルダ */}
          <section className="w-1/2 p-4 flex flex-col">
            <h2 className="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-2">
              シミュレーション結果
            </h2>
            <div className="flex-1 flex items-center justify-center text-gray-400 dark:text-gray-500">
              <p>シミュレーション結果がここに表示されます</p>
            </div>
          </section>
        </div>
      )}
    </div>
  );
}

export default App;
