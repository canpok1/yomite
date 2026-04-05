import { useState } from "react";
import { Editor } from "./components/Editor";
import { SentenceList } from "./components/SentenceList";
import { StepList } from "./components/StepList";
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
  const simulation = useSimulation();

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
      <header className="shrink-0 px-6 py-3 border-b border-gray-200 dark:border-gray-700">
        <h1 className="text-xl font-bold">yomite</h1>
      </header>

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
          {sentences.length > 0 && (
            <div className="mt-4 overflow-y-auto max-h-[40%]">
              <h2 className="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-2">
                文一覧
              </h2>
              <SentenceList sentences={sentences} />
            </div>
          )}
        </section>

        {/* 右パネル: シミュレーション結果 */}
        <section className="w-1/2 p-4 flex flex-col">
          <div className="flex items-center gap-3 mb-2">
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
          </div>

          {simulation.error && (
            <div className="mb-3 p-3 rounded bg-red-100 dark:bg-red-900 text-red-700 dark:text-red-300 text-sm">
              {simulation.error}
            </div>
          )}

          <div className="flex-1 overflow-y-auto">
            {simulation.steps.length > 0 ? (
              <StepList steps={simulation.steps} sentences={sentences} />
            ) : (
              <div className="flex items-center justify-center h-full text-gray-400 dark:text-gray-500">
                <p>シミュレーション結果がここに表示されます</p>
              </div>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}

export default App;
