import { profileManager, getAvatarUrl } from "./profile";

const API_URL = process.env.API_URL;
const TURNSTILE_COMMENTS_SITE_KEY =
  process.env.TURNSTILE_COMMENTS_SITE_KEY || "";
const COMMENT_COOLDOWN = 30 * 1000;
const LAST_COMMENT_TIME_KEY = "kappalib_last_comment_time";

interface Comment {
  id: string;
  chapter_id: string;
  user_id: string;
  content_html: string;
  status: string;
  created_at: string;
  user_display_name: string;
  user_avatar_seed: string;
  user_has_custom_avatar: boolean;
  user_avatar_updated_at: number;
}

interface CommentsPage {
  comments: Comment[];
  page: number;
  page_size: number;
  total_count: number;
  total_pages: number;
}

function getLastCommentTime(): number {
  const stored = localStorage.getItem(LAST_COMMENT_TIME_KEY);
  return stored ? parseInt(stored, 10) : 0;
}

function setLastCommentTime(): void {
  localStorage.setItem(LAST_COMMENT_TIME_KEY, Date.now().toString());
}

function getRemainingCooldown(): number {
  const elapsed = Date.now() - getLastCommentTime();
  return Math.max(0, COMMENT_COOLDOWN - elapsed);
}

function startCooldownTimer(button: HTMLButtonElement): void {
  const updateButton = () => {
    const remaining = getRemainingCooldown();
    if (remaining <= 0) {
      button.disabled = false;
      button.textContent = "Отправить";
      return;
    }
    button.disabled = true;
    button.textContent = `Кулдаун ${Math.ceil(remaining / 1000)} сек.`;
    requestAnimationFrame(() => setTimeout(updateButton, 100));
  };
  updateButton();
}

function pluralize(
  count: number,
  one: string,
  few: string,
  many: string,
): string {
  const n = Math.abs(count) % 100;
  const n1 = n % 10;
  if (n > 10 && n < 20) return many;
  if (n1 === 1) return one;
  if (n1 >= 2 && n1 <= 4) return few;
  return many;
}

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  if (isNaN(date.getTime())) return "";

  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (seconds < 60) return "только что";
  if (minutes < 60)
    return `${minutes} ${pluralize(minutes, "минуту", "минуты", "минут")} назад`;
  if (hours < 24)
    return `${hours} ${pluralize(hours, "час", "часа", "часов")} назад`;
  if (days < 30)
    return `${days} ${pluralize(days, "день", "дня", "дней")} назад`;
  if (days < 365) {
    const months = Math.floor(days / 30);
    return `${months} ${pluralize(months, "месяц", "месяца", "месяцев")} назад`;
  }
  const years = Math.floor(days / 365);
  return `${years} ${pluralize(years, "год", "года", "лет")} назад`;
}

function createCommentHTML(comment: Comment): string {
  const avatarUrl = getAvatarUrl(
    comment.user_id,
    comment.user_has_custom_avatar,
    comment.user_avatar_seed,
    comment.user_avatar_updated_at,
  );

  let statusBadge = "";
  let extraClass = "";

  switch (comment.status) {
    case "pending":
      statusBadge =
        '<span class="comment-moderation-badge">На модерации</span>';
      extraClass = " comment-pending";
      break;
    case "rejected":
      statusBadge = '<span class="comment-rejected-badge">Отклонено</span>';
      extraClass = " comment-rejected";
      break;
    default:
      statusBadge = `<span class="comment-date">${formatRelativeTime(comment.created_at)}</span>`;
  }

  return `
    <div class="comment-item${extraClass}" data-comment-id="${comment.id}" data-comment-status="${comment.status}">
      <img src="${avatarUrl}" alt="${comment.user_display_name}" class="comment-avatar" loading="lazy"/>
      <div class="comment-body">
        <div class="comment-header">
          <span class="comment-author">${comment.user_display_name}</span>
          ${statusBadge}
        </div>
        <div class="comment-content">${comment.content_html}</div>
      </div>
    </div>
  `;
}

function renderComments(container: HTMLElement, comments: Comment[]): void {
  let html = "";

  comments.forEach((c) => {
    html += createCommentHTML(c);
  });

  const listEl = container.querySelector(".comments-list");
  if (listEl) {
    if (html) {
      listEl.innerHTML = html;
    } else {
      listEl.innerHTML =
        '<div class="comments-empty">Комментариев пока нет. Будьте первым!</div>';
    }
  }
}

