import { useEffect, useMemo, useRef } from "react";
import type { NoteType, Sentence, SimulationStep } from "../types";

interface SentenceListProps {
  sentences: Sentence[];
  steps?: SimulationStep[];
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

const NOTE_TYPE_STYLES: Record<NoteType, { label: string; color: string }> = {
  QUESTION: { label: "疑問", color: "text-purple-600 dark:text-purple-400" },
  RESOLVED: { label: "解決", color: "text-green-600 dark:text-green-400" },
  CONFUSION: { label: "混乱", color: "text-red-600 dark:text-red-400" },
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
        <span className="text-xs">
          <span
            className={`font-semibold ${NOTE_TYPE_STYLES[step.note.type].color}`}
          >
            [{NOTE_TYPE_STYLES[step.note.type].label}]
          </span>{" "}
          <span className="text-gray-600 dark:text-gray-400">
            {step.note.content}
          </span>
        </span>
      )}
    </div>
  );
}

export function SentenceList({ sentences, steps = [] }: SentenceListProps) {
  const stepsBySentence = useMemo(
    () => groupStepsBySentence(steps),
    [steps],
  );

  const bottomRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (steps.length > 0) {
      bottomRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [steps.length]);

  return (
    <ol className="flex flex-col gap-2">
      {sentences.map((sentence) => {
        const relatedSteps = stepsBySentence.get(sentence.index);
        return (
          <li
            key={sentence.index}
            className="flex gap-3 p-3 rounded border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800"
          >
            <span className="shrink-0 w-8 h-8 flex items-center justify-center rounded-full bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 text-sm font-bold">
              {sentence.index + 1}
            </span>
            <div className="flex-1 min-w-0">
              <p className="pt-1">{sentence.content}</p>
              {relatedSteps && relatedSteps.length > 0 && (
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
      <div ref={bottomRef} />
    </ol>
  );
}
