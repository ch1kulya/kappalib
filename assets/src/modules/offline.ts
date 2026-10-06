import { trackEvent } from "./analytics";
import { renderRichText } from "./bookmarks";
import { formatRelativeTime, pluralize } from "./comments";
import { FALLBACK_COVER } from "./list";
import {
  getOfflineChapter,
  getOfflineJob,
  getOfflineNovel,
  getOfflineState,
  getSavedChapters,
  listOfflineJobs,
  listOfflineNovels,
  OFFLINE_CHANNEL,
  OFFLINE_CHAPTER_PATH_RE,
  OFFLINE_FLAG_KEY,
  OFFLINE_NOVEL_PATH_RE,
  OfflineBroadcast,
  OfflineChapter,
  OfflineCommand,
  OfflineJob,
  OfflineJobError,
  OfflineNovel,
  OfflineState,
  offlineStateOf,
  OfflineTocEntry,
} from "./offline-db";
import { profileManager } from "./profile";
import { getProgressCookie, NovelProgress } from "./progress";
import { mapStatus } from "./search";

interface DropdownView {
  icon: string;
  label: string;
  status: string;
  download: string | null;
  cancel: boolean;
  retry: boolean;
  remove: boolean;
}

interface DownloadEntry {
  novelId: string;
  novel?: OfflineNovel;
  job?: OfflineJob;
}

type OfflineListener = (message: OfflineBroadcast) => void;

const API_URL = process.env.API_URL;
const SERVICE_WORKER_URL = "/sw.js";
const UNTITLED_CHAPTER = "Без названия";
const WATCHDOG_INTERVAL_MS = 10000;
const WATCHDOG_STALL_MS = 20000;
const BATCH_IDS_LIMIT = 50;
const UPDATE_CHECK_CONCURRENCY = 4;
const WORKER_ACTIVATION_TIMEOUT_MS = 30000;

const ERROR_MESSAGES: Record<OfflineJobError, string> = {
  auth: "Войдите в аккаунт, чтобы продолжить загрузку",
  network: "Нет соединения с сервером",
  quota: "Недостаточно места на устройстве",
  not_found: "Новелла недоступна на сервере",
  unknown: "Не удалось загрузить новеллу",
};

const sizeFormatter = new Intl.NumberFormat("ru-RU", {
  maximumFractionDigits: 1,
});

const listeners = new Set<OfflineListener>();
let channel: BroadcastChannel | null = null;
let lastWorkerActivity = Date.now();
let watchdogTimer: number | null = null;

function isOfflineSupported(): boolean {
  return window.isSecureContext
    && "serviceWorker" in navigator
    && typeof indexedDB !== "undefined"
    && typeof BroadcastChannel !== "undefined";
}

function hasOfflineData(): boolean {
  try {
    return localStorage.getItem(OFFLINE_FLAG_KEY) === "true";
  } catch {
    return false;
  }
}

function markOfflineData(): void {
  try {
    localStorage.setItem(OFFLINE_FLAG_KEY, "true");
  } catch (err) {
    console.warn("Failed to persist offline flag", err);
  }
}

async function requestPersistentStorage(): Promise<void> {
  try {
    if (navigator.storage?.persist && !(await navigator.storage.persisted())) {
      await navigator.storage.persist();
    }
  } catch (err) {
    console.warn("Persistent storage request failed", err);
  }
}

function waitForActiveWorker(
  registration: ServiceWorkerRegistration,
): Promise<ServiceWorker> {
  if (registration.active) return Promise.resolve(registration.active);

  const worker = registration.installing ?? registration.waiting;
  if (!worker) {
    return Promise.reject(new Error("Service worker is unavailable"));
  }

  return new Promise((resolve, reject) => {
    const cleanup = () => {
      window.clearTimeout(timer);
      worker.removeEventListener("statechange", onStateChange);
    };
    const onStateChange = () => {
      if (worker.state === "activated") {
        cleanup();
        resolve(worker);
      } else if (worker.state === "redundant") {
        cleanup();
        reject(new Error("Service worker installation failed"));
      }
    };
    const timer = window.setTimeout(() => {
      cleanup();
      reject(new Error("Service worker activation timed out"));
    }, WORKER_ACTIVATION_TIMEOUT_MS);
    worker.addEventListener("statechange", onStateChange);
  });
}

