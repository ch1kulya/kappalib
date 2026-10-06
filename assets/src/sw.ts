import {
  byteSize,
  clearOfflineData,
  deleteOfflineJob,
  deleteOfflineNovel,
  getOfflineChapter,
  getOfflineJob,
  getOfflineNovel,
  getSavedChapters,
  listOfflineJobs,
  NOVEL_ID_RE,
  OFFLINE_CHANNEL,
  OFFLINE_CHAPTER_PATH_RE,
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
  putOfflineJob,
  putOfflineNovel,
  removeOfflineChapters,
  saveOfflineChapters,
} from "./modules/offline-db";

declare const self: ServiceWorkerGlobalScope;

interface ApiNovel {
  id: string;
  title: string;
  title_en: string;
  author: string;
  description: string;
  status: string;
  age_rating: string | null;
  cover_url: string | null;
}

interface ApiChapterSummary {
  id: string;
  chapter_num: number;
  title: string;
}

interface ApiChaptersList {
  chapters: ApiChapterSummary[];
}

interface ApiSource {
  name: string;
  logo_url: string | null;
  label: string;
}

interface ApiChapter {
  id: string;
  novel_id: string;
  chapter_num: number;
  title: string;
  content: string;
  source: ApiSource | null;
  created_at: string;
}

interface ApiOfflinePage {
  chapters: ApiChapter[];
  next_after: number | null;
}

interface ActiveJob {
  novelId: string;
  controller: AbortController;
  finished: Promise<void>;
}

const VERSION = process.env.ASSET_VERSION;
const API_URL = process.env.API_URL;
const STATIC_CACHE_PREFIX = "kpl-static-";
const STATIC_CACHE = `${STATIC_CACHE_PREFIX}${VERSION}`;
const FONTS_CACHE = "kpl-fonts";
const FONT_STYLES_CACHE = "kpl-font-styles";
const COVERS_CACHE = "kpl-covers";
const FONTS_CACHE_LIMIT = 600;
const FONT_ORIGIN = "https://cdn.jsdelivr.net";
const COVER_ORIGIN = originOf(process.env.S3_PUBLIC_URL);
const LIBRARY_SHELL = "/offline/library";
const READER_SHELL = "/offline/reader";
const SHELL_URLS = [LIBRARY_SHELL, READER_SHELL];
const PRECACHE_ASSET_URLS = [
  `/assets/dist/app.js?v=${VERSION}`,
  `/assets/dist/styles/main.css?v=${VERSION}`,
  "/assets/fonts/InterVariable.woff2?v=4.1",
  "/assets/fonts/InterVariable-Italic.woff2?v=4.1",
];
const BRAND_ICON_URLS = process.env.COLOR_SCHEMES.map(
  (scheme) => `/assets/icons/${scheme}/logo.png`,
);
const OPTIONAL_PRECACHE_INTERVAL_MS = 100;
const BATCH_LIMIT = 50;
const BATCH_INTERVAL_MS = 1000;
const MAX_ATTEMPTS = 6;
const MAX_BACKOFF_MS = 30000;

class DownloadError extends Error {
  readonly reason: OfflineJobError;

  constructor(reason: OfflineJobError) {
    super(reason);
    this.reason = reason;
  }
}

const channel = new BroadcastChannel(OFFLINE_CHANNEL);
const queue: string[] = [];
let active: ActiveJob | null = null;
let draining: Promise<void> | null = null;
let lastBatchAt = 0;

function originOf(rawUrl: string): string | null {
  try {
    const url = new URL(rawUrl);
    return url.protocol === "https:" || url.protocol === "http:" ? url.origin : null;
  } catch {
    return null;
  }
}

function normalizeCover(rawUrl: string | null): string | null {
  const trimmed = rawUrl?.trim();
  if (!trimmed) return null;
  return originOf(trimmed) ? trimmed : null;
}

self.addEventListener("install", (event) => {
  event.waitUntil(
    (async () => {
      const cache = await caches.open(STATIC_CACHE);
      await cache.addAll(
        [
          ...SHELL_URLS.map((url) => new Request(url, { cache: "no-cache" })),
          ...PRECACHE_ASSET_URLS.map((url) => new Request(url)),
        ],
      );
      await precacheOptional(cache, BRAND_ICON_URLS);
      await self.skipWaiting();
    })(),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    (async () => {
      const keys = await caches.keys();
      await Promise.all(
        keys
          .filter((key) => key.startsWith(STATIC_CACHE_PREFIX) && key !== STATIC_CACHE)
          .map((key) => caches.delete(key)),
      );
      await removeFontStylesFromFilesCache();
      if (self.registration.navigationPreload) {
        await self.registration.navigationPreload.enable();
      }
      await self.clients.claim();
    })(),
  );
});

