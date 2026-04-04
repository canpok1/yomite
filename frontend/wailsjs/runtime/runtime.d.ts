// Wails runtime type declarations (placeholder for development).
// At build time, `wails build` provides the real runtime.

export function EventsOn(
  eventName: string,
  callback: (...data: any[]) => void,
): () => void;

export function EventsOff(eventName: string, ...additionalEventNames: string[]): void;