function renderPagination(
  container: HTMLElement,
  page: number,
  totalPages: number,
  chapterId: string,
): void {
  const paginationEl = container.querySelector(".comments-pagination");
  if (!paginationEl || totalPages <= 1) {
    if (paginationEl) paginationEl.innerHTML = "";
    return;
  }

  let html = "";

  if (page > 1) {
    html += `<button class="page-link prev-next" data-page="${page - 1}">←</button>`;
  } else {
    html += `<span class="page-link prev-next disabled">←</span>`;
  }

  const pages = calculatePagination(page, totalPages);
  pages.forEach((p) => {
    if (p === -1) {
      html += `<span class="page-ellipsis">...</span>`;
    } else if (p === page) {
      html += `<span class="page-link active">${p}</span>`;
    } else {
      html += `<button class="page-link" data-page="${p}">${p}</button>`;
    }
  });

  if (page < totalPages) {
    html += `<button class="page-link prev-next" data-page="${page + 1}">→</button>`;
  } else {
    html += `<span class="page-link prev-next disabled">→</span>`;
  }

  paginationEl.innerHTML = html;
}

function calculatePagination(current: number, total: number): number[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1);
  }

  const pages: number[] = [1];
  let start = current - 2;
  let end = current + 2;

  if (start <= 2) {
    start = 2;
    end = 5;
  }

  if (end >= total - 1) {
    end = total - 1;
    start = total - 4;
  }

  if (start > 2) pages.push(-1);

  for (let i = start; i <= end; i++) {
    pages.push(i);
  }

  if (end < total - 1) pages.push(-1);

  pages.push(total);
  return pages;
}

async function loadComments(
  container: HTMLElement,
  chapterId: string,
  page: number = 1,
  noCache: boolean = false,
  showLoader: boolean = true,
): Promise<void> {
  const listEl = container.querySelector(".comments-list");
  if (listEl && showLoader) {
    listEl.innerHTML =
      '<div class="comments-loading">Загрузка комментариев...</div>';
  }

  try {
    const res = await fetch(
      `${API_URL}/chapters/${chapterId}/comments?page=${page}`,
      noCache
        ? { credentials: "include", cache: "no-cache" }
        : { credentials: "include" },
    );
    if (!res.ok) throw new Error("Failed to load comments");

    const data: CommentsPage = await res.json();

    renderComments(container, data.comments);
    renderPagination(container, data.page, data.total_pages, chapterId);

    const countEl = container.querySelector(".comments-count");
    if (countEl) {
      countEl.textContent = `${data.total_count}`;
    }
  } catch (err) {
    console.error("Failed to load comments", err);
    if (listEl && showLoader) {
      listEl.innerHTML =
        '<div class="comments-error">Не удалось загрузить комментарии</div>';
    }
  }
}

let turnstileWidgetId: string | null = null;
let turnstileToken: string | null = null;
let turnstileLoaded = false;

function loadTurnstileScript(): Promise<void> {
  return new Promise((resolve) => {
    if ((window as any).turnstile) {
      resolve();
      return;
    }

    const existing = document.querySelector(
      'script[src*="challenges.cloudflare.com/turnstile"]',
    );
    if (existing) {
      const checkLoaded = setInterval(() => {
        if ((window as any).turnstile) {
          clearInterval(checkLoaded);
          resolve();
        }
      }, 100);
      return;
    }

    const script = document.createElement("script");
    script.src =
      "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";
    script.async = true;
    script.onload = () => {
      const checkLoaded = setInterval(() => {
        if ((window as any).turnstile) {
          clearInterval(checkLoaded);
          resolve();
        }
      }, 100);
    };
    document.head.appendChild(script);
  });
}

async function initTurnstileForComments(container: HTMLElement): Promise<void> {
  if (!TURNSTILE_COMMENTS_SITE_KEY) {
    console.warn("TURNSTILE_COMMENTS_SITE_KEY not set");
    return;
  }

  await loadTurnstileScript();

  const turnstileContainer = container.querySelector(
    "#comments-turnstile-container",
  );
  if (!turnstileContainer || turnstileLoaded) return;

  turnstileLoaded = true;

  turnstileWidgetId = (window as any).turnstile.render(turnstileContainer, {
    sitekey: TURNSTILE_COMMENTS_SITE_KEY,
    size: "invisible",
    callback: (token: string) => {
      turnstileToken = token;
    },
    "expired-callback": () => {
      turnstileToken = null;
    },
    "error-callback": () => {
      turnstileToken = null;
    },
  });
}

