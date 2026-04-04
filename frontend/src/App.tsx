import { useState } from 'react';
import { Greet } from '../wailsjs/go/main/App';

function App() {
  const [name, setName] = useState('');
  const [result, setResult] = useState('');

  async function greet() {
    if (!name.trim()) return;
    try {
      const greeting = await Greet(name);
      setResult(greeting);
    } catch (e) {
      setResult(`エラー: ${e}`);
    }
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100 p-8">
      <h1 className="text-4xl font-bold mb-8">yomite</h1>
      <div className="flex gap-2 mb-4">
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && greet()}
          placeholder="名前を入力"
          className="px-4 py-2 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <button
          onClick={greet}
          className="px-6 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
        >
          Greet
        </button>
      </div>
      {result && (
        <p className="text-lg mt-4 p-4 bg-white dark:bg-gray-800 rounded shadow">
          {result}
        </p>
      )}
    </div>
  );
}

export default App;
