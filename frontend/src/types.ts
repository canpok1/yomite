export type NoteType = "QUESTION" | "RESOLVED" | "CONFUSION";

export interface Sentence {
  index: number;
  content: string;
}

// NOTE: Go側のJSONタグ(json:"current_index"等)に合わせてスネークケースを使用
export interface SimulationStep {
  step: number;
  current_index: number;
  next_index: number | null;
  note: Note | null;
}

export interface Note {
  type: NoteType;
  content: string;
}

// NOTE: Go側の core.Config 構造体のJSONタグに合わせてスネークケースを使用
export interface Config {
  log: LogConfig;
  default_provider: string;
  default_persona: string;
  providers: Record<string, ProviderConfig>;
  personas: Record<string, Persona>;
}

export interface LogConfig {
  level: string;
  path: string;
}

export interface ProviderConfig {
  type: string;
  model: string;
  origin: string;
}

export interface Persona {
  display_name: string;
  system_prompt: string;
  memory_capacity: number;
  max_steps: number;
}
