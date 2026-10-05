export const OFFLINE_CHANNEL = "kappalib-offline";
export const OFFLINE_FLAG_KEY = "kappalib_offline_enabled";
export const NOVEL_ID_RE = /^nvl_[a-z0-9]{8}$/;
export const OFFLINE_NOVEL_PATH_RE = /^\/(nvl_[a-z0-9]+)\/?$/;
export const OFFLINE_CHAPTER_PATH_RE = /^\/(nvl_[a-z0-9]+)\/chapter\/(chp_[a-z0-9]+)\/?$/;

const DB_NAME = "kappalib-offline";
const DB_VERSION = 1;
const NOVELS = "novels";
const CHAPTERS = "chapters";
const JOBS = "jobs";
const BY_NOVEL = "byNovel";

export interface OfflineTocEntry {
  id: string;
  num: number;
  title: string;
}

export interface OfflineSource {
  name: string;
  logoUrl: string | null;
  label: string;
}

export interface OfflineNovel {
  id: string;
  title: string;
  titleEn: string;
  author: string;
  description: string;
  status: string;
  ageRating: string | null;
  coverUrl: string | null;
  toc: OfflineTocEntry[];
  savedCount: number;
  bytes: number;
  complete: boolean;
  downloadedAt: number;
  updatedAt: number;
}

export interface OfflineChapter {
  id: string;
  novelId: string;
  num: number;
  title: string;
  content: string;
  source: OfflineSource | null;
  createdAt: string;
  bytes: number;
}

export type OfflineJobError =
  | "auth"
  | "network"
  | "quota"
  | "not_found"
  | "unknown";

export interface OfflineJob {
  novelId: string;
  status: "queued" | "running" | "error";
  error: OfflineJobError | null;
  done: number;
  total: number;
  updatedAt: number;
}

export type OfflineState =
  | { kind: "none" }
  | { kind: "queued"; done: number; total: number }
  | { kind: "downloading"; done: number; total: number }
  | { kind: "error"; error: OfflineJobError; done: number; total: number }
  | { kind: "ready"; saved: number; total: number; bytes: number };

export type OfflineCommand =
  | { type: "download"; novelId: string }
  | { type: "cancel"; novelId: string }
  | { type: "delete"; novelId: string }
  | { type: "delete-all" }
  | { type: "resume" };

export interface OfflineBroadcast {
  novelId: string | null;
  state: OfflineState;
}

let dbPromise: Promise<IDBDatabase> | null = null;

export function openOfflineDb(): Promise<IDBDatabase> {
  if (!dbPromise) {
    dbPromise = new Promise<IDBDatabase>((resolve, reject) => {
      const request = indexedDB.open(DB_NAME, DB_VERSION);
      request.onupgradeneeded = () => {
        const db = request.result;
        if (!db.objectStoreNames.contains(NOVELS)) {
          db.createObjectStore(NOVELS, { keyPath: "id" });
        }
        if (!db.objectStoreNames.contains(CHAPTERS)) {
          const chapters = db.createObjectStore(CHAPTERS, { keyPath: "id" });
          chapters.createIndex(BY_NOVEL, ["novelId", "num"]);
        }
        if (!db.objectStoreNames.contains(JOBS)) {
          db.createObjectStore(JOBS, { keyPath: "novelId" });
        }
      };
      request.onsuccess = () => {
        const db = request.result;
        db.onversionchange = () => {
          db.close();
          dbPromise = null;
        };
        db.onclose = () => {
          dbPromise = null;
        };
        resolve(db);
      };
      request.onerror = () => {
        dbPromise = null;
        reject(request.error);
      };
    });
  }
  return dbPromise;
}

function promisify<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

function completion(tx: IDBTransaction): Promise<void> {
  return new Promise((resolve, reject) => {
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
    tx.onabort = () => reject(tx.error ?? new DOMException("Transaction aborted", "AbortError"));
  });
}

function novelRange(novelId: string): IDBKeyRange {
  return IDBKeyRange.bound([novelId, -Infinity], [novelId, Infinity]);
}

async function readOne<T>(storeName: string, key: string): Promise<T | undefined> {
  const db = await openOfflineDb();
  const request: IDBRequest<T | undefined> = db
    .transaction(storeName, "readonly")
    .objectStore(storeName)
    .get(key);
  return promisify(request);
}

async function readAll<T>(storeName: string): Promise<T[]> {
  const db = await openOfflineDb();
  const request: IDBRequest<T[]> = db
    .transaction(storeName, "readonly")
    .objectStore(storeName)
    .getAll();
  return promisify(request);
}

async function writeOne(storeName: string, value: OfflineNovel | OfflineJob): Promise<void> {
  const db = await openOfflineDb();
  const tx = db.transaction(storeName, "readwrite");
  tx.objectStore(storeName).put(value);
  return completion(tx);
}

export function getOfflineNovel(novelId: string): Promise<OfflineNovel | undefined> {
  return readOne<OfflineNovel>(NOVELS, novelId);
}

export function getOfflineChapter(chapterId: string): Promise<OfflineChapter | undefined> {
  return readOne<OfflineChapter>(CHAPTERS, chapterId);
}