async function postCommand(command: OfflineCommand): Promise<void> {
  const registration = await navigator.serviceWorker.register(
    SERVICE_WORKER_URL,
    { scope: "/" },
  );
  const worker = await waitForActiveWorker(registration);
  worker.postMessage(command);
}

async function sendCommand(command: OfflineCommand): Promise<void> {
  try {
    await postCommand(command);
  } catch (err) {
    console.error("Offline command failed", err);
    alert("Не удалось выполнить действие. Обновите страницу и попробуйте снова");
  }
}

function ensureChannel(): void {
  if (channel) return;
  channel = new BroadcastChannel(OFFLINE_CHANNEL);
  channel.onmessage = (event: MessageEvent<OfflineBroadcast>) => {
    lastWorkerActivity = Date.now();
    if (event.data.state.kind === "queued" || event.data.state.kind === "downloading") {
      startWatchdog();
    }
    notifyListeners(event.data);
  };
}

function notifyListeners(message: OfflineBroadcast): void {
  listeners.forEach((listener) => listener(message));
}

function subscribe(listener: OfflineListener): void {
  ensureChannel();
  listeners.add(listener);
}

function isResumable(job: OfflineJob): boolean {
  return job.status !== "error" || job.error === "network";
}

async function resumePendingJobs(): Promise<void> {
  try {
    const jobs = await listOfflineJobs();
    if (!jobs.some(isResumable)) {
      stopWatchdog();
      return;
    }
    startWatchdog();
    await postCommand({ type: "resume" });
  } catch (err) {
    console.warn("Failed to resume offline downloads", err);
  }
}

function startWatchdog(): void {
  if (watchdogTimer !== null) return;
  ensureChannel();
  lastWorkerActivity = Date.now();
  watchdogTimer = window.setInterval(() => {
    if (!navigator.onLine) return;
    if (Date.now() - lastWorkerActivity < WATCHDOG_STALL_MS) return;
    lastWorkerActivity = Date.now();
    void resumePendingJobs();
  }, WATCHDOG_INTERVAL_MS);
}

function stopWatchdog(): void {
  if (watchdogTimer === null) return;
  window.clearInterval(watchdogTimer);
  watchdogTimer = null;
}

async function requestDownload(novelId: string): Promise<void> {
  if (!profileManager.isLoggedIn()) {
    alert("Войдите в аккаунт, чтобы скачать новеллу");
    return;
  }

  markOfflineData();
  void requestPersistentStorage();
  notifyListeners({ novelId, state: { kind: "queued", done: 0, total: 0 } });

  try {
    await postCommand({ type: "download", novelId });
    startWatchdog();
    trackEvent("offline_download", { novel_id: novelId });
  } catch (err) {
    console.error("Failed to start offline download", err);
    const state = await getOfflineState(novelId).catch((): OfflineState => ({ kind: "none" }));
    notifyListeners({ novelId, state });
    alert("Не удалось начать загрузку. Обновите страницу и попробуйте снова");
  }
}

async function runAction(action: string | undefined, novelId: string): Promise<void> {
  switch (action) {
    case "download":
      await requestDownload(novelId);
      return;
    case "cancel":
      await sendCommand({ type: "cancel", novelId });
      return;
    case "delete":
      if (!confirm("Удалить сохранённую новеллу с устройства?")) return;
      trackEvent("offline_delete", { novel_id: novelId });
      await sendCommand({ type: "delete", novelId });
      return;
  }
}

