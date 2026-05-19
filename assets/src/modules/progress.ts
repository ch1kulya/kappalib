import { setKappalibCookie, updateKappalibCookieQuietly } from "./profile";

const API_URL = process.env.API_URL;

export interface NovelProgress {
  chapterId: string;
  chapterNum: number;
  readAt: number;
}

export interface LastReadData {
  novelId: string;
  title: string;
  author: string;
  coverUrl: string;
  chapterId: string;
  chapterNum: number;
  totalChapters: number;
  readAt: number;
}

export interface ProgressCookie {
  novels: Record<string, NovelProgress>;
  lastRead: LastReadData | null;
}

const COOKIE_NAME = "kappalib_progress";
const MAX_COOKIE_SIZE = 4000;

export function getProgressCookie(): ProgressCookie {
  const cookies = document.cookie.split(";");
  for (const cookie of cookies) {
    const [name, value] = cookie.trim().split("=");
    if (name === COOKIE_NAME && value) {
      try {
        return JSON.parse(decodeURIComponent(value));
      } catch {
        break;
      }
    }
  }
  return { novels: {}, lastRead: null };
}

function trimProgressCookie(data: ProgressCookie): string {
  let json = JSON.stringify(data);

  while (json.length > MAX_COOKIE_SIZE && Object.keys(data.novels).length > 1) {
    const entries = Object.entries(data.novels);
    entries.sort((a, b) => a[1].readAt - b[1].readAt);
    const oldestNovelId = entries[0][0];

    if (data.lastRead?.novelId === oldestNovelId) {
      if (entries.length > 1) {
        delete data.novels[entries[1][0]];
      } else {
        break;
      }
    } else {
      delete data.novels[oldestNovelId];
    }

    json = JSON.stringify(data);
  }

  return json;
}

export function saveProgressCookie(data: ProgressCookie): void {
  setKappalibCookie(COOKIE_NAME, trimProgressCookie(data));
}

function saveProgressCookieQuietly(data: ProgressCookie): void {
  updateKappalibCookieQuietly(COOKIE_NAME, trimProgressCookie(data));
}

export function initReadingProgressSaver(): void {
  const tracker = document.getElementById("reading-tracker");
  const TIMER_DELAY = 60000;

  if (!tracker) return;

  const novelId = tracker.dataset.novelId;
  const currentChapterId = tracker.dataset.chapterId;
  const currentChapterNumStr = tracker.dataset.chapterNum;
  const novelTitle = tracker.dataset.novelTitle || "";
  const novelAuthor = tracker.dataset.novelAuthor || "";
  const novelCover = tracker.dataset.novelCover || "";
  const totalChaptersStr = tracker.dataset.totalChapters || "0";

  if (!novelId || !currentChapterId) return;

  const currentChapterNum = parseInt(currentChapterNumStr || "0", 10);
  const totalChapters = parseInt(totalChaptersStr, 10);

  const data = getProgressCookie();
  if (
    data.lastRead &&
    data.lastRead.novelId === novelId &&
    data.lastRead.totalChapters < totalChapters
  ) {
    data.lastRead.totalChapters = totalChapters;
    saveProgressCookieQuietly(data);
  }

  const saveProgress = (
    targetChapterId: string,
    targetChapterNum: number,
  ): void => {
    try {
      const data = getProgressCookie();
      const existing = data.novels[novelId];

      if (existing && existing.chapterNum > targetChapterNum) {
        return;
      }

      if (existing && existing.chapterId === targetChapterId) {
        return;
      }

      const now = Date.now();

      data.novels[novelId] = {
        chapterId: targetChapterId,
        chapterNum: targetChapterNum,
        readAt: now,
      };

      data.lastRead = {
        novelId,
        title: novelTitle,
        author: novelAuthor,
        coverUrl: novelCover,
        chapterId: targetChapterId,
        chapterNum: targetChapterNum,
        totalChapters,
        readAt: now,
      };

      saveProgressCookie(data);
      console.info(`Progress saved: Novel ${novelId}, Ch ${targetChapterNum}`);
    } catch (e) {
      console.error("Failed to save reading progress", e);
    }
  };

  const timerId = window.setTimeout(() => {
    saveProgress(currentChapterId, currentChapterNum);
  }, TIMER_DELAY);

  const nextButtons =
    document.querySelectorAll<HTMLElement>(".js-next-chapter");

  nextButtons.forEach((btn) => {
    btn.addEventListener("click", () => {
      const nextId = btn.dataset.chapterId;
      const nextNumStr = btn.dataset.chapterNum;

      if (nextId && nextNumStr) {
        const nextNum = parseInt(nextNumStr, 10);
        if (nextNum > 0) {
          saveProgress(nextId, nextNum);
          clearTimeout(timerId);
        }
      }
    });
  });

  window.addEventListener("beforeunload", () => {
    clearTimeout(timerId);
  });

  setupKeyboardNavigation();
}

