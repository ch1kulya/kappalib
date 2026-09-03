declare global {
  interface Window {
    umami?: {
      identify: (id: string) => void;
      track: (name: string, data?: Record<string, any>) => void;
    };
  }
  interface Document {
    prerendering?: boolean;
  }
}

let lastIdentifiedId: string | null = null;
let pendingIdentifyId: string | null = null;
let pendingEvents: Array<{ name: string; data?: Record<string, any> }> = [];

function flushPending(): void {
  if (!window.umami) return;

  if (pendingIdentifyId && window.umami.identify) {
    if (pendingIdentifyId !== lastIdentifiedId) {
      lastIdentifiedId = pendingIdentifyId;
      window.umami.identify(pendingIdentifyId);
    }
    pendingIdentifyId = null;
  }

  if (window.umami.track && pendingEvents.length > 0) {
    const events = [...pendingEvents];
    pendingEvents = [];
    for (const evt of events) {
      window.umami.track(evt.name, evt.data);
    }
  }
}

if (typeof window !== "undefined") {
  window.addEventListener("umami:ready", flushPending);
  document.addEventListener("prerenderingchange", () => {
    const checkInterval = setInterval(() => {
      if (window.umami?.identify) {
        flushPending();
        clearInterval(checkInterval);
      }
    }, 200);
    setTimeout(() => clearInterval(checkInterval), 10000);
  });
}

export function trackEvent(name: string, data?: Record<string, any>): void {
  if (typeof window === "undefined") return;
  if (window.umami?.track) {
    window.umami.track(name, data);
    return;
  }
  pendingEvents.push({ name, data });
}

export function identifyUmami(id: string): void {
  if (typeof window === "undefined" || !id || id === lastIdentifiedId) return;
  pendingIdentifyId = id;

  if (window.umami?.identify) {
    lastIdentifiedId = id;
    pendingIdentifyId = null;
    window.umami.identify(id);
    return;
  }

  const interval = setInterval(() => {
    if (window.umami?.identify) {
      flushPending();
      clearInterval(interval);
    }
  }, 300);

  setTimeout(() => clearInterval(interval), 10000);
}

export function clearIdentifiedUmami(): void {
  lastIdentifiedId = null;
  pendingIdentifyId = null;
}
