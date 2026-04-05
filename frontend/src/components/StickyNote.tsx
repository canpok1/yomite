import type { Note, NoteType } from "../types";
import { NOTE_LABELS } from "../constants/noteStyles";

const NOTE_STYLES: Record<
  NoteType,
  { bg: string; border: string; text: string }
> = {
  QUESTION: {
    bg: "bg-yellow-100 dark:bg-yellow-900",
    border: "border-yellow-400 dark:border-yellow-600",
    text: "text-yellow-800 dark:text-yellow-200",
  },
  CONFUSION: {
    bg: "bg-red-100 dark:bg-red-900",
    border: "border-red-400 dark:border-red-600",
    text: "text-red-800 dark:text-red-200",
  },
  RESOLVED: {
    bg: "bg-green-100 dark:bg-green-900",
    border: "border-green-400 dark:border-green-600",
    text: "text-green-800 dark:text-green-200",
  },
};

interface StickyNoteProps {
  note: Note;
  stepNumber: number;
}

export function StickyNote({ note, stepNumber }: StickyNoteProps) {
  const style = NOTE_STYLES[note.type];
  const label = NOTE_LABELS[note.type];
  return (
    <div
      className={`${style.bg} ${style.border} border-l-4 rounded-r px-3 py-1.5 text-sm`}
    >
      <span className={`font-semibold ${style.text}`}>
        [{label}]
      </span>{" "}
      <span className="text-gray-700 dark:text-gray-300">
        {note.content}
      </span>
      <span className="ml-2 text-xs text-gray-400 dark:text-gray-500">
        ステップ {stepNumber}
      </span>
    </div>
  );
}

interface StickyNoteStackProps {
  notes: { note: Note; stepNumber: number }[];
}

export function StickyNoteStack({ notes }: StickyNoteStackProps) {
  if (notes.length === 0) return null;
  return (
    <div className="flex flex-col gap-1 mt-2">
      {notes.map(({ note, stepNumber }) => (
        <StickyNote key={stepNumber} note={note} stepNumber={stepNumber} />
      ))}
    </div>
  );
}