self.addEventListener("fetch", (event) => {
  const { request } = event;
  if (request.method !== "GET") return;

  const url = new URL(request.url);

  if (request.mode === "navigate") {
    event.respondWith(handleNavigation(event, url));
    return;
  }

  if (url.origin === self.location.origin) {
    if (url.pathname.startsWith("/assets/")) {
      event.respondWith(cacheFirst(event, STATIC_CACHE));
    }
    return;
  }

  if (url.origin === FONT_ORIGIN && request.destination === "style") {
    event.respondWith(cacheFirst(event, FONT_STYLES_CACHE));
    return;
  }

  if (url.origin === FONT_ORIGIN && request.destination === "font") {
    event.respondWith(cacheFirst(event, FONTS_CACHE, FONTS_CACHE_LIMIT));
    return;
  }

  if (COVER_ORIGIN !== null && url.origin === COVER_ORIGIN && request.destination === "image") {
    event.respondWith(coverFirst(request));
  }
});

self.addEventListener("message", (event) => {
  const command = parseCommand(event.data);
  if (!command) return;
  event.waitUntil(
    handleCommand(command).catch((err) => {
      console.error("Offline command failed", command.type, err);
    }),
  );
});

async function precacheOptional(cache: Cache, urls: string[]): Promise<void> {
  for (const url of urls) {
    if (await cache.match(url)) continue;
    try {
      const response = await fetch(url);
      if (response.status === 200) await cache.put(url, response);
    } catch (err) {
      console.warn("Failed to precache", url, err);
    }
    await new Promise((resolve) => setTimeout(resolve, OPTIONAL_PRECACHE_INTERVAL_MS));
  }
}

async function handleNavigation(event: FetchEvent, url: URL): Promise<Response> {
  try {
    const preloaded: Response | undefined = await event.preloadResponse;
    const response = preloaded ?? (await fetch(event.request));
    if (response.status < 500) return response;
    return (await offlineFallback(url, false)) ?? response;
  } catch {
    return (await offlineFallback(url, true)) ?? Response.error();
  }
}

async function offlineFallback(url: URL, networkFailed: boolean): Promise<Response | undefined> {
  try {
    const chapterMatch = OFFLINE_CHAPTER_PATH_RE.exec(url.pathname);
    if (chapterMatch) {
      const chapter = await getOfflineChapter(chapterMatch[2]);
      if (chapter && chapter.novelId === chapterMatch[1]) {
        return await caches.match(READER_SHELL);
      }
    }

    const novelMatch = OFFLINE_NOVEL_PATH_RE.exec(url.pathname);
    if (novelMatch && (await getOfflineNovel(novelMatch[1]))) {
      return await caches.match(LIBRARY_SHELL);
    }

    if (networkFailed) {
      return await caches.match(LIBRARY_SHELL);
    }
  } catch (err) {
    console.error("Offline fallback failed", err);
  }
  return undefined;
}

async function cacheFirst(event: FetchEvent, cacheName: string, limit?: number): Promise<Response> {
  const cache = await caches.open(cacheName);
  const cached = await cache.match(event.request);
  if (cached) return cached;

  const response = await fetch(event.request);
  if (response.status === 200) {
    const copy = response.clone();
    event.waitUntil(
      cache
        .put(event.request, copy)
        .then(() => (limit ? trimCache(cache, limit) : undefined))
        .catch((err) => console.warn("Failed to cache response", err)),
    );
  }
  return response;
}

async function removeFontStylesFromFilesCache(): Promise<void> {
  if (!(await caches.has(FONTS_CACHE))) return;
  const cache = await caches.open(FONTS_CACHE);
  const requests = await cache.keys();
  await Promise.all(
    requests
      .filter((request) => new URL(request.url).pathname.endsWith(".css"))
      .map((request) => cache.delete(request)),
  );
}

async function trimCache(cache: Cache, limit: number): Promise<void> {
  const keys = await cache.keys();
  const excess = keys.length - limit;
  if (excess <= 0) return;
  await Promise.all(keys.slice(0, excess).map((key) => cache.delete(key)));
}

async function coverFirst(request: Request): Promise<Response> {
  const cached = await caches.match(request.url, { cacheName: COVERS_CACHE });
  return cached ?? fetch(request);
}

