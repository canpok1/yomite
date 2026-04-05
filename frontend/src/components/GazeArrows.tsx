import { useCallback, useEffect, useRef, useState } from "react";
import type { SimulationStep } from "../types";

interface GazeArrowsProps {
  steps: SimulationStep[];
  // NOTE: このrefはスクロールコンテナを指す必要がある。
  // 矢印のY座標計算にcontainer.scrollTopを使用するため。
  containerRef: React.RefObject<HTMLElement | null>;
}

type ArrowType = "forward" | "backward" | "reread" | "done";

interface ArrowData {
  stepNumber: number;
  type: ArrowType;
  fromY: number;
  toY: number;
}

interface ArrowState {
  arrows: ArrowData[];
  height: number;
}

const ARROW_COLORS: Record<ArrowType, string> = {
  forward: "#3b82f6",
  backward: "#f59e0b",
  reread: "#8b5cf6",
  done: "#10b981",
};

const SVG_WIDTH = 64;
const ARROW_X = 24;

function getArrowType(step: SimulationStep): ArrowType {
  if (step.next_index === null) return "done";
  if (step.next_index === step.current_index) return "reread";
  if (step.next_index > step.current_index) return "forward";
  return "backward";
}

function getSentenceElement(
  container: HTMLElement,
  index: number,
): HTMLElement | null {
  return container.querySelector(`[data-sentence-index="${index}"]`);
}

function getElementCenterY(el: HTMLElement, containerRect: DOMRect): number {
  const rect = el.getBoundingClientRect();
  return rect.top + rect.height / 2 - containerRect.top;
}

export function GazeArrows({ steps, containerRef }: GazeArrowsProps) {
  const [state, setState] = useState<ArrowState>({ arrows: [], height: 0 });
  // NOTE: recalcRef経由でrecalculateを参照することで、ResizeObserverのセットアップを
  // stepsの変更のたびに再実行せずに、常に最新のrecalculateを呼べるようにしている。
  const recalcRef = useRef<() => void>(null);

  const recalculate = useCallback(() => {
    const container = containerRef.current;
    if (!container) return;

    const containerRect = container.getBoundingClientRect();
    const elCache = new Map<number, HTMLElement | null>();
    const getEl = (index: number) => {
      if (!elCache.has(index)) {
        elCache.set(index, getSentenceElement(container, index));
      }
      return elCache.get(index) ?? null;
    };

    const arrows: ArrowData[] = [];
    for (const step of steps) {
      const type = getArrowType(step);
      // NOTE: forward矢印は非表示にする。
      // 読書中の視線移動はほとんどが前進であり、表示すると矢印が大量になって見づらいため。
      if (type === "forward") continue;

      const fromEl = getEl(step.current_index);
      if (!fromEl) continue;

      const fromY =
        getElementCenterY(fromEl, containerRect) + container.scrollTop;

      if (type === "done") {
        arrows.push({ stepNumber: step.step, type, fromY, toY: fromY });
        continue;
      }

      const toEl = getEl(step.next_index!);
      if (!toEl) continue;

      const toY = getElementCenterY(toEl, containerRect) + container.scrollTop;
      arrows.push({ stepNumber: step.step, type, fromY, toY });
    }
    setState({ arrows, height: container.scrollHeight });
  }, [steps, containerRef]);

  useEffect(() => {
    recalcRef.current = recalculate;
    recalculate();
  }, [recalculate]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const observer = new ResizeObserver(() => {
      recalcRef.current?.();
    });
    observer.observe(container);
    return () => observer.disconnect();
  }, [containerRef]);

  if (state.arrows.length === 0) return null;

  return (
    <svg
      className="absolute top-0 right-0 pointer-events-none"
      width={SVG_WIDTH}
      height={state.height}
      style={{ overflow: "visible" }}
    >
      <defs>
        {(Object.keys(ARROW_COLORS) as ArrowType[]).map((type) => (
          <marker
            key={type}
            id={`arrowhead-${type}`}
            markerWidth="8"
            markerHeight="6"
            refX="8"
            refY="3"
            orient="auto"
          >
            <polygon points="0 0, 8 3, 0 6" fill={ARROW_COLORS[type]} />
          </marker>
        ))}
      </defs>

      {state.arrows.map((arrow) => {
        const color = ARROW_COLORS[arrow.type];

        if (arrow.type === "done") {
          return (
            <rect
              key={arrow.stepNumber}
              x={ARROW_X - 5}
              y={arrow.fromY - 5}
              width={10}
              height={10}
              fill={color}
              rx={2}
            />
          );
        }

        let d: string;
        if (arrow.type === "reread") {
          const loopR = 12;
          const cx = ARROW_X + loopR * 2;
          const cy1 = arrow.fromY - loopR;
          const cy2 = arrow.fromY + loopR;
          d = `M ${ARROW_X} ${arrow.fromY} C ${cx} ${cy1} ${cx} ${cy2} ${ARROW_X} ${arrow.fromY}`;
        } else {
          const controlPointX =
            ARROW_X + Math.min(Math.abs(arrow.toY - arrow.fromY) * 0.3, 30);
          d = `M ${ARROW_X} ${arrow.fromY} C ${controlPointX} ${arrow.fromY} ${controlPointX} ${arrow.toY} ${ARROW_X} ${arrow.toY}`;
        }

        return (
          <path
            key={arrow.stepNumber}
            d={d}
            fill="none"
            stroke={color}
            strokeWidth={1.5}
            markerEnd={`url(#arrowhead-${arrow.type})`}
            opacity={0.7}
          />
        );
      })}
    </svg>
  );
}