function formatBytes(bytes: number): string {
  const units = ["Б", "КБ", "МБ", "ГБ"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return `${sizeFormatter.format(value)} ${units[unit]}`;
}

function chaptersLabel(count: number): string {
  return `${count} ${pluralize(count, "глава", "главы", "глав")}`;
}

function novelsLabel(count: number): string {
  return `${count} ${pluralize(count, "новелла", "новеллы", "новелл")}`;
}

function percentOf(done: number, total: number): number {
  if (total <= 0) return 0;
  return Math.min(100, Math.floor((done / total) * 100));
}

function readingPercent(chapterNum: number, total: number): number {
  if (chapterNum <= 0 || total <= 0) return 0;
  return Math.min(100, Math.max(1, Math.floor((chapterNum / total) * 100)));
}

function isHttpUrl(value: string | null): value is string {
  if (!value) return false;
  try {
    const url = new URL(value);
    return url.protocol === "https:" || url.protocol === "http:";
  } catch {
    return false;
  }
}

function resolveCover(coverUrl: string | null): string {
  return isHttpUrl(coverUrl) ? coverUrl : FALLBACK_COVER;
}

function chapterUrl(novelId: string, chapterId: string): string {
  return `/${novelId}/chapter/${chapterId}`;
}

function chapterDocumentTitle(chapter: OfflineChapter): string {
  const prefix = `Глава ${chapter.num}`;
  return chapter.title === UNTITLED_CHAPTER ? prefix : `${prefix}: ${chapter.title}`;
}

function requireElement<T extends Element = HTMLElement>(root: ParentNode, selector: string): T {
  const element = root.querySelector<T>(selector);
  if (!element) throw new Error(`Element ${selector} not found`);
  return element;
}

function field<T extends Element = HTMLElement>(root: ParentNode, name: string): T {
  return requireElement<T>(root, `[data-field="${name}"]`);
}

function cloneTemplate(id: string): DocumentFragment {
  const template = document.getElementById(id);
  if (!(template instanceof HTMLTemplateElement)) {
    throw new Error(`Template #${id} not found`);
  }
  return template.content.cloneNode(true) as DocumentFragment;
}

function setCountText(el: Element, prefix: string, value: number, suffix: string): void {
  const strong = document.createElement("strong");
  strong.textContent = String(value);
  el.replaceChildren(prefix, strong, suffix);
}

function setCover(img: HTMLImageElement, coverUrl: string | null, alt: string): void {
  const src = resolveCover(coverUrl);
  const wrapper = img.closest<HTMLElement>(".poster-wrapper");
  img.alt = alt;
  img.onerror = () => {
    img.onerror = null;
    img.src = FALLBACK_COVER;
    wrapper?.style.setProperty("--bg-url", `url(${JSON.stringify(FALLBACK_COVER)})`);
  };
  img.src = src;
  wrapper?.style.setProperty("--bg-url", `url(${JSON.stringify(src)})`);
}

function closeDropdown(root: HTMLElement): void {
  root.classList.remove("active");
  root.querySelector(".dropdown-btn")?.setAttribute("aria-expanded", "false");
}

export async function renderOfflinePage(): Promise<void> {
  try {
    const reader = document.getElementById("offline-reader");
    if (reader) {
      await renderOfflineReader(reader);
      return;
    }
    if (document.getElementById("downloads-page")) {
      await renderDownloadsShell();
    }
  } catch (err) {
    console.error("Failed to render offline page", err);
  }
}

async function renderOfflineReader(root: HTMLElement): Promise<void> {
  const match = OFFLINE_CHAPTER_PATH_RE.exec(window.location.pathname);
  const [novel, chapter] = match && isOfflineSupported()
    ? await Promise.all([getOfflineNovel(match[1]), getOfflineChapter(match[2])])
    : [undefined, undefined];

  if (!novel || !chapter || chapter.novelId !== novel.id) {
    window.location.replace("/downloads");
    return;
  }

  const saved = await getSavedChapters(novel.id);
  const available = novel.toc.filter((entry) => saved.has(entry.id));
  const index = available.findIndex((entry) => entry.id === chapter.id);
  const prev = index > 0 ? available[index - 1] : undefined;
  const next = index >= 0 ? available[index + 1] : undefined;

  const fragment = cloneTemplate("tpl-offline-chapter");

  const tracker = requireElement(fragment, "#reading-tracker");
  tracker.dataset.novelId = novel.id;
  tracker.dataset.chapterId = chapter.id;
  tracker.dataset.chapterNum = String(chapter.num);
  tracker.dataset.novelTitle = novel.title;
  tracker.dataset.novelAuthor = novel.author;
  tracker.dataset.novelCover = resolveCover(novel.coverUrl);
  tracker.dataset.chapterTitle = chapter.title;
  tracker.dataset.totalChapters = String(novel.toc.length);

  fillChapterHeading(field(fragment, "heading"), chapter.num, chapter.title);
  field(fragment, "content").innerHTML = chapter.content;
  fillChapterSource(fragment, chapter);
  fillChapterNavigation(fragment, novel.id, prev, next);

  root.appendChild(fragment);

  const backLink = document.querySelector<HTMLAnchorElement>("#main-header .header-left");
  if (backLink) backLink.href = `/${novel.id}`;
  document.title = chapterDocumentTitle(chapter);
  if (novel.ageRating === "18+") window.isAdultContent = true;
}

function fillChapterHeading(el: HTMLElement, num: number, title: string): void {
  el.textContent = `Глава ${num}`;
  if (title === UNTITLED_CHAPTER) return;
  const titleEl = document.createElement("span");
  renderRichText(titleEl, title);
  el.append(": ", ...Array.from(titleEl.childNodes));
}

function fillChapterSource(root: ParentNode, chapter: OfflineChapter): void {
  const card = field(root, "source");
  const source = chapter.source;
  if (!source) {
    card.remove();
    return;
  }

  field(card, "sourceLabel").textContent = source.label;
  field(card, "sourceName").textContent = source.name;
  field(card, "sourceDate").textContent = formatRelativeTime(chapter.createdAt);

  const logo = field<HTMLImageElement>(card, "sourceLogo");
  const placeholder = field(card, "sourcePlaceholder");
  placeholder.textContent = Array.from(source.name)[0] ?? "";

  const showPlaceholder = () => {
    logo.remove();
    placeholder.style.display = "";
  };

  if (isHttpUrl(source.logoUrl)) {
    logo.alt = source.name;
    logo.addEventListener("error", showPlaceholder, { once: true });
    logo.src = source.logoUrl;
  } else {
    showPlaceholder();
  }
}

function fillChapterNavigation(
  root: ParentNode,
  novelId: string,
  prev: OfflineTocEntry | undefined,
  next: OfflineTocEntry | undefined,
): void {
  const prevLink = field<HTMLAnchorElement>(root, "prev");
  const prevDisabled = field(root, "prevDisabled");
  if (prev) {
    prevLink.href = chapterUrl(novelId, prev.id);
    prevDisabled.remove();
  } else {
    prevLink.remove();
  }

  const nextLink = field<HTMLAnchorElement>(root, "next");
  const nextDisabled = field(root, "nextDisabled");
  if (next) {
    nextLink.href = chapterUrl(novelId, next.id);
    nextLink.dataset.chapterId = next.id;
    nextLink.dataset.chapterNum = String(next.num);
    nextDisabled.remove();
  } else {
    nextLink.remove();
  }
}

async function renderDownloadsShell(): Promise<void> {
  const match = OFFLINE_NOVEL_PATH_RE.exec(window.location.pathname);
  if (!match || !isOfflineSupported()) return;
  const novel = await getOfflineNovel(match[1]);
  if (novel) await renderOfflineNovel(novel);
}

async function renderOfflineNovel(novel: OfflineNovel): Promise<void> {
  const container = document.getElementById("offline-novel");
  const library = document.getElementById("downloads-library");
  if (!container || !library) return;

  const saved = await getSavedChapters(novel.id);
  const fragment = cloneTemplate("tpl-offline-novel");

  setCover(field<HTMLImageElement>(fragment, "cover"), novel.coverUrl, novel.titleEn || novel.title);
  field(fragment, "title").textContent = novel.title;
  field(fragment, "author").textContent = novel.author;
  field(fragment, "status").textContent = mapStatus(novel.status);

  const ageBadge = field(fragment, "ageRating");
  if (novel.ageRating) {
    ageBadge.textContent = novel.ageRating;
    ageBadge.classList.toggle("badge-danger", novel.ageRating === "18+");
    ageBadge.style.display = "";
  }

  if (novel.description) {
    field(fragment, "description").textContent = novel.description;
  } else {
    field(fragment, "descriptionWrapper").remove();
  }

  fillReadActions(fragment, novel);

  const list = requireElement(fragment, "#chapters-list");
  novel.toc.forEach((entry) => {
    list.appendChild(createTocItem(novel.id, entry, saved.has(entry.id)));
  });

  container.replaceChildren(fragment);
  container.style.display = "";
  library.style.display = "none";
  document.title = `${novel.title} — kappalib`;
  if (novel.ageRating === "18+") window.isAdultContent = true;
}

function fillReadActions(root: ParentNode, novel: OfflineNovel): void {
  const progress = getProgressCookie().novels[novel.id];
  const current = progress
    ? novel.toc.find((entry) => entry.id === progress.chapterId)
    : undefined;
  const target = current ?? novel.toc[0];

  root.querySelectorAll<HTMLAnchorElement>("[data-field=\"read\"]").forEach((link) => {
    if (!target) {
      link.style.display = "none";
      return;
    }
    link.href = chapterUrl(novel.id, target.id);
    link.textContent = current ? "Продолжить" : "Начать читать";
    if (current) {
      link.classList.remove("btn-start");
      link.classList.add("btn-continue", "btn-primary");
    }
  });

  if (!current) return;

  const total = novel.toc.length;
  const percent = readingPercent(current.num, total);
  root.querySelectorAll<HTMLElement>("[data-field=\"progress\"]").forEach((container) => {
    setCountText(requireElement(container, ".progress-text"), "Глава ", current.num, ` из ${total}`);
    requireElement(container, ".progress-percent").textContent = `${percent}%`;
    requireElement(container, ".progress-bar-fill").style.width = `${percent}%`;
    container.style.display = "";
  });
}

function createTocItem(novelId: string, entry: OfflineTocEntry, available: boolean): HTMLElement {
  const fragment = cloneTemplate("tpl-offline-chapter-item");
  const item = requireElement<HTMLAnchorElement>(fragment, ".chapter-item");
  item.href = chapterUrl(novelId, entry.id);
  item.dataset.chapterId = entry.id;
  item.dataset.sortValue = String(entry.num);
  requireElement(item, ".chapter-num").textContent = `Глава ${entry.num}.`;
  if (entry.title !== UNTITLED_CHAPTER) {
    renderRichText(requireElement(item, ".chapter-title"), entry.title);
  }
  item.classList.toggle("is-offline-missing", !available);
  return item;
}

function dropdownView(state: OfflineState, newChapters: number): DropdownView {
  switch (state.kind) {
    case "none":
      return {
        icon: "none",
        label: "Скачать",
        status: "",
        download: "Скачать новеллу",
        cancel: false,
        retry: false,
        remove: false,
      };
    case "queued":
      return {
        icon: "progress",
        label: "В очереди",
        status: "В очереди на загрузку",
        download: null,
        cancel: true,
        retry: false,
        remove: false,
      };
    case "downloading":
      return {
        icon: "progress",
        label: `Загрузка ${percentOf(state.done, state.total)}%`,
        status: state.total > 0
          ? `Загружено ${state.done} из ${state.total}`
          : "Подготовка загрузки…",
        download: null,
        cancel: true,
        retry: false,
        remove: false,
      };
    case "error":
      return {
        icon: "error",
        label: "Ошибка",
        status: ERROR_MESSAGES[state.error],
        download: null,
        cancel: false,
        retry: true,
        remove: true,
      };
    case "ready": {
      const partial = state.saved < state.total;
      const saved = partial ? `${state.saved} из ${state.total}` : chaptersLabel(state.saved);
      const details = `${saved} · ${formatBytes(state.bytes)}`;
      if (partial) {
        return {
          icon: "update",
          label: "Докачать",
          status: details,
          download: "Докачать",
          cancel: false,
          retry: false,
          remove: true,
        };
      }
      if (newChapters > 0) {
        return {
          icon: "update",
          label: "Обновить",
          status: `${details}. Новых глав: ${newChapters}`,
          download: "Скачать новые главы",
          cancel: false,
          retry: false,
          remove: true,
        };
      }
      return {
        icon: "ready",
        label: "Скачано",
        status: details,
        download: null,
        cancel: false,
        retry: false,
        remove: true,
      };
    }
  }
}

function initOfflineDropdown(root: HTMLElement): void {
  const novelId = root.dataset.novelId;
  if (!novelId) return;

  const status = field(root, "status");
  const menu = requireElement(root, ".dropdown-menu");
  const button = requireElement(root, ".dropdown-btn");
  const label = requireElement(root, ".of-btn-label");
  const downloadItem = requireElement(root, "[data-item=\"download\"]");
  const downloadLabel = field(downloadItem, "downloadLabel");
  const cancelItem = requireElement(root, "[data-item=\"cancel\"]");
  const retryItem = requireElement(root, "[data-item=\"retry\"]");
  const deleteItem = requireElement(root, "[data-item=\"delete\"]");
  const serverIds = Array.from(
    document.querySelectorAll<HTMLElement>("#chapters-list .chapter-item[data-chapter-id]"),
  )
    .map((item) => item.dataset.chapterId)
    .filter((id): id is string => !!id);

  let state: OfflineState = { kind: "none" };
  let tocIds: Set<string> | null = null;

  const newChapterCount = (): number => {
    const ids = tocIds;
    return ids ? serverIds.filter((id) => !ids.has(id)).length : 0;
  };

  const render = () => {
    const view = dropdownView(state, newChapterCount());
    const progress = state.kind === "queued" || state.kind === "downloading"
      ? percentOf(state.done, state.total)
      : 0;
    root.dataset.state = view.icon;
    root.style.setProperty("--of-progress", String(progress));
    label.textContent = view.label;
    button.setAttribute("aria-label", view.label);
    button.title = view.label;
    status.textContent = view.status;
    status.style.display = view.status ? "" : "none";
    if (view.download) downloadLabel.textContent = view.download;
    downloadItem.style.display = view.download ? "" : "none";
    cancelItem.style.display = view.cancel ? "" : "none";
    retryItem.style.display = view.retry ? "" : "none";
    deleteItem.style.display = view.remove ? "" : "none";
  };

  const refresh = async () => {
    if (!hasOfflineData()) {
      state = { kind: "none" };
      tocIds = null;
      render();
      return;
    }
    const [novel, job] = await Promise.all([
      getOfflineNovel(novelId),
      getOfflineJob(novelId),
    ]);
    state = offlineStateOf(novel, job);
    tocIds = novel ? new Set(novel.toc.map((entry) => entry.id)) : null;
    render();
  };

  menu.addEventListener("click", (e) => {
    const button = (e.target as HTMLElement).closest<HTMLElement>("[data-action]");
    if (!button) return;
    e.preventDefault();
    closeDropdown(root);
    void runAction(button.dataset.action, novelId);
  });

  subscribe((message) => {
    if (message.novelId !== null && message.novelId !== novelId) return;
    if (message.state.kind === "queued" || message.state.kind === "downloading") {
      state = message.state;
      render();
      return;
    }
    refresh().catch((err) => console.error("Failed to refresh offline state", err));
  });

  render();
  refresh().catch((err) => console.error("Failed to load offline state", err));
}

function itemButton(action: string, label: string): HTMLButtonElement {
  const button = document.createElement("button");
  button.type = "button";
  button.className = "btn-text";
  button.dataset.action = action;
  button.textContent = label;
  return button;
}

function fillItemState(
  item: HTMLElement,
  novelId: string,
  state: OfflineState,
  newChapters: number,
  progress: Record<string, NovelProgress>,
): void {
  const text = field(item, "progress");
  let percent = 0;

  switch (state.kind) {
    case "none":
      text.textContent = "";
      break;
    case "queued":
      text.textContent = "В очереди на загрузку";
      percent = percentOf(state.done, state.total);
      break;
    case "downloading":
      setCountText(text, "Загрузка: ", state.done, ` из ${state.total}`);
      percent = percentOf(state.done, state.total);
      break;
    case "error":
      text.textContent = ERROR_MESSAGES[state.error];
      percent = percentOf(state.done, state.total);
      break;
    case "ready": {
      const reading = progress[novelId];
      if (state.saved < state.total) {
        setCountText(text, "Сохранено ", state.saved, ` из ${state.total}`);
        percent = percentOf(state.saved, state.total);
      } else if (reading) {
        setCountText(text, "Глава ", reading.chapterNum, ` из ${state.total}`);
        percent = readingPercent(reading.chapterNum, state.total);
      } else {
        text.textContent = "Не начато";
      }
      break;
    }
  }

  field(item, "percent").textContent = percent > 0 ? `${percent}%` : "";
  field(item, "progressBar").style.width = `${percent}%`;

  const buttons: HTMLButtonElement[] = [];
  if (state.kind === "queued" || state.kind === "downloading") {
    buttons.push(itemButton("cancel", "Отменить"));
  } else if (state.kind === "error") {
    buttons.push(itemButton("download", "Повторить"));
  } else if (state.kind === "ready" && state.saved < state.total) {
    buttons.push(itemButton("download", "Докачать"));
  } else if (state.kind === "ready" && newChapters > 0) {
    buttons.push(itemButton("download", "Скачать новые главы"));
  }

  const actions = field(item, "actions");
  actions.replaceChildren(...buttons);
  actions.style.display = buttons.length > 0 ? "" : "none";
}

function fillDownloadItem(
  item: HTMLElement,
  entry: DownloadEntry,
  newChapters: number,
  progress: Record<string, NovelProgress>,
): void {
  const { novelId, novel, job } = entry;
  const url = `/${novelId}`;

  item.dataset.novelId = novelId;
  item.querySelectorAll<HTMLAnchorElement>("a.history-poster-link, a.history-title").forEach((link) => {
    link.href = url;
  });

  field(item, "title").textContent = novel?.title ?? "Подготовка загрузки…";
  field(item, "meta").textContent = novel
    ? [novel.author, chaptersLabel(novel.savedCount), formatBytes(novel.bytes)]
      .filter(Boolean)
      .join(" · ")
    : "";

  const img = requireElement<HTMLImageElement>(item, "img");
  const cover = novel?.coverUrl ?? "";
  if (img.dataset.cover !== cover || !img.getAttribute("src")) {
    img.dataset.cover = cover;
    setCover(img, novel?.coverUrl ?? null, novel?.title ?? "");
  }

  fillItemState(item, novelId, offlineStateOf(novel, job), newChapters, progress);
}

async function fetchChapterCounts(ids: string[]): Promise<Map<string, number>> {
  const chunks: string[][] = [];
  for (let i = 0; i < ids.length; i += BATCH_IDS_LIMIT) {
    chunks.push(ids.slice(i, i + BATCH_IDS_LIMIT));
  }
  const pages = await Promise.all(chunks.map(async (chunk) => {
    const res = await fetch(`${API_URL}/novels/batch`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ ids: chunk }),
    });
    if (!res.ok) return [];
    const novels: { id: string; chapter_count: number }[] = await res.json();
    return novels;
  }));
  return new Map(pages.flat().map((novel): [string, number] => [novel.id, novel.chapter_count]));
}

