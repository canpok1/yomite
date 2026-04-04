// Wails auto-generated bindings (placeholder for development).
// At build time, `wails build` regenerates these from Go struct methods.

import type { Sentence } from "../../../src/types";

export function LoadDocument(rawText: string): Promise<Sentence[]>;

export function ListProviders(): Promise<string[]>;

export function ListPersonas(): Promise<Record<string, string>>;

export function StartSimulation(
  rawText: string,
  providerID: string,
  personaID: string,
): Promise<void>;

export function StopSimulation(): Promise<void>;
