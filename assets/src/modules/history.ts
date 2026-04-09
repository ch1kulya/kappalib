import {
  getProgressCookie,
  saveProgressCookie,
  NovelProgress,
} from "./progress";

const API_URL = process.env.API_URL;

interface NovelData {
  id: string;
  title: string;
  author: string;
  cover_url: string | null;
  chapter_count: number;
}

interface HistoryItem {
  novelId: string;
  title: string;
  author: string;
  coverUrl: string;
  chapterNum: number;
  totalChapters: number;
  readAt: number;
}

async function fetchNovelsBatch(ids: string[]): Promise<Map<string, NovelData>> {
  const map = new Map<string, NovelData>();
  if (ids.length === 0) return map;

  try {
    const res = await fetch(`${API_URL}/novels/batch`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ ids }),
    });
    if (res.ok) {
      const novels: NovelData[] = await res.json();
      for (const novel of novels) {
        map.set(novel.id, novel);
      }
    }
  } catch {
    // ignore
  }
  return map;
}

export function initHistoryPage(): void {
  const historyList = document.getElementById("history-list");
  const historyEmpty = document.getElementById("history-empty");
  const historySkeleton = document.getElementById("history-skeleton");
  const clearAllBtn = document.getElementById("history-clear-all");

  if (!historyList || !historyEmpty) return;

  const progress = getProgressCookie();
  const novelIds = Object.keys(progress.novels || {});

  if (novelIds.length === 0) {
    historyEmpty.style.display = "block";
    return;
  }

  const skeletonTemplate = document.getElementById(
    "tpl-history-skeleton",
  ) as HTMLTemplateElement;
  if (historySkeleton && skeletonTemplate) {
    for (let i = 0; i < novelIds.length; i++) {
      historySkeleton.appendChild(
        skeletonTemplate.content.cloneNode(true),
      );
    }
  }

  const hideSkeleton = (): void => {
    if (historySkeleton) historySkeleton.style.display = "none";
  };

  let loadedItems: HistoryItem[] = [];

  const renderHistory = async (): Promise<void> => {
    const items: HistoryItem[] = [];

    const novelDataMap = await fetchNovelsBatch(novelIds);

    for (const [novelId, novelProgress] of Object.entries(
      progress.novels as Record<string, NovelProgress>,
    )) {
      const novelData = novelDataMap.get(novelId);
      items.push({
        novelId,
        title: novelData?.title || novelId,
        author: novelData?.author || "",
        coverUrl: novelData?.cover_url || "",
        chapterNum: novelProgress.chapterNum,
        totalChapters: novelData?.chapter_count || 0,
        readAt: novelProgress.readAt,
      });
    }

    items.sort((a, b) => b.readAt - a.readAt);
    loadedItems = items;

    hideSkeleton();
    if (clearAllBtn) clearAllBtn.style.display = "block";
    historyList.innerHTML = "";
    const template = document.getElementById(
      "tpl-history-item",
    ) as HTMLTemplateElement;
    if (!template) return;

    for (const item of items) {
      const fragment = template.content.cloneNode(true) as DocumentFragment;
      const itemEl = fragment.querySelector(".history-item") as HTMLElement;
      if (!itemEl) continue;

      itemEl.dataset.novelId = item.novelId;

      const posterLink = itemEl.querySelector(
        ".history-poster-link",
      ) as HTMLAnchorElement;
      const titleLink = itemEl.querySelector(
        ".history-title",
      ) as HTMLAnchorElement;
      const img = itemEl.querySelector("img") as HTMLImageElement;
      const titleEl = itemEl.querySelector(
        '[data-field="title"]',
      ) as HTMLElement;
      const authorEl = itemEl.querySelector(
        '[data-field="author"]',
      ) as HTMLElement;
      const progressTextEl = itemEl.querySelector(
        '[data-field="progress"]',
      ) as HTMLElement;
      const percentEl = itemEl.querySelector(
        '[data-field="percent"]',
      ) as HTMLElement;
      const progressBar = itemEl.querySelector(
        '[data-field="progressBar"]',
      ) as HTMLElement;
      const removeBtn = itemEl.querySelector(
        ".history-remove",
      ) as HTMLButtonElement;

      const novelUrl = `/${item.novelId}`;
      posterLink.href = novelUrl;
      titleLink.href = novelUrl;

      if (item.coverUrl) {
        img.src = item.coverUrl;
        img.alt = item.title;
        const wrapper = itemEl.querySelector(".poster-wrapper") as HTMLElement;
        if (wrapper) {
          wrapper.style.setProperty("--bg-url", `url(${item.coverUrl})`);
        }
      }

      if (titleEl) titleEl.textContent = item.title;
      if (authorEl) authorEl.textContent = item.author || "";

      if (item.totalChapters > 0) {
        let percent = Math.round(
          (item.chapterNum / item.totalChapters) * 100,
        );
        if (percent < 1) percent = 1;
        if (percent > 100) percent = 100;
        if (progressTextEl) {
          progressTextEl.innerHTML = `Глава <strong>${item.chapterNum}</strong> из ${item.totalChapters}`;
        }
        if (percentEl) percentEl.textContent = `${percent}%`;
        if (progressBar) progressBar.style.width = `${percent}%`;
      } else {
        if (progressTextEl) {
          progressTextEl.innerHTML = `Глава <strong>${item.chapterNum}</strong>`;
        }
        if (percentEl) percentEl.textContent = "";
        if (progressBar) progressBar.style.width = "0%";
      }

      removeBtn.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        removeHistoryItem(item.novelId);
        itemEl.remove();
        checkEmpty();
      });

      historyList.appendChild(fragment);
    }
  };

  const removeHistoryItem = (novelId: string): void => {
    const progress = getProgressCookie();
    if (progress.novels[novelId]) {
      delete progress.novels[novelId];
    }
    if (progress.lastRead?.novelId === novelId) {
      const remaining = loadedItems.filter((item) => item.novelId !== novelId);
      if (remaining.length > 0) {
        const next = remaining[0];
        const nextProgress = progress.novels[next.novelId];
        if (nextProgress) {
          progress.lastRead = {
            novelId: next.novelId,
            title: next.title,
            author: next.author,
            coverUrl: next.coverUrl,
            chapterId: nextProgress.chapterId,
            chapterNum: next.chapterNum,
            totalChapters: next.totalChapters,
            readAt: Date.now(),
          };
        } else {
          progress.lastRead = null;
        }
      } else {
        progress.lastRead = null;
      }
    }
    loadedItems = loadedItems.filter((item) => item.novelId !== novelId);
    saveProgressCookie(progress);
  };

  const checkEmpty = (): void => {
    const progress = getProgressCookie();
    const hasItems = Object.keys(progress.novels || {}).length > 0;
    historyEmpty.style.display = hasItems ? "none" : "block";
    if (clearAllBtn) clearAllBtn.style.display = hasItems ? "block" : "none";
  };

  const clearAllHistory = (): void => {
    const progress = getProgressCookie();
    progress.novels = {};
    progress.lastRead = null;
    saveProgressCookie(progress);
    historyList.innerHTML = "";
    checkEmpty();
  };

  clearAllBtn?.addEventListener("click", () => {
    if (confirm("Очистить всю историю чтения?")) {
      clearAllHistory();
    }
  });

  renderHistory();
}

