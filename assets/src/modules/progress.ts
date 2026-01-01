import { setKappalibCookie } from "./profile";

interface HistoryItem {
  id: string;
  num: number;
  updatedAt: number;
}

interface ProgressHistory {
  [novelId: string]: HistoryItem;
}

export function initReadingProgressSaver(): void {
  const STORAGE_KEY = "kappalib_progress";
  const tracker = document.getElementById("reading-tracker");
  const TIMER_DELAY = 60000;

  if (!tracker) return;

  const novelId = tracker.dataset.novelId;
  const currentChapterId = tracker.dataset.chapterId;
  const currentChapterNumStr = tracker.dataset.chapterNum;

  if (!novelId || !currentChapterId) return;

  const currentChapterNum = parseInt(currentChapterNumStr || "0", 10);

  const saveProgress = (
    targetChapterId: string,
    targetChapterNum: number,
  ): void => {
    try {
      const rawHistory = localStorage.getItem(STORAGE_KEY);
      const history: ProgressHistory = rawHistory ? JSON.parse(rawHistory) : {};

      const savedData = history[novelId] || { id: "", num: -1, updatedAt: 0 };

      if (savedData.num > targetChapterNum) {
        return;
      }

      if (savedData.id === targetChapterId) return;

      history[novelId] = {
        id: targetChapterId,
        num: targetChapterNum,
        updatedAt: Date.now(),
      };

      localStorage.setItem(STORAGE_KEY, JSON.stringify(history));
      setKappalibCookie(`prog_${novelId}`, targetChapterId);
      setKappalibCookie("last_read", novelId);

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
}
