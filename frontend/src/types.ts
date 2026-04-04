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
