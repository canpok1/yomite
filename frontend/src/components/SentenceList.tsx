import { useEffect, useMemo, useRef } from "react";
import type { Note, NoteType, Sentence, SimulationStep } from "../types";
import { NOTE_LABELS } from "../constants/noteStyles";
import { StickyNoteStack } from "./StickyNote";

interface SentenceListProps {
  sentences: Sentence[];
  steps?: SimulationStep[];
  isRunning?: boolean;
}

type Direction = "done" | "forward" | "backward";

const DIRECTION_STYLES: Record<Direction, { label: string; color: string }> = {
  done: {
    label: "読了",
    color: "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300",
  },
  forward: {
    label: "前進",
    color: "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300",
  },
  backward: {
    label: "後退",
    color: "bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-300",
  },
};

// NOTE: StepRowのバッジ色はStickyNoteの背景色系統と意図的に異ならせている。
// StickyNoteは付箋カードとして背景色+左ボーダーで表現し、StepRowバッジはテキスト色のみで簡潔に表示する。
const NOTE_TYPE_COLORS: Record<NoteType, string> = {
  QUESTION: "text-purple-600 dark:text-purple-400",
  RESOLVED: "text-green-600 dark:text-green-400",
  CONFUSION: "text-red-600 dark:text-red-400",
};

function getDirection(step: SimulationStep): Direction {
  if (step.next_index === null) return "done";
  if (step.next_index > step.current_index) return "forward";
  return "backward";
}

function groupStepsBySentence(steps: SimulationStep[]): Map<number, SimulationStep[]> {
  const map = new Map<number, SimulationStep[]>();
  for (const step of steps) {
    const group = map.get(step.current_index) ?? [];
    group.push(step);
    map.set(step.current_index, group);
  }
  return map;
}

type SentenceNoteState = "confusion" | "resolved" | "none";

function getSentenceNoteState(relatedSteps: SimulationStep[] | undefined): SentenceNoteState {
  if (!relatedSteps) return "none";
  let hasResolved = false;
  for (const step of relatedSteps) {
    // NOTE: 混乱は解決より読者体験上重大なため、見つかった時点で即確定する
    if (step.note?.type === "CONFUSION") return "confusion";
    if (step.note?.type === "RESOLVED") hasResolved = true;
  }
  if (hasResolved) return "resolved";
  return "none";
}

const SENTENCE_BG_STYLES: Record<SentenceNoteState, string> = {
  confusion: "bg-red-50 dark:bg-red-950 border-red-200 dark:border-red-800",
  resolved: "bg-green-50 dark:bg-green-950 border-green-200 dark:border-green-800",
  none: "bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700",
};

function StepRow({ step }: { step: SimulationStep }) {
  const dirStyle = DIRECTION_STYLES[getDirection(step)];
  return (
    <div className="flex items-center gap-2 flex-wrap">
      <span className="shrink-0 w-6 h-6 flex items-center justify-center rounded-full bg-gray-200 dark:bg-gray-700 text-xs font-bold">
        {step.step}
      </span>
      <span
        className={`px-2 py-0.5 rounded text-xs font-semibold ${dirStyle.color}`}
      >
        {dirStyle.label}
      </span>
      {step.next_index !== null && (
        <span className="text-xs text-gray-500 dark:text-gray-400">
          → 文 {step.next_index + 1}
        </span>
      )}
      {step.note && (
        <span
          className={`text-xs font-semibold ${NOTE_TYPE_COLORS[step.note.type]}`}
        >
          [{NOTE_LABELS[step.note.type]}]
        </span>
      )}
    </div>
  );
}

export function SentenceList({ sentences, steps = [], isRunning = false }: SentenceListProps) {
  const { stepsBySentence, noteStateBySentence, notesBySentence } = useMemo(() => {
    const stepsBySentence = groupStepsBySentence(steps);
    const noteStateBySentence = new Map<number, SentenceNoteState>();
    const notesBySentence = new Map<number, { note: Note; stepNumber: number }[]>();
    for (const [idx, relatedSteps] of stepsBySentence) {
      noteStateBySentence.set(idx, getSentenceNoteState(relatedSteps));
      const notes: { note: Note; stepNumber: number }[] = [];
      for (const step of relatedSteps) {
        if (step.note) {
          notes.push({ note: step.note, stepNumber: step.step });
        }
      }
      if (notes.length > 0) {
        notesBySentence.set(idx, notes);
      }
    }
    return { stepsBySentence, noteStateBySentence, notesBySentence };
  }, [steps]);

  // NOTE: isRunning=true のときのみ、最終ステップの next_index を「現在読書中の文」とみなす。
  // 完了ステップ（next_index=null）は読書中扱いしない。
  const currentSentenceIndex = useMemo(() => {
    if (!isRunning || steps.length === 0) return null;
    const lastStep = steps[steps.length - 1];
    if (lastStep.next_index === null) return null;
    return lastStep.next_index;
  }, [steps, isRunning]);

  const currentSentenceRef = useRef<HTMLLIElement>(null);
  // NOTE: 依存配列はcurrentSentenceIndexのみ。同一文に留まるステップ追加時の再スクロールは不要なため、steps.lengthは含めない。
  useEffect(() => {
    if (currentSentenceIndex !== null) {
      currentSentenceRef.current?.scrollIntoView({ behavior: "smooth", block: "center" });
    }
  }, [currentSentenceIndex]);

  return (
    <ol className="flex flex-col gap-2">
      {sentences.map((sentence) => {
        const relatedSteps = stepsBySentence.get(sentence.index);
        const noteState = noteStateBySentence.get(sentence.index) ?? "none";
        const isCurrent = sentence.index === currentSentenceIndex;
        return (
          <li
            key={sentence.index}
            ref={isCurrent ? currentSentenceRef : null}
            data-sentence-index={sentence.index}
            className={`flex gap-3 p-3 rounded border ${SENTENCE_BG_STYLES[noteState]}${isCurrent ? " ring-2 ring-blue-400 dark:ring-blue-500" : ""}`}
          >
            <span className="shrink-0 w-8 h-8 flex items-center justify-center rounded-full bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 text-sm font-bold">
              {sentence.index + 1}
            </span>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 pt-1">
                <span>{sentence.content}</span>
                {isCurrent && (
                  <span className="shrink-0 px-2 py-0.5 rounded text-xs font-semibold bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300 animate-pulse">
                    読書中
                  </span>
                )}
              </div>
              <StickyNoteStack notes={notesBySentence.get(sentence.index) ?? []} />
              {relatedSteps && (
                <div className="mt-2 flex flex-col gap-1">
                  {relatedSteps.map((step) => (
                    <StepRow key={step.step} step={step} />
                  ))}
                </div>
              )}
            </div>
          </li>
        );
      })}
    </ol>
  );
}