let tokenPromise: Promise<string | null> | null = null;

async function getTurnstileToken(): Promise<string | null> {
  if (!turnstileWidgetId) return null;

  if (tokenPromise) {
    return tokenPromise;
  }

  tokenPromise = new Promise((resolve) => {
    if (turnstileToken) {
      const token = turnstileToken;
      turnstileToken = null;
      (window as any).turnstile.reset(turnstileWidgetId);
      resolve(token);
      return;
    }

    (window as any).turnstile.reset(turnstileWidgetId);
    (window as any).turnstile.execute(turnstileWidgetId);

    let attempts = 0;
    const maxAttempts = 300;

    const checkToken = setInterval(() => {
      attempts++;
      if (turnstileToken) {
        clearInterval(checkToken);
        const token = turnstileToken;
        turnstileToken = null;
        resolve(token);
      } else if (attempts >= maxAttempts) {
        clearInterval(checkToken);
        resolve(null);
      }
    }, 100);
  });

  const result = await tokenPromise;
  tokenPromise = null;
  return result;
}

function updateCharCounter(textarea: HTMLTextAreaElement): void {
  const counter = document.getElementById("comment-char-counter");
  if (counter) {
    const len = textarea.value.length;
    counter.textContent = `${len}/1000`;
    counter.classList.toggle("count-warning", len > 900);
    counter.classList.toggle("count-error", len >= 1000);
  }
}

function autoResizeTextarea(textarea: HTMLTextAreaElement): void {
  textarea.style.height = "auto";
  const lineHeight = parseInt(getComputedStyle(textarea).lineHeight) || 24;
  const maxHeight = lineHeight * 8;
  const newHeight = Math.min(textarea.scrollHeight, maxHeight);
  textarea.style.height = newHeight + "px";
  textarea.style.overflowY =
    textarea.scrollHeight > maxHeight ? "auto" : "hidden";
}

