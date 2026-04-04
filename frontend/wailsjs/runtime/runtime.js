// @ts-check
// Wails runtime (placeholder for development).
// At build time, `wails build` provides the real runtime.

export function EventsOn(eventName, callback) {
  if (window['runtime'] && window['runtime']['EventsOn']) {
    return window['runtime']['EventsOn'](eventName, callback);
  }
  // NOTE: Wailsランタイム未注入時（開発環境でのスタンドアロン実行時）はno-opで返す。
  // wails dev/build時は本物のruntimeが注入されるため、ここには到達しない。
  return () => {};
}

export function EventsOff(eventName, ...additionalEventNames) {
  if (window['runtime'] && window['runtime']['EventsOff']) {
    window['runtime']['EventsOff'](eventName, ...additionalEventNames);
  }
}
