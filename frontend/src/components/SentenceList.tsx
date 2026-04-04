import type { Sentence } from "../types";

interface SentenceListProps {
  sentences: Sentence[];
}

export function SentenceList({ sentences }: SentenceListProps) {
  return (
    <ol className="flex flex-col gap-2">
      {sentences.map((sentence) => (
        <li
          key={sentence.index}
          className="flex gap-3 p-3 rounded border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800"
        >
          <span className="shrink-0 w-8 h-8 flex items-center justify-center rounded-full bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 text-sm font-bold">
            {sentence.index + 1}
          </span>
          <span className="pt-1">{sentence.content}</span>
        </li>
      ))}
    </ol>
  );
}