async function cacheCover(coverUrl: string | null, signal: AbortSignal): Promise<void> {
  if (!coverUrl) return;
  try {
    const cache = await caches.open(COVERS_CACHE);
    if (await cache.match(coverUrl)) return;
    const response = await fetchCover(coverUrl, signal);
    if (signal.aborted) return;
    if (response.ok || response.type === "opaque") {
      await cache.put(coverUrl, response);
    }
  } catch (err) {
    if (!signal.aborted) console.warn("Failed to cache cover", err);
  }
}

async function fetchCover(coverUrl: string, signal: AbortSignal): Promise<Response> {
  try {
    return await fetch(coverUrl, { mode: "cors", credentials: "omit", signal });
  } catch (err) {
    if (signal.aborted) throw err;
    return fetch(coverUrl, { mode: "no-cors", credentials: "omit", signal });
  }
}

async function removeCover(coverUrl: string | null): Promise<void> {
  if (!coverUrl) return;
  const cache = await caches.open(COVERS_CACHE);
  await cache.delete(coverUrl);
}

function parseCommand(data: unknown): OfflineCommand | null {
  if (typeof data !== "object" || data === null) return null;
  const { type, novelId } = data as { type?: unknown; novelId?: unknown };

  if (type === "delete-all" || type === "resume") {
    return { type };
  }
  if (
    (type === "download" || type === "redownload" || type === "cancel" || type === "delete")
    && typeof novelId === "string"
    && NOVEL_ID_RE.test(novelId)
  ) {
    return { type, novelId };
  }
  return null;
}

async function handleCommand(command: OfflineCommand): Promise<void> {
  switch (command.type) {
    case "download":
      await startDownload(command.novelId);
      return;
    case "cancel": {
      await stopJob(command.novelId);
      const novel = await getOfflineNovel(command.novelId);
      if (novel && novel.savedCount === 0) {
        await deleteOfflineNovel(command.novelId);
        await removeCover(novel.coverUrl);
      } else {
        await deleteOfflineJob(command.novelId);
      }
      broadcast(command.novelId, await currentState(command.novelId));
      return;
    }
    case "redownload": {
      await stopJob(command.novelId);
      const novel = await getOfflineNovel(command.novelId);
      await deleteOfflineNovel(command.novelId);
      await removeCover(novel?.coverUrl ?? null);
      await startDownload(command.novelId);
      return;
    }
    case "delete": {
      await stopJob(command.novelId);
      const novel = await getOfflineNovel(command.novelId);
      await deleteOfflineNovel(command.novelId);
      await removeCover(novel?.coverUrl ?? null);
      broadcast(command.novelId, { kind: "none" });
      return;
    }
    case "delete-all":
      queue.length = 0;
      if (active) {
        const current = active;
        current.controller.abort();
        await current.finished;
      }
      await clearOfflineData();
      await caches.delete(COVERS_CACHE);
      broadcast(null, { kind: "none" });
      return;
    case "resume":
      await resumeJobs();
      return;
  }
}

function broadcast(novelId: string | null, state: OfflineState): void {
  const message: OfflineBroadcast = { novelId, state };
  channel.postMessage(message);
}

async function currentState(novelId: string): Promise<OfflineState> {
  const [novel, job] = await Promise.all([
    getOfflineNovel(novelId),
    getOfflineJob(novelId),
  ]);
  return offlineStateOf(novel, job);
}

async function startDownload(novelId: string): Promise<void> {
  if (active?.novelId !== novelId && !queue.includes(novelId)) {
    const [novel, job] = await Promise.all([
      getOfflineNovel(novelId),
      getOfflineJob(novelId),
    ]);
    const queued: OfflineJob = {
      novelId,
      status: "queued",
      error: null,
      done: novel?.savedCount ?? job?.done ?? 0,
      total: novel?.toc.length ?? job?.total ?? 0,
      updatedAt: Date.now(),
    };
    await putOfflineJob(queued);
    broadcast(novelId, offlineStateOf(novel, queued));
    enqueue(novelId);
  }
  await drainQueue();
}

async function resumeJobs(): Promise<void> {
  const jobs = await listOfflineJobs();
  jobs
    .filter((job) => job.status !== "error" || job.error === "network")
    .sort((a, b) => a.updatedAt - b.updatedAt)
    .forEach((job) => enqueue(job.novelId));
  await drainQueue();
}

function enqueue(novelId: string): void {
  if (active?.novelId === novelId || queue.includes(novelId)) return;
  queue.push(novelId);
}