export function initComments(): void {
  const container = document.getElementById("comments-section");
  if (!container) return;

  const chapterId = container.dataset.chapterId;
  if (!chapterId) return;

  renderCommentForm(container);
  loadComments(container, chapterId);

  container.addEventListener("click", (e) => {
    const target = e.target as HTMLElement;

    const spoiler = target.closest(".spoiler") as HTMLElement | null;
    if (spoiler) {
      spoiler.classList.toggle("revealed");
      return;
    }

    const pageBtn = target.closest(".page-link[data-page]") as HTMLElement;
    if (pageBtn && !pageBtn.classList.contains("disabled")) {
      const page = parseInt(pageBtn.dataset.page || "1", 10);
      loadComments(container, chapterId, page);

      container.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  });
}

function renderCommentForm(container: HTMLElement): void {
  const formWrapper = container.querySelector(".comment-form-wrapper");
  if (!formWrapper) return;

  if (profileManager.isLoggedIn()) {
    formWrapper.innerHTML = `
      <div class="comment-form">
        <div class="comment-toolbar">
          <button type="button" class="toolbar-btn" data-action="h1" title="Заголовок 1">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 12h8"/><path d="M4 18V6"/><path d="M12 18V6"/><path d="m17 12 3-2v8"/></svg>
          </button>
          <button type="button" class="toolbar-btn" data-action="h2" title="Заголовок 2">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 12h8"/><path d="M4 18V6"/><path d="M12 18V6"/><path d="M21 18h-4c0-4 4-3 4-6 0-1.5-2-2.5-4-1"/></svg>
          </button>
          <button type="button" class="toolbar-btn" data-action="h3" title="Заголовок 3">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 12h8"/><path d="M4 18V6"/><path d="M12 18V6"/><path d="M17.5 10.5c1.7-1 3.5 0 3.5 1.5a2 2 0 0 1-2 2"/><path d="M17 17.5c2 1.5 4 .3 4-1.5a2 2 0 0 0-2-2"/></svg>
          </button>
          <span class="toolbar-separator"></span>
          <button type="button" class="toolbar-btn" data-action="bold" title="Жирный">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 12h9a4 4 0 0 1 0 8H7a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1h7a4 4 0 0 1 0 8"/></svg>
          </button>
          <button type="button" class="toolbar-btn" data-action="italic" title="Курсив">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="19" x2="10" y1="4" y2="4"/><line x1="14" x2="5" y1="20" y2="20"/><line x1="15" x2="9" y1="4" y2="20"/></svg>
          </button>
          <button type="button" class="toolbar-btn" data-action="spoiler" title="Спойлер">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0"/><circle cx="12" cy="12" r="3"/></svg>
          </button>
          <button type="button" class="toolbar-btn" data-action="quote" title="Цитата">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 3a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2 1 1 0 0 1 1 1v1a2 2 0 0 1-2 2 1 1 0 0 0-1 1v2a1 1 0 0 0 1 1 6 6 0 0 0 6-6V5a2 2 0 0 0-2-2z"/><path d="M5 3a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2 1 1 0 0 1 1 1v1a2 2 0 0 1-2 2 1 1 0 0 0-1 1v2a1 1 0 0 0 1 1 6 6 0 0 0 6-6V5a2 2 0 0 0-2-2z"/></svg>
          </button>
          <span class="toolbar-separator"></span>
          <button type="button" class="toolbar-btn" data-action="image" title="Изображение (до 5 МБ)">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>
          </button>
          <input type="file" id="comment-image-input" accept="image/jpeg,image/png,image/gif" style="display:none"/>
        </div>
        <textarea
          id="comment-textarea"
          class="comment-textarea"
          placeholder="Написать комментарий..."
          maxlength="1000"
          rows="2"
        ></textarea>
        <div class="comment-form-footer">
          <span id="comment-char-counter" class="comment-char-counter">0/1000</span>
          <a href="/markdown" target="_blank" class="comment-markdown-hint">Форматирование</a>
          <div id="comments-turnstile-container"></div>
          <button id="comment-submit" class="action-btn btn-primary comment-submit-btn">Отправить</button>
        </div>
      </div>
    `;
    initFormHandlers(container);
  } else {
    formWrapper.innerHTML = `
      <div class="comment-form comment-form-guest">
        <p class="comment-guest-message">Войдите или создайте аккаунт, чтобы писать комментарии</p>
      </div>
    `;
  }
}

function wrapSelection(
  textarea: HTMLTextAreaElement,
  before: string,
  after: string,
): void {
  const start = textarea.selectionStart;
  const end = textarea.selectionEnd;
  const selected = textarea.value.substring(start, end);
  const value = textarea.value;

  if (
    start >= before.length &&
    value.substring(start - before.length, start) === before &&
    value.substring(end, end + after.length) === after
  ) {
    textarea.setRangeText(
      selected,
      start - before.length,
      end + after.length,
      "select",
    );
    textarea.focus();
    updateCharCounter(textarea);
    autoResizeTextarea(textarea);
    return;
  }

  if (
    selected.startsWith(before) &&
    selected.endsWith(after) &&
    selected.length >= before.length + after.length
  ) {
    const unwrapped = selected.slice(before.length, -after.length);
    textarea.setRangeText(unwrapped, start, end, "select");
    textarea.focus();
    updateCharCounter(textarea);
    autoResizeTextarea(textarea);
    return;
  }

  if (selected.length === 0) {
    const textBefore = value.substring(0, start);
    const textAfter = value.substring(start);
    const lastOpenIndex = textBefore.lastIndexOf(before);

    if (lastOpenIndex !== -1) {
      const closeIndex = textAfter.indexOf(after);
      if (closeIndex !== -1) {
        const between = value.substring(
          lastOpenIndex + before.length,
          start + closeIndex,
        );
        if (!between.includes(before) && !between.includes(after)) {
          textarea.setRangeText(
            between,
            lastOpenIndex,
            start + closeIndex + after.length,
            "end",
          );
          textarea.selectionStart = textarea.selectionEnd =
            lastOpenIndex + between.length;
          textarea.focus();
          updateCharCounter(textarea);
          autoResizeTextarea(textarea);
          return;
        }
      }
    }
  }

  const replacement = `${before}${selected}${after}`;
  textarea.setRangeText(replacement, start, end, "end");
  if (selected.length === 0) {
    textarea.selectionStart = textarea.selectionEnd = start + before.length;
  }
  textarea.focus();
  updateCharCounter(textarea);
  autoResizeTextarea(textarea);
}

const headingPrefixes = ["# ", "## ", "### "];

function insertLinePrefix(
  textarea: HTMLTextAreaElement,
  prefix: string,
  alternates?: string[],
): void {
  const start = textarea.selectionStart;
  const lineStart = textarea.value.lastIndexOf("\n", start - 1) + 1;
  const lineEnd = textarea.value.indexOf("\n", start);
  const lineContent = textarea.value.substring(
    lineStart,
    lineEnd === -1 ? textarea.value.length : lineEnd,
  );

  if (lineContent.startsWith(prefix)) {
    const before = textarea.value.substring(0, lineStart);
    const after = textarea.value.substring(lineStart + prefix.length);
    textarea.value = `${before}${after}`;
    textarea.selectionStart = textarea.selectionEnd = Math.max(
      lineStart,
      start - prefix.length,
    );
    textarea.focus();
    updateCharCounter(textarea);
    autoResizeTextarea(textarea);
    return;
  }

  if (alternates) {
    for (const alt of alternates) {
      if (alt !== prefix && lineContent.startsWith(alt)) {
        const before = textarea.value.substring(0, lineStart);
        const after = textarea.value.substring(lineStart + alt.length);
        textarea.value = `${before}${prefix}${after}`;
        textarea.selectionStart = textarea.selectionEnd =
          start - alt.length + prefix.length;
        textarea.focus();
        updateCharCounter(textarea);
        autoResizeTextarea(textarea);
        return;
      }
    }
  }

  const before = textarea.value.substring(0, lineStart);
  const after = textarea.value.substring(lineStart);
  textarea.value = `${before}${prefix}${after}`;
  textarea.selectionStart = textarea.selectionEnd = start + prefix.length;
  textarea.focus();
  updateCharCounter(textarea);
  autoResizeTextarea(textarea);
}

function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result as string;
      resolve(result.split(",")[1]);
    };
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });
}

