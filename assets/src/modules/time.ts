import { profileManager } from "./profile";

const API_URL = process.env.API_URL;
const INACTIVITY_TIMEOUT_MS = 45000;
const FLUSH_INTERVAL_SECONDS = 60;
const MIN_FLUSH_SECONDS = 5;

class ActiveTimeTracker {
  private activeSeconds = 0;
  private lastActivityTime = Date.now();
  private lastFlushTime = 0;
  private lastUnloadFlushTime = 0;
  private backoffUntil = 0;
  private isFlushing = false;
  private intervalId: number | null = null;

  start(): void {
    if (typeof window === "undefined" || this.intervalId !== null) return;

    this.lastActivityTime = Date.now();
    this.bindEvents();

    this.intervalId = window.setInterval(() => {
      this.tick();
    }, 1000);
  }

  stop(): void {
    if (this.intervalId !== null) {
      window.clearInterval(this.intervalId);
      this.intervalId = null;
    }
  }

  private bindEvents(): void {
    const onActivity = () => {
      this.lastActivityTime = Date.now();
    };

    window.addEventListener("mousemove", onActivity, { passive: true });
    window.addEventListener("mousedown", onActivity, { passive: true });
    window.addEventListener("keydown", onActivity, { passive: true });
    window.addEventListener("scroll", onActivity, { passive: true });
    window.addEventListener("touchstart", onActivity, { passive: true });

    document.addEventListener("visibilitychange", () => {
      if (document.visibilityState === "hidden") {
        this.flush();
      } else if (document.visibilityState === "visible") {
        this.lastActivityTime = Date.now();
      }
    });

    window.addEventListener("pagehide", () => {
      this.flush(true);
    });

    window.addEventListener("beforeunload", () => {
      this.flush(true);
    });
  }

  private tick(): void {
    if (document.visibilityState !== "visible") return;

    const isUserActive =
      Date.now() - this.lastActivityTime < INACTIVITY_TIMEOUT_MS;
    if (isUserActive) {
      this.activeSeconds = Math.min(this.activeSeconds + 1, 300);
      if (this.activeSeconds >= FLUSH_INTERVAL_SECONDS) {
        this.flush();
      }
    }
  }

  async flush(isUnload = false): Promise<void> {
    if (this.activeSeconds <= 0 || this.isFlushing) return;

    const now = Date.now();
    if (now < this.backoffUntil) return;

    if (isUnload) {
      if (now - this.lastUnloadFlushTime < 2000) return;
      this.lastUnloadFlushTime = now;
    } else {
      if (this.activeSeconds < MIN_FLUSH_SECONDS) return;
      if (now - this.lastFlushTime < 10000) return;
    }

    if (!profileManager.isLoggedIn()) {
      this.activeSeconds = 0;
      return;
    }

    const secondsToSend = this.activeSeconds;
    this.activeSeconds = 0;
    this.lastFlushTime = now;
    this.isFlushing = true;

    try {
      const res = await fetch(`${API_URL}/stats/time`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ seconds: secondsToSend }),
        credentials: "include",
        keepalive: isUnload,
      });

      if (res.status === 429) {
        this.backoffUntil = Date.now() + 60000;
        if (!isUnload) {
          this.activeSeconds = Math.min(this.activeSeconds + secondsToSend, 300);
        }
      } else if (res.status >= 500) {
        this.backoffUntil = Date.now() + 30000;
        if (!isUnload) {
          this.activeSeconds = Math.min(this.activeSeconds + secondsToSend, 300);
        }
      }
    } catch {
      this.backoffUntil = Date.now() + 30000;
      if (!isUnload) {
        this.activeSeconds = Math.min(this.activeSeconds + secondsToSend, 300);
      }
    } finally {
      this.isFlushing = false;
    }
  }
}

export const activeTimeTracker = new ActiveTimeTracker();

export function initTimeTracker(): void {
  activeTimeTracker.start();
}
