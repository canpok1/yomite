// @ts-check
// Wails auto-generated bindings (placeholder for development).
// At build time, `wails build` regenerates these from Go struct methods.

export function LoadDocument(rawText) {
  return window['go']['main']['App']['LoadDocument'](rawText);
}

export function ListProviders() {
  return window['go']['main']['App']['ListProviders']();
}

export function ListPersonas() {
  return window['go']['main']['App']['ListPersonas']();
}

export function StartSimulation(rawText, providerID, personaID) {
  return window['go']['main']['App']['StartSimulation'](rawText, providerID, personaID);
}

export function StopSimulation() {
  return window['go']['main']['App']['StopSimulation']();
}