export function refreshHistoryProgress(): void {
  const historyList = document.getElementById("history-list");
  if (!historyList) return;

  const progressBars = historyList.querySelectorAll(".history-item");
  if (progressBars.length === 0) return;

  progressBars.forEach((item) => {
    const itemEl = item as HTMLElement;
    const novelId = itemEl.dataset.novelId;
    if (!novelId) return;

    const progressTextEl = itemEl.querySelector('[data-field="progress"]');
    const percentEl = itemEl.querySelector('[data-field="percent"]');
    const progressBar = itemEl.querySelector('[data-field="progressBar"]');

    fetch(`${process.env.API_URL}/novels/${novelId}`)
      .then((res) => res.ok ? res.json() : null)
      .then((novel) => {
        if (!novel) return;
        const data = getProgressCookie();
        const novelProgress = data.novels?.[novelId];
        if (!novelProgress) return;

        const totalChapters = novel.chapter_count || 0;
        const chapterNum = novelProgress.chapterNum;

        if (totalChapters > 0) {
          let percent = Math.round((chapterNum / totalChapters) * 100);
          if (percent < 1) percent = 1;
          if (percent > 100) percent = 100;

          if (progressTextEl) {
            progressTextEl.innerHTML = `Глава <strong>${chapterNum}</strong> из ${totalChapters}`;
          }
          if (percentEl) {
            percentEl.textContent = `${percent}%`;
          }
          if (progressBar) {
            (progressBar as HTMLElement).style.width = `${percent}%`;
          }
        } else {
          if (progressTextEl) {
            progressTextEl.innerHTML = `Глава <strong>${chapterNum}</strong>`;
          }
          if (percentEl) {
            percentEl.textContent = "";
          }
          if (progressBar) {
            (progressBar as HTMLElement).style.width = "0%";
          }
        }
      })
      .catch(() => {});
  });
}
