import { useEffect, useRef } from "react";
import type { NoteType, Sentence, SimulationStep } from "../types";

interface StepListProps {
  steps: SimulationStep[];
  sentences: Sentence[];
}

const NOTE_TYPE_STYLES: Record<NoteType, { label: string; color: string }> = {
  QUESTION: { label: "疑問", color: "text-purple-600 dark:text-purple-400" },
  RESOLVED: { label: "解決", color: "text-green-600 dark:text-green-400" },
  CONFUSION: { label: "混乱", color: "text-red-600 dark:text-red-400" },
};

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

function getDirection(step: SimulationStep): Direction {
  if (step.next_index === null) return "done";
  if (step.next_index > step.current_index) return "forward";
  return "backward";
}

export function StepList({ steps, sentences }: StepListProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [steps.length]);

  if (steps.length === 0) return null;

  return (
    <div className="flex flex-col gap-3">
      {steps.map((step) => {
        const sentence = sentences[step.current_index];
        return (
          <div
            key={step.step}
            className="p-3 rounded border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800"
          >
            <div className="flex items-center gap-2 mb-2">
              <span className="shrink-0 w-7 h-7 flex items-center justify-center rounded-full bg-gray-200 dark:bg-gray-700 text-sm font-bold">
                {step.step}
              </span>
              <span
                className={`px-2 py-0.5 rounded text-xs font-semibold ${DIRECTION_STYLES[getDirection(step)].color}`}
              >
                {DIRECTION_STYLES[getDirection(step)].label}
              </span>
              <span className="text-xs text-gray-500 dark:text-gray-400">
                文 {step.current_index + 1}
                {step.next_index !== null && ` → 文 ${step.next_index + 1}`}
              </span>
            </div>

            {sentence && (
              <p className="text-sm mb-2 pl-9 text-gray-700 dark:text-gray-300">
                {sentence.content}
              </p>
            )}

            {step.note && (
              <div className="pl-9">
                <span
                  className={`text-xs font-semibold ${NOTE_TYPE_STYLES[step.note.type].color}`}
                >
                  [{NOTE_TYPE_STYLES[step.note.type].label}]
                </span>{" "}
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  {step.note.content}
                </span>
              </div>
            )}
          </div>
        );
      })}
      <div ref={bottomRef} />
    </div>
  );
}