export function getOfflineJob(novelId: string): Promise<OfflineJob | undefined> {
  return readOne<OfflineJob>(JOBS, novelId);
}

export function listOfflineNovels(): Promise<OfflineNovel[]> {
  return readAll<OfflineNovel>(NOVELS);
}

export function listOfflineJobs(): Promise<OfflineJob[]> {
  return readAll<OfflineJob>(JOBS);
}

export function putOfflineNovel(novel: OfflineNovel): Promise<void> {
  return writeOne(NOVELS, novel);
}

export function putOfflineJob(job: OfflineJob): Promise<void> {
  return writeOne(JOBS, job);
}

export async function deleteOfflineJob(novelId: string): Promise<void> {
  const db = await openOfflineDb();
  const tx = db.transaction(JOBS, "readwrite");
  tx.objectStore(JOBS).delete(novelId);
  return completion(tx);
}

export async function getSavedChapters(novelId: string): Promise<Map<string, number>> {
  const db = await openOfflineDb();
  const index = db
    .transaction(CHAPTERS, "readonly")
    .objectStore(CHAPTERS)
    .index(BY_NOVEL);
  const saved = new Map<string, number>();

  return new Promise((resolve, reject) => {
    const request = index.openKeyCursor(novelRange(novelId));
    request.onsuccess = () => {
      const cursor = request.result;
      if (!cursor) {
        resolve(saved);
        return;
      }
      const [, num] = cursor.key as [string, number];
      saved.set(cursor.primaryKey as string, num);
      cursor.continue();
    };
    request.onerror = () => reject(request.error);
  });
}

export async function saveOfflineChapters(
  novel: OfflineNovel,
  chapters: OfflineChapter[],
  job: OfflineJob,
): Promise<void> {
  const db = await openOfflineDb();
  const tx = db.transaction([NOVELS, CHAPTERS, JOBS], "readwrite");
  const store = tx.objectStore(CHAPTERS);
  chapters.forEach((chapter) => store.put(chapter));
  tx.objectStore(NOVELS).put(novel);
  tx.objectStore(JOBS).put(job);
  return completion(tx);
}

export async function removeOfflineChapters(
  novel: OfflineNovel,
  chapterIds: string[],
): Promise<OfflineNovel> {
  if (chapterIds.length === 0) return novel;

  const records = await Promise.all(chapterIds.map(getOfflineChapter));
  const existing = records.filter((c): c is OfflineChapter => c !== undefined);
  const removedBytes = existing.reduce((sum, c) => sum + c.bytes, 0);
  const updated: OfflineNovel = {
    ...novel,
    savedCount: Math.max(0, novel.savedCount - existing.length),
    bytes: Math.max(0, novel.bytes - removedBytes),
  };

  const db = await openOfflineDb();
  const tx = db.transaction([NOVELS, CHAPTERS], "readwrite");
  const store = tx.objectStore(CHAPTERS);
  existing.forEach((chapter) => store.delete(chapter.id));
  tx.objectStore(NOVELS).put(updated);
  await completion(tx);
  return updated;
}

export async function deleteOfflineNovel(novelId: string): Promise<void> {
  const db = await openOfflineDb();
  const tx = db.transaction([NOVELS, CHAPTERS, JOBS], "readwrite");
  const chapters = tx.objectStore(CHAPTERS);
  const request = chapters.index(BY_NOVEL).openKeyCursor(novelRange(novelId));
  request.onsuccess = () => {
    const cursor = request.result;
    if (!cursor) return;
    chapters.delete(cursor.primaryKey);
    cursor.continue();
  };
  tx.objectStore(NOVELS).delete(novelId);
  tx.objectStore(JOBS).delete(novelId);
  return completion(tx);
}

export async function clearOfflineData(): Promise<void> {
  const db = await openOfflineDb();
  const tx = db.transaction([NOVELS, CHAPTERS, JOBS], "readwrite");
  tx.objectStore(NOVELS).clear();
  tx.objectStore(CHAPTERS).clear();
  tx.objectStore(JOBS).clear();
  return completion(tx);
}

export function offlineStateOf(
  novel: OfflineNovel | undefined,
  job: OfflineJob | undefined,
): OfflineState {
  if (job) {
    if (job.status === "error") {
      return {
        kind: "error",
        error: job.error ?? "unknown",
        done: job.done,
        total: job.total,
      };
    }
    return {
      kind: job.status === "queued" ? "queued" : "downloading",
      done: job.done,
      total: job.total,
    };
  }
  if (novel) {
    return {
      kind: "ready",
      saved: novel.savedCount,
      total: novel.toc.length,
      bytes: novel.bytes,
    };
  }
  return { kind: "none" };
}

export async function getOfflineState(novelId: string): Promise<OfflineState> {
  const [novel, job] = await Promise.all([
    getOfflineNovel(novelId),
    getOfflineJob(novelId),
  ]);
  return offlineStateOf(novel, job);
}

export function byteSize(value: string): number {
  return new Blob([value]).size;
}
