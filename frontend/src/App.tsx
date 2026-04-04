import { useState } from "react";
import { Editor } from "./components/Editor";
import { SentenceList } from "./components/SentenceList";
import type { Sentence } from "./types";

// NOTE: Go側のLoadDocumentバインディングが未接続のため、
// クライアント側で簡易的に文分割を行うスタブ実装。
// 正確な分割ロジックはGo側(core/document.go)にあり、バインディング接続時に置き換える。
function splitSentencesStub(text: string): Sentence[] {
  return text
    .split(/(?<=[。！？」])|(?<=[.!?])\s/)
    .map((s) => s.trim())
    .filter(Boolean)
    .map((content, index) => ({ index, content }));
}

function App() {
  const [sentences, setSentences] = useState<Sentence[]>([]);

  function handleLoadDocument(text: string) {
    // TODO: LoadDocumentバインディングに置き換え
    const result = splitSentencesStub(text);
    setSentences(result);
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
            <Editor onLoadDocument={handleLoadDocument} />
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

        {/* 右パネル: シミュレーション結果（プレースホルダー） */}
        <section className="w-1/2 p-4 flex flex-col">
          <h2 className="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-2">
            シミュレーション結果
          </h2>
          <div className="flex-1 flex items-center justify-center text-gray-400 dark:text-gray-500">
            <p>シミュレーション結果がここに表示されます</p>
          </div>
        </section>
      </div>
    </div>
  );
}

export default App;
