import "@testing-library/jest-dom/vitest";
import { vi, afterEach } from "vitest";
import { cleanup } from "@testing-library/react";

// Replace the ConnectRPC client module for every suite. vitest resolves mock
// specifiers to file paths, so this also intercepts the components' relative
// imports ("../api/client" etc.), and it must happen before any component
// import: src/api/oplog.ts and src/state/peerStates.ts open streaming
// connections at import time against the real client.
vi.mock("@/api/client", () => import("./mocks/client"));

// --- jsdom polyfills -------------------------------------------------------

if (!window.matchMedia) {
  window.matchMedia = (query: string): MediaQueryList =>
    ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }) as MediaQueryList;
}

if (!window.ResizeObserver) {
  window.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver;
}

if (!window.IntersectionObserver) {
  window.IntersectionObserver = class {
    root = null;
    rootMargin = "";
    thresholds = [];
    observe() {}
    unobserve() {}
    disconnect() {}
    takeRecords() {
      return [];
    }
  } as unknown as typeof IntersectionObserver;
}

Element.prototype.scrollIntoView ??= () => {};
Element.prototype.scrollTo ??= (() => {}) as typeof Element.prototype.scrollTo;
// Zag.js (Chakra v3 Select/Menu/Dialog) uses pointer capture APIs jsdom lacks.
Element.prototype.hasPointerCapture ??= () => false;
Element.prototype.setPointerCapture ??= () => {};
Element.prototype.releasePointerCapture ??= () => {};

// jsdom logs "Not implemented: navigation" errors when components call
// window.location.reload() (login/logout paths); location is unforgeable in
// jsdom so it can't be stubbed — silence just that noise.
const realConsoleError = console.error.bind(console);
console.error = (...args: unknown[]) => {
  if (
    typeof args[0] === "string" &&
    args[0].includes("Not implemented: navigation")
  ) {
    return;
  }
  if (
    args[0] instanceof Error &&
    args[0].message.includes("Not implemented: navigation")
  ) {
    return;
  }
  realConsoleError(...args);
};

// --- per-test teardown -----------------------------------------------------

afterEach(async () => {
  cleanup();
  localStorage.clear();
  sessionStorage.clear();
  // Drain the module-level Chakra toast store so toasts don't leak across tests.
  const { toaster } = await import("@/components/ui/toaster");
  toaster.dismiss();
  vi.clearAllMocks();
  const { resetClientMocks } = await import("./mocks/client");
  resetClientMocks();
});
