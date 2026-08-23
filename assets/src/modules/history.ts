import {
  getProgressCookie,
  saveProgressCookie,
  saveProgressCookieQuietly,
  NovelProgress,
} from "./progress";

const API_URL = process.env.API_URL;
const NOVEL_CACHE_PREFIX = "kappalib_novel_cache_";
const memoryNovelCache = new Map<string, NovelData>();

let historySetupDone = false;
let loadedItems: HistoryItem[] = [];
let inFlightBatchPromise: Promise<Map<string, NovelData>> | null = null;
let inFlightBatchIds = "";

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

function getCachedNovel(novelId: string): NovelData | null {
  if (memoryNovelCache.has(novelId)) {
    return memoryNovelCache.get(novelId)!;
  }
  try {
    const cached = sessionStorage.getItem(`${NOVEL_CACHE_PREFIX}${novelId}`);
    if (cached) {
      const data: NovelData = JSON.parse(cached);
      memoryNovelCache.set(novelId, data);
      return data;
    }
  } catch {
    return null;
  }
  return null;
}

function setCachedNovel(novelId: string, data: NovelData): void {
  memoryNovelCache.set(novelId, data);
  try {
    sessionStorage.setItem(`${NOVEL_CACHE_PREFIX}${novelId}`, JSON.stringify(data));
  } catch {
    return;
  }
}

async function fetchNovelsBatch(
  ids: string[],
): Promise<Map<string, NovelData>> {
  const map = new Map<string, NovelData>();
  if (ids.length === 0) return map;

  const sortedKey = [...ids].sort().join(",");
  if (inFlightBatchPromise && inFlightBatchIds === sortedKey) {
    return inFlightBatchPromise;
  }

  inFlightBatchIds = sortedKey;
  inFlightBatchPromise = (async () => {
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
          setCachedNovel(novel.id, novel);
        }
      }
    } catch {
      return map;
    } finally {
      inFlightBatchPromise = null;
      inFlightBatchIds = "";
    }
    return map;
  })();

  return inFlightBatchPromise;
}

function createItemElement(
  item: HistoryItem,
  template: HTMLTemplateElement,
  onRemove: (novelId: string, itemEl: HTMLElement) => void,
): HTMLElement | null {
  const fragment = template.content.cloneNode(true) as DocumentFragment;
  const itemEl = fragment.querySelector(".history-item") as HTMLElement;
  if (!itemEl) return null;

  itemEl.dataset.novelId = item.novelId;
  updateItemElement(itemEl, item);

  const removeBtn = itemEl.querySelector(".history-remove") as HTMLButtonElement;
  removeBtn?.addEventListener("click", (e) => {
    e.preventDefault();
    e.stopPropagation();
    onRemove(item.novelId, itemEl);
  });

  return itemEl;
}

function updateItemElement(itemEl: HTMLElement, item: HistoryItem): void {
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

  const novelUrl = `/${item.novelId}`;
  if (posterLink && posterLink.getAttribute("href") !== novelUrl) {
    posterLink.href = novelUrl;
  }
  if (titleLink && titleLink.getAttribute("href") !== novelUrl) {
    titleLink.href = novelUrl;
  }

  if (item.coverUrl) {
    if (img && img.getAttribute("src") !== item.coverUrl) {
      img.src = item.coverUrl;
      img.alt = item.title;
    }
    const wrapper = itemEl.querySelector(".poster-wrapper") as HTMLElement;
    if (wrapper) {
      wrapper.style.setProperty("--bg-url", `url(${item.coverUrl})`);
    }
  }

  if (titleEl && titleEl.textContent !== item.title) {
    titleEl.textContent = item.title;
  }
  if (authorEl && authorEl.textContent !== (item.author || "")) {
    authorEl.textContent = item.author || "";
  }

  if (item.totalChapters > 0) {
    let percent = Math.round((item.chapterNum / item.totalChapters) * 100);
    if (percent < 1) percent = 1;
    if (percent > 100) percent = 100;
    const progressHTML = `Глава <strong>${item.chapterNum}</strong> из ${item.totalChapters}`;
    if (progressTextEl && progressTextEl.innerHTML !== progressHTML) {
      progressTextEl.innerHTML = progressHTML;
    }
    const percentStr = `${percent}%`;
    if (percentEl && percentEl.textContent !== percentStr) {
      percentEl.textContent = percentStr;
    }
    if (progressBar && progressBar.style.width !== percentStr) {
      progressBar.style.width = percentStr;
    }
  } else {
    const progressHTML = `Глава <strong>${item.chapterNum}</strong>`;
    if (progressTextEl && progressTextEl.innerHTML !== progressHTML) {
      progressTextEl.innerHTML = progressHTML;
    }
    if (percentEl && percentEl.textContent !== "") {
      percentEl.textContent = "";
    }
    if (progressBar && progressBar.style.width !== "0%") {
      progressBar.style.width = "0%";
    }
  }
}