let isUploadingImage = false;
let uploadAbortController: AbortController | null = null;
let uploadAnimationInterval: ReturnType<typeof setInterval> | null = null;

async function uploadCommentImage(
  file: File,
  textarea: HTMLTextAreaElement,
): Promise<void> {
  if (isUploadingImage) return;

  if (file.size > 5 * 1024 * 1024) {
    alert("Файл слишком большой (максимум 5 МБ)");
    return;
  }

  isUploadingImage = true;
  const imageBtn = document.querySelector(
    '.toolbar-btn[data-action="image"]',
  ) as HTMLButtonElement | null;
  if (imageBtn) imageBtn.disabled = true;

  const base64 = await fileToBase64(file);
  const fileName = file.name.replace(/\.[^.]+$/, "");

  const basePlaceholder = `![Загрузка ${fileName}`;
  let dotCount = 1;
  const getPlaceholder = () => `${basePlaceholder}${".".repeat(dotCount)}]()`;

  const start = textarea.selectionStart;
  const needsNewlineBefore = start > 0 && textarea.value[start - 1] !== "\n";
  const prefix = needsNewlineBefore ? "\n" : "";

  let currentPlaceholder = getPlaceholder();
  textarea.setRangeText(prefix + currentPlaceholder, start, start, "end");
  updateCharCounter(textarea);
  autoResizeTextarea(textarea);

  uploadAnimationInterval = setInterval(() => {
    const placeholderStart = textarea.value.indexOf(currentPlaceholder);
    if (placeholderStart !== -1) {
      dotCount = (dotCount % 3) + 1;
      const newPlaceholder = getPlaceholder();
      textarea.setRangeText(
        newPlaceholder,
        placeholderStart,
        placeholderStart + currentPlaceholder.length,
        "end",
      );
      currentPlaceholder = newPlaceholder;
    }
  }, 400);

  uploadAbortController = new AbortController();

  try {
    const res = await fetch(`${API_URL}/comments/image`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ image: base64 }),
      signal: uploadAbortController.signal,
    });

    if (!res.ok) {
      const err = await res.json().catch(() => null);
      throw new Error(err?.detail || `HTTP error! status: ${res.status}`);
    }

    const data: { url: string } = await res.json();
    const markdown = `![${fileName}](${data.url})`;
    const placeholderStart = textarea.value.indexOf(currentPlaceholder);
    if (placeholderStart !== -1) {
      textarea.setRangeText(
        markdown,
        placeholderStart,
        placeholderStart + currentPlaceholder.length,
        "end",
      );
    }
  } catch (err) {
    if (uploadAnimationInterval) {
      clearInterval(uploadAnimationInterval);
      uploadAnimationInterval = null;
    }

    const placeholderStart = textarea.value.indexOf(currentPlaceholder);
    if (placeholderStart !== -1) {
      const needsNewlineBefore =
        start > 0 && textarea.value[start - 1] !== "\n";
      const removeStart = needsNewlineBefore
        ? placeholderStart - 1
        : placeholderStart;
      textarea.setRangeText(
        "",
        removeStart,
        placeholderStart + currentPlaceholder.length,
        "end",
      );
    }

    if (err instanceof DOMException && err.name === "AbortError") {
    } else {
      console.error("Failed to upload image:", err);
      const errorMessage =
        err instanceof Error ? err.message : "Не удалось загрузить изображение";
      alert(errorMessage);
    }
  } finally {
    if (uploadAnimationInterval) {
      clearInterval(uploadAnimationInterval);
      uploadAnimationInterval = null;
    }
    isUploadingImage = false;
    uploadAbortController = null;
    if (imageBtn) imageBtn.disabled = false;
  }

  updateCharCounter(textarea);
  autoResizeTextarea(textarea);
}

