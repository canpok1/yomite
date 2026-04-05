import { useCallback, useEffect, useRef, useState } from "react";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import {
  StartSimulation,
  StopSimulation,
} from "../../wailsjs/go/gui/App";
import type { SimulationStep } from "../types";

export type SimulationStatus = "idle" | "running" | "completed" | "error";

interface UseSimulationReturn {
  status: SimulationStatus;
  steps: SimulationStep[];
  error: string | null;
  start: (rawText: string, providerID: string, personaID: string) => Promise<void>;
  stop: () => Promise<void>;
  reset: () => void;
}

export function useSimulation(): UseSimulationReturn {
  const [status, setStatus] = useState<SimulationStatus>("idle");
  const [steps, setSteps] = useState<SimulationStep[]>([]);
  const [error, setError] = useState<string | null>(null);
  // NOTE: stopコールバックはuseCallbackで固定されるため、stateのstatusを
  // キャプチャするとstaleになる。refで最新値を参照する。
  const statusRef = useRef<SimulationStatus>("idle");

  useEffect(() => {
    statusRef.current = status;
  }, [status]);

  useEffect(() => {
    const offStep = EventsOn("simulation:step", (data: SimulationStep) => {
      setSteps((prev) => [...prev, data]);
    });

    const offDone = EventsOn("simulation:done", () => {
      setStatus("completed");
    });

    const offError = EventsOn("simulation:error", (message: string) => {
      setError(message);
      setStatus("error");
    });

    return () => {
      offStep();
      offDone();
      offError();
    };
  }, []);

  const start = useCallback(
    async (rawText: string, providerID: string, personaID: string) => {
      setSteps([]);
      setError(null);
      setStatus("running");
      try {
        // NOTE: StartSimulationはgoroutineを起動するだけで即座に返る。
        // 実行中のエラーはsimulation:errorイベントで通知されるため、
        // ここでcatchされるのは起動前のバリデーションエラーのみ。
        await StartSimulation(rawText, providerID, personaID);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
        setStatus("error");
      }
    },
    [],
  );

  const stop = useCallback(async () => {
    try {
      await StopSimulation();
    } catch {
      // NOTE: StopSimulationの失敗はUIに影響させない。
      // 次の実行開始時にリセットされるため問題ない。
    }
    if (statusRef.current === "running") {
      setStatus("idle");
    }
  }, []);

  const reset = useCallback(() => {
    setSteps([]);
    setError(null);
    setStatus("idle");
  }, []);

  return { status, steps, error, start, stop, reset };
}