export function initHistoryPage(): void {
  const historyList = document.getElementById("history-list");
  const historyEmpty = document.getElementById("history-empty");
  const historySkeleton = document.getElementById("history-skeleton");
  const clearAllBtn = document.getElementById("history-clear-all");

  if (!historyList || !historyEmpty) return;

  const template = document.getElementById(
    "tpl-history-item",
  ) as HTMLTemplateElement;

  const handleRemove = (novelId: string, itemEl: HTMLElement): void => {
    removeHistoryItem(novelId);
    itemEl.remove();
    checkEmpty();
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
            chapterNum: nextProgress.chapterNum,
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

  if (!historySetupDone) {
    historySetupDone = true;
    clearAllBtn?.addEventListener("click", () => {
      if (confirm("Очистить всю историю чтения?")) {
        const progress = getProgressCookie();
        progress.novels = {};
        progress.lastRead = null;
        saveProgressCookie(progress);
        loadedItems = [];
        historyList.innerHTML = "";
        if (historySkeleton) {
          historySkeleton.innerHTML = "";
          historySkeleton.style.display = "none";
        }
        historyEmpty.style.display = "block";
        if (clearAllBtn) clearAllBtn.style.display = "none";
      }
    });
  }

  const progress = getProgressCookie();
  const novelIds = Object.keys(progress.novels || {});

  if (novelIds.length === 0) {
    historyList.innerHTML = "";
    historyEmpty.style.display = "block";
    if (historySkeleton) {
      historySkeleton.innerHTML = "";
      historySkeleton.style.display = "none";
    }
    if (clearAllBtn) clearAllBtn.style.display = "none";
    loadedItems = [];
    return;
  }

  historyEmpty.style.display = "none";
  if (clearAllBtn) clearAllBtn.style.display = "block";

  const items: HistoryItem[] = [];
  let allCached = true;

  for (const [novelId, novelProgress] of Object.entries(
    progress.novels as Record<string, NovelProgress>,
  )) {
    const cached = getCachedNovel(novelId);
    if (cached) {
      items.push({
        novelId,
        title: cached.title || novelId,
        author: cached.author || "",
        coverUrl: cached.cover_url || "",
        chapterNum: novelProgress.chapterNum,
        totalChapters: cached.chapter_count || 0,
        readAt: novelProgress.readAt,
      });
    } else if (progress.lastRead?.novelId === novelId) {
      items.push({
        novelId,
        title: progress.lastRead.title || novelId,
        author: progress.lastRead.author || "",
        coverUrl: progress.lastRead.coverUrl || "",
        chapterNum: novelProgress.chapterNum,
        totalChapters: progress.lastRead.totalChapters || 0,
        readAt: novelProgress.readAt,
      });
    } else {
      allCached = false;
      items.push({
        novelId,
        title: novelId,
        author: "",
        coverUrl: "",
        chapterNum: novelProgress.chapterNum,
        totalChapters: 0,
        readAt: novelProgress.readAt,
      });
    }
  }

  items.sort((a, b) => b.readAt - a.readAt);
  loadedItems = items;

  const existingElements = Array.from(
    historyList.querySelectorAll<HTMLElement>(".history-item"),
  );
  const hasExistingDOM = existingElements.length > 0;

  if (!hasExistingDOM) {
    if (allCached && template) {
      const fragment = document.createDocumentFragment();
      for (const item of items) {
        const itemEl = createItemElement(item, template, handleRemove);
        if (itemEl) fragment.appendChild(itemEl);
      }
      historyList.appendChild(fragment);
      if (historySkeleton) {
        historySkeleton.innerHTML = "";
        historySkeleton.style.display = "none";
      }
    } else {
      if (historySkeleton) {
        historySkeleton.innerHTML = "";
        historySkeleton.style.display = "";
        const skeletonTemplate = document.getElementById(
          "tpl-history-skeleton",
        ) as HTMLTemplateElement;
        if (skeletonTemplate) {
          for (let i = 0; i < novelIds.length; i++) {
            historySkeleton.appendChild(skeletonTemplate.content.cloneNode(true));
          }
        }
      }
    }
  } else {
    if (historySkeleton) {
      historySkeleton.innerHTML = "";
      historySkeleton.style.display = "none";
    }
  }

  fetchNovelsBatch(novelIds).then((novelDataMap) => {
    if (historySkeleton) {
      historySkeleton.innerHTML = "";
      historySkeleton.style.display = "none";
    }

    const currentProgress = getProgressCookie();
    const currentNovelIds = Object.keys(currentProgress.novels || {});
    if (currentNovelIds.length === 0) {
      historyList.innerHTML = "";
      historyEmpty.style.display = "block";
      if (clearAllBtn) clearAllBtn.style.display = "none";
      loadedItems = [];
      return;
    }

    let progressUpdated = false;
    const updatedItems: HistoryItem[] = [];
    for (const [novelId, novelProgress] of Object.entries(
      currentProgress.novels as Record<string, NovelProgress>,
    )) {
      const novelData = novelDataMap.get(novelId) || getCachedNovel(novelId);
      updatedItems.push({
        novelId,
        title: novelData?.title || novelId,
        author: novelData?.author || "",
        coverUrl: novelData?.cover_url || "",
        chapterNum: novelProgress.chapterNum,
        totalChapters: novelData?.chapter_count || 0,
        readAt: novelProgress.readAt,
      });

      if (
        currentProgress.lastRead?.novelId === novelId &&
        novelData?.chapter_count &&
        novelData.chapter_count > currentProgress.lastRead.totalChapters
      ) {
        currentProgress.lastRead.totalChapters = novelData.chapter_count;
        progressUpdated = true;
      }
    }

    if (progressUpdated) {
      saveProgressCookieQuietly(currentProgress);
    }

    updatedItems.sort((a, b) => b.readAt - a.readAt);
    loadedItems = updatedItems;

    if (!template) return;

    const currentElements = Array.from(
      historyList.querySelectorAll<HTMLElement>(".history-item"),
    );
    const currentMap = new Map<string, HTMLElement>();
    currentElements.forEach((el) => {
      if (el.dataset.novelId) currentMap.set(el.dataset.novelId, el);
    });

    const updatedIdsSet = new Set(updatedItems.map((it) => it.novelId));
    currentElements.forEach((el) => {
      if (el.dataset.novelId && !updatedIdsSet.has(el.dataset.novelId)) {
        el.remove();
      }
    });

    let previousEl: HTMLElement | null = null;
    for (const item of updatedItems) {
      let el = currentMap.get(item.novelId);
      if (el) {
        updateItemElement(el, item);
      } else {
        el = createItemElement(item, template, handleRemove);
      }

      if (el) {
        if (!previousEl) {
          if (historyList.firstElementChild !== el) {
            historyList.prepend(el);
          }
        } else {
          if (previousEl.nextElementSibling !== el) {
            previousEl.after(el);
          }
        }
        previousEl = el;
      }
    }
  });
}

export function refreshHistory(): void {
  initHistoryPage();
}
