declare global {
  interface Window {
    umami?: {
      identify: (id: string) => void;
      track: (name: string, data?: Record<string, any>) => void;
    };
  }
}

export function trackEvent(name: string, data?: Record<string, any>): void {
  if (typeof window === "undefined") return;
  if (window.umami?.track) {
    window.umami.track(name, data);
  }
}

export function identifyUmami(id: string): void {
  if (typeof window === "undefined" || !id) return;
  if (window.umami?.identify) {
    window.umami.identify(id);
    return;
  }
  let attempts = 0;
  const interval = setInterval(() => {
    attempts++;
    if (window.umami?.identify) {
      window.umami.identify(id);
      clearInterval(interval);
    } else if (attempts >= 10) {
      clearInterval(interval);
    }
  }, 500);
}