function setupKeyboardNavigation(): void {
  const nav = document.querySelector<HTMLElement>(".chapter-navigation");
  if (!nav) return;

  document.addEventListener("keydown", (e: KeyboardEvent) => {
    if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;

    const prevLink = nav.querySelector<HTMLAnchorElement>("a:not(.js-next-chapter)");
    const nextLink = nav.querySelector<HTMLAnchorElement>("a.js-next-chapter");

    if (e.key === "ArrowLeft" && prevLink) {
      prevLink.click();
    } else if (e.key === "ArrowRight" && nextLink) {
      nextLink.click();
    }
  });
}

export async function refreshLastReadTotalChapters(): Promise<void> {
  const data = getProgressCookie();
  if (!data.lastRead) return;

  try {
    const res = await fetch(`${API_URL}/novels/${data.lastRead.novelId}`);
    if (!res.ok) {
      updateCRCard(data.lastRead);
      return;
    }

    const novel = await res.json();
    const newTotalChapters = novel.chapter_count;

    if (newTotalChapters && newTotalChapters > data.lastRead.totalChapters) {
      data.lastRead.totalChapters = newTotalChapters;
      saveProgressCookieQuietly(data);
    }

    updateCRCard(data.lastRead);
  } catch {
    updateCRCard(data.lastRead);
  }
}

function updateCRCard(lastRead: LastReadData): void {
  const wrapper = document.querySelector(".cr-wrapper");
  if (!wrapper) return;

  const posterLink = wrapper.querySelector(".cr-poster-link") as HTMLAnchorElement;
  const posterImg = wrapper.querySelector(".cr-poster-link img") as HTMLImageElement;
  const posterWrapper = wrapper.querySelector(".cr-poster-link .poster-wrapper") as HTMLElement;
  const titleLink = wrapper.querySelector(".cr-title") as HTMLAnchorElement;
  const title = wrapper.querySelector(".cr-title h3") as HTMLElement;
  const author = wrapper.querySelector(".cr-meta span") as HTMLElement;
  const continueBtn = wrapper.querySelector(".cr-head-row .action-btn") as HTMLAnchorElement;

  if (posterLink) {
    posterLink.href = `/${lastRead.novelId}`;
  }
  if (posterImg) {
    posterImg.src = lastRead.coverUrl;
    posterImg.alt = lastRead.title;
  }
  if (posterWrapper) {
    posterWrapper.style.setProperty("--bg-url", `url(${lastRead.coverUrl})`);
  }
  if (titleLink) {
    titleLink.href = `/${lastRead.novelId}`;
  }
  if (title) {
    title.textContent = lastRead.title;
  }
  if (author) {
    author.textContent = lastRead.author;
  }
  if (continueBtn) {
    continueBtn.href = `/${lastRead.novelId}/chapter/${lastRead.chapterId}`;
  }

  const totalChapters = lastRead.totalChapters;
  const chapterNum = lastRead.chapterNum;

  const progressText = wrapper.querySelector(".progress-text");
  const progressPercent = wrapper.querySelector(".progress-percent");
  const progressBarFill = wrapper.querySelector(".progress-bar-fill");

  if (progressText) {
    progressText.innerHTML = `Глава <strong>${chapterNum}</strong> из ${totalChapters}`;
  }

  if (totalChapters > 0) {
    let percent = Math.round((chapterNum / totalChapters) * 100);
    if (percent < 1) percent = 1;
    if (percent > 100) percent = 100;

    if (progressPercent) {
      progressPercent.textContent = `${percent}%`;
    }
    if (progressBarFill) {
      (progressBarFill as HTMLElement).style.width = `${percent}%`;
    }
  }
}