function drainQueue(): Promise<void> {
  if (!draining) {
    draining = (async () => {
      let novelId = queue.shift();
      while (novelId !== undefined) {
        const controller = new AbortController();
        const finished = runJob(novelId, controller.signal);
        active = { novelId, controller, finished };
        await finished;
        active = null;
        novelId = queue.shift();
      }
    })().finally(() => {
      draining = null;
    });
  }
  return draining;
}

async function stopJob(novelId: string): Promise<void> {
  const index = queue.indexOf(novelId);
  if (index !== -1) queue.splice(index, 1);

  const current = active;
  if (current?.novelId === novelId) {
    current.controller.abort();
    await current.finished;
  }
}

async function runJob(novelId: string, signal: AbortSignal): Promise<void> {
  let job: OfflineJob | undefined;
  try {
    const stored = await getOfflineJob(novelId);
    if (!stored || signal.aborted) return;

    job = { ...stored, status: "running", error: null, updatedAt: Date.now() };
    await putOfflineJob(job);
    broadcast(novelId, offlineStateOf(undefined, job));

    await downloadNovel(job, signal);
    throwIfAborted(signal);

    await deleteOfflineJob(novelId);
    broadcast(novelId, await currentState(novelId));
  } catch (err) {
    if (signal.aborted || !job) return;
    const failed: OfflineJob = {
      ...job,
      status: "error",
      error: failureReason(err),
      updatedAt: Date.now(),
    };
    try {
      await putOfflineJob(failed);
    } catch (writeErr) {
      console.error("Failed to persist download error", writeErr);
    }
    broadcast(novelId, offlineStateOf(undefined, failed));
  }
}

function failureReason(err: unknown): OfflineJobError {
  if (err instanceof DownloadError) return err.reason;
  if (err instanceof DOMException && err.name === "QuotaExceededError") return "quota";
  console.error("Offline download failed", err);
  return "unknown";
}

async function downloadNovel(job: OfflineJob, signal: AbortSignal): Promise<void> {
  const novelId = job.novelId;
  const [apiNovel, apiToc] = await Promise.all([
    fetchJson<ApiNovel>(`/novels/${novelId}`, signal),
    fetchJson<ApiChaptersList>(`/novels/${novelId}/chapters`, signal),
  ]);
  const [existing, saved] = await Promise.all([
    getOfflineNovel(novelId),
    getSavedChapters(novelId),
  ]);

  const serverToc = apiToc.chapters.map((c) => ({ id: c.id, num: c.chapter_num, title: c.title }));
  const serverIds = new Set(serverToc.map((entry) => entry.id));
  const serverNums = new Set(serverToc.map((entry) => entry.num));
  const replaced = [...saved]
    .filter(([id, num]) => !serverIds.has(id) && serverNums.has(num))
    .map(([id]) => id);
  const kept = (existing?.toc ?? []).filter(
    (entry) => saved.has(entry.id) && !serverIds.has(entry.id) && !serverNums.has(entry.num),
  );
  const toc = sortToc([...serverToc, ...kept]);
  const tocIds = new Set(toc.map((entry) => entry.id));

  throwIfAborted(signal);
  let novel: OfflineNovel = {
    id: novelId,
    title: apiNovel.title,
    titleEn: apiNovel.title_en,
    author: apiNovel.author,
    description: apiNovel.description,
    status: apiNovel.status,
    ageRating: apiNovel.age_rating,
    coverUrl: normalizeCover(apiNovel.cover_url),
    toc,
    savedCount: saved.size,
    bytes: existing?.bytes ?? 0,
    complete: false,
    downloadedAt: existing?.downloadedAt ?? Date.now(),
    updatedAt: Date.now(),
  };
  await putOfflineNovel(novel);
  if (existing?.coverUrl && existing.coverUrl !== novel.coverUrl) {
    await removeCover(existing.coverUrl).catch((err) => {
      console.warn("Failed to remove stale cover", err);
    });
  }
  const coverTask = cacheCover(novel.coverUrl, signal);
  try {
    novel = await removeOfflineChapters(novel, replaced);
    replaced.forEach((id) => saved.delete(id));

    job.done = saved.size;
    job.total = toc.length;
    job.updatedAt = Date.now();
    await putOfflineJob(job);
    broadcast(novelId, offlineStateOf(undefined, job));

    const firstMissing = toc.find((entry) => !saved.has(entry.id));
    let after: number | null = firstMissing ? firstMissing.num - 1 : null;

    while (after !== null) {
      const page = await fetchOfflinePage(novelId, after, signal);
      if (page.next_after !== null && page.next_after <= after) {
        throw new DownloadError("unknown");
      }

      const fresh = page.chapters
        .filter((c) => c.novel_id === novelId && !saved.has(c.id))
        .map(toOfflineChapter);

      if (fresh.length > 0) {
        const added = fresh.filter((c) => !tocIds.has(c.id));
        added.forEach((c) => tocIds.add(c.id));
        const updated: OfflineNovel = {
          ...novel,
          toc: added.length > 0
            ? sortToc([...novel.toc, ...added.map((c) => ({ id: c.id, num: c.num, title: c.title }))])
            : novel.toc,
          savedCount: novel.savedCount + fresh.length,
          bytes: novel.bytes + fresh.reduce((sum, c) => sum + c.bytes, 0),
          updatedAt: Date.now(),
        };
        const progress: OfflineJob = {
          ...job,
          done: saved.size + fresh.length,
          total: updated.toc.length,
          updatedAt: Date.now(),
        };

        throwIfAborted(signal);
        await saveOfflineChapters(updated, fresh, progress);

        novel = updated;
        fresh.forEach((c) => saved.set(c.id, c.num));
        job.done = progress.done;
        job.total = progress.total;
        job.updatedAt = progress.updatedAt;
        broadcast(novelId, offlineStateOf(undefined, job));
      }

      after = page.next_after;
    }

    throwIfAborted(signal);
    await putOfflineNovel({
      ...novel,
      complete: novel.toc.every((entry) => saved.has(entry.id)),
      updatedAt: Date.now(),
    });
  } finally {
    await coverTask;
  }
}