function initFormHandlers(container: HTMLElement): void {
  const textarea = container.querySelector(
    "#comment-textarea",
  ) as HTMLTextAreaElement;
  const submitBtn = container.querySelector(
    "#comment-submit",
  ) as HTMLButtonElement;
  const chapterId = container.dataset.chapterId;
  const imageInput = container.querySelector(
    "#comment-image-input",
  ) as HTMLInputElement;

  if (!chapterId) return;

  if (submitBtn && getRemainingCooldown() > 0) {
    startCooldownTimer(submitBtn);
  }

  container.querySelectorAll(".toolbar-btn").forEach((btn) => {
    btn.addEventListener("mousedown", (e) => {
      e.preventDefault();
      if (!textarea) return;
      const action = (btn as HTMLElement).dataset.action;
      switch (action) {
        case "h1":
          insertLinePrefix(textarea, "# ", headingPrefixes);
          break;
        case "h2":
          insertLinePrefix(textarea, "## ", headingPrefixes);
          break;
        case "h3":
          insertLinePrefix(textarea, "### ", headingPrefixes);
          break;
        case "bold":
          wrapSelection(textarea, "**", "**");
          break;
        case "italic":
          wrapSelection(textarea, "*", "*");
          break;
        case "spoiler":
          wrapSelection(textarea, "||", "||");
          break;
        case "quote":
          insertLinePrefix(textarea, "> ");
          break;
        case "image":
          if (!isUploadingImage) imageInput?.click();
          break;
      }
    });
  });

  imageInput?.addEventListener("change", async () => {
    const file = imageInput.files?.[0];
    if (!file || !textarea) return;
    imageInput.value = "";
    initTurnstileForComments(container);
    await uploadCommentImage(file, textarea);
  });

  if (textarea) {
    textarea.addEventListener("input", () => {
      updateCharCounter(textarea);
      autoResizeTextarea(textarea);
    });

    textarea.addEventListener("focus", () => {
      initTurnstileForComments(container);
    });
  }

  if (submitBtn && textarea) {
    submitBtn.addEventListener("click", async () => {
      const content = textarea.value.trim();
      if (!content) return;
      if (content.length > 1000) {
        alert("Комментарий слишком длинный (максимум 1000 символов)");
        return;
      }

      if (!profileManager.isLoggedIn()) {
        alert("Войдите в аккаунт, чтобы оставить комментарий");
        return;
      }

      submitBtn.disabled = true;
      submitBtn.textContent = "Проверка...";

      const token = await getTurnstileToken();
      if (!token) {
        alert("Не удалось пройти проверку. Попробуйте ещё раз.");
        submitBtn.disabled = false;
        submitBtn.textContent = "Отправить";
        return;
      }

      submitBtn.textContent = "Отправка...";

      try {
        const res = await fetch(`${API_URL}/chapters/${chapterId}/comments`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({
            content: content,
            turnstile_token: token,
          }),
        });

        if (!res.ok) {
          const err = await res.json();
          throw new Error(err.detail || "Failed to create comment");
        }

        textarea.value = "";
        updateCharCounter(textarea);
        autoResizeTextarea(textarea);

        if (turnstileWidgetId) {
          (window as any).turnstile.reset(turnstileWidgetId);
        }

        await loadComments(container, chapterId, 1, true, false);
        setLastCommentTime();
        startCooldownTimer(submitBtn);
      } catch (err) {
        console.error("Failed to submit comment", err);
        alert("Не удалось отправить комментарий. Попробуйте ещё раз.");
      } finally {
        if (!submitBtn.disabled) {
          submitBtn.disabled = false;
          submitBtn.textContent = "Отправить";
        }
      }
    });
  }
}