async function mapWithConcurrency<T, R>(
  items: T[],
  limit: number,
  task: (item: T) => Promise<R>,
): Promise<R[]> {
  const results: R[] = new Array(items.length);
  let next = 0;
  const worker = async () => {
    while (next < items.length) {
      const index = next++;
      results[index] = await task(items[index]);
    }
  };
  await Promise.all(Array.from({ length: Math.min(limit, items.length) }, worker));
  return results;
}

async function fetchNewChapterCount(novel: OfflineNovel): Promise<number> {
  const res = await fetch(`${API_URL}/novels/${novel.id}/chapters`);
  if (!res.ok) return 0;
  const list: { chapters: { id: string }[] } = await res.json();
  const known = new Set(novel.toc.map((entry) => entry.id));
  return list.chapters.filter((chapter) => !known.has(chapter.id)).length;
}

function initDownloadsPage(): void {
  const library = document.getElementById("downloads-library");
  const list = document.getElementById("downloads-list");
  const empty = document.getElementById("downloads-empty");
  const unsupported = document.getElementById("downloads-unsupported");
  const summary = document.getElementById("downloads-summary");
  const clearAll = document.getElementById("downloads-clear-all");
  const help = document.getElementById("downloads-help");
  if (!library || !list || !empty || !unsupported || !summary || !clearAll || !help) return;
  if (library.style.display === "none") return;

  if (!isOfflineSupported()) {
    unsupported.style.display = "";
    help.style.display = "none";
    return;
  }

  const items = new Map<string, HTMLElement>();
  const newChapters = new Map<string, number>();
  let entries: DownloadEntry[] = [];

  const render = () => {
    const progress = getProgressCookie().novels;
    const seen = new Set<string>();

    entries.forEach((entry) => {
      let item = items.get(entry.novelId);
      if (!item) {
        item = requireElement(cloneTemplate("tpl-download-item"), ".download-item");
        items.set(entry.novelId, item);
      }
      fillDownloadItem(item, entry, newChapters.get(entry.novelId) ?? 0, progress);
      list.appendChild(item);
      seen.add(entry.novelId);
    });

    items.forEach((item, novelId) => {
      if (!seen.has(novelId)) {
        item.remove();
        items.delete(novelId);
      }
    });

    const novels = entries.filter((entry) => entry.novel);
    const totalBytes = novels.reduce((sum, entry) => sum + (entry.novel?.bytes ?? 0), 0);
    const hasEntries = entries.length > 0;
    empty.style.display = hasEntries ? "none" : "";
    clearAll.style.display = hasEntries ? "" : "none";
    help.style.display = hasEntries ? "none" : "";
    summary.style.display = hasEntries ? "" : "none";
    summary.textContent = `${novelsLabel(novels.length)} · ${formatBytes(totalBytes)}`;
  };

  const refresh = async () => {
    const [novels, jobs] = await Promise.all([listOfflineNovels(), listOfflineJobs()]);
    const jobMap = new Map(jobs.map((job): [string, OfflineJob] => [job.novelId, job]));
    const pending = jobs
      .filter((job) => !novels.some((novel) => novel.id === job.novelId))
      .map((job) => ({ novelId: job.novelId, job }));
    const stored = novels
      .sort((a, b) => b.downloadedAt - a.downloadedAt)
      .map((novel) => ({ novelId: novel.id, novel, job: jobMap.get(novel.id) }));
    entries = [...pending, ...stored];
    render();
  };

  const checkUpdates = async () => {
    const novels = entries.flatMap((entry) => (entry.novel && !entry.job ? [entry.novel] : []));
    if (!navigator.onLine || novels.length === 0) return;
    const counts = await fetchChapterCounts(novels.map((novel) => novel.id));
    const candidates = novels.filter((novel) => (counts.get(novel.id) ?? 0) > novel.toc.length);
    const found = await mapWithConcurrency(
      candidates,
      UPDATE_CHECK_CONCURRENCY,
      (novel) => fetchNewChapterCount(novel).catch(() => 0),
    );
    candidates.forEach((novel, index) => {
      if (found[index] > 0) {
        newChapters.set(novel.id, found[index]);
      } else {
        newChapters.delete(novel.id);
      }
    });
    render();
  };

  list.addEventListener("click", (e) => {
    const target = e.target as HTMLElement;
    const novelId = target.closest<HTMLElement>(".download-item")?.dataset.novelId;
    if (!novelId) return;

    if (target.closest(".history-remove")) {
      e.preventDefault();
      void runAction("delete", novelId);
      return;
    }

    const button = target.closest<HTMLElement>("[data-action]");
    if (button) {
      e.preventDefault();
      void runAction(button.dataset.action, novelId);
    }
  });

  clearAll.addEventListener("click", () => {
    if (!confirm("Удалить все сохранённые новеллы с устройства?")) return;
    trackEvent("offline_delete_all");
    void sendCommand({ type: "delete-all" });
  });

  subscribe((message) => {
    const { novelId, state } = message;
    if (novelId && (state.kind === "queued" || state.kind === "downloading")) {
      const item = items.get(novelId);
      if (item) {
        fillItemState(item, novelId, state, newChapters.get(novelId) ?? 0, getProgressCookie().novels);
        return;
      }
    }
    if (state.kind === "ready" && novelId) newChapters.delete(novelId);
    refresh().catch((err) => console.error("Failed to refresh downloads", err));
  });

  refresh()
    .then(checkUpdates)
    .catch((err) => console.error("Failed to load downloads", err));
}

export function initOffline(): void {
  initDownloadsPage();

  const dropdowns = document.querySelectorAll<HTMLElement>(".novel-offline-dropdown[data-novel-id]");
  if (!isOfflineSupported()) {
    dropdowns.forEach((root) => {
      root.style.display = "none";
    });
    return;
  }

  dropdowns.forEach((root) => initOfflineDropdown(root));

  if (!hasOfflineData()) return;

  navigator.serviceWorker
    .register(SERVICE_WORKER_URL, { scope: "/" })
    .catch((err) => console.warn("Service worker registration failed", err));
  ensureChannel();
  window.addEventListener("online", () => void resumePendingJobs());
  if (navigator.onLine) void resumePendingJobs();
}