function sortToc(entries: OfflineTocEntry[]): OfflineTocEntry[] {
  return entries.sort((a, b) => a.num - b.num);
}

function toOfflineChapter(chapter: ApiChapter): OfflineChapter {
  return {
    id: chapter.id,
    novelId: chapter.novel_id,
    num: chapter.chapter_num,
    title: chapter.title,
    content: chapter.content,
    source: chapter.source
      ? {
        name: chapter.source.name,
        logoUrl: chapter.source.logo_url,
        label: chapter.source.label,
      }
      : null,
    createdAt: chapter.created_at,
    bytes: byteSize(chapter.content),
  };
}

async function fetchOfflinePage(
  novelId: string,
  after: number,
  signal: AbortSignal,
): Promise<ApiOfflinePage> {
  const wait = lastBatchAt + BATCH_INTERVAL_MS - Date.now();
  if (wait > 0) await delay(wait, signal);
  lastBatchAt = Date.now();
  return fetchJson<ApiOfflinePage>(
    `/offline/novels/${novelId}/chapters?after=${after}&limit=${BATCH_LIMIT}`,
    signal,
  );
}

async function fetchJson<T>(path: string, signal: AbortSignal): Promise<T> {
  for (let attempt = 1;; attempt++) {
    let retryResponse: Response | null = null;
    try {
      const response = await fetch(`${API_URL}${path}`, {
        credentials: "same-origin",
        cache: "no-store",
        signal,
      });
      if (response.ok) return (await response.json()) as T;
      if (response.status === 401) throw new DownloadError("auth");
      if (response.status === 404) throw new DownloadError("not_found");
      if (response.status !== 429 && response.status < 500) {
        throw new DownloadError("unknown");
      }
      retryResponse = response;
    } catch (err) {
      if (signal.aborted || err instanceof DownloadError) throw err;
    }

    if (attempt >= MAX_ATTEMPTS) throw new DownloadError("network");
    await delay(retryDelay(retryResponse, attempt), signal);
  }
}

function retryDelay(response: Response | null, attempt: number): number {
  const retryAfter = Number(response?.headers.get("Retry-After"));
  if (retryAfter > 0) return retryAfter * 1000;
  return Math.min(MAX_BACKOFF_MS, 1000 * 2 ** (attempt - 1));
}

function delay(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal.aborted) {
      reject(abortError());
      return;
    }
    const onAbort = () => {
      clearTimeout(timer);
      reject(abortError());
    };
    const timer = setTimeout(() => {
      signal.removeEventListener("abort", onAbort);
      resolve();
    }, ms);
    signal.addEventListener("abort", onAbort, { once: true });
  });
}

function throwIfAborted(signal: AbortSignal): void {
  if (signal.aborted) throw abortError();
}

function abortError(): DOMException {
  return new DOMException("Download aborted", "AbortError");
}
