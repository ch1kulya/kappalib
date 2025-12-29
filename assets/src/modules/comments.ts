import { profileManager } from "./profile";

const API_URL = process.env.API_URL;
const TURNSTILE_COMMENTS_SITE_KEY =
  process.env.TURNSTILE_COMMENTS_SITE_KEY || "";
const PENDING_COMMENTS_KEY = "kappalib_pending_comments";
const PENDING_TTL = 3 * 60 * 60 * 1000;

interface PendingComment {
  id: string;
  chapterId: string;
  contentHtml: string;
  createdAt: number;
  userDisplayName: string;
  userAvatarSeed: string;
}

interface Comment {
  id: string;
  chapter_id: string;
  user_id: string;
  content_html: string;
  status: string;
  created_at: string;
  user_display_name: string;
  user_avatar_seed: string;
}

interface CommentsPage {
  comments: Comment[];
  page: number;
  page_size: number;
  total_count: number;
  total_pages: number;
}

function getPendingComments(): PendingComment[] {
  try {
    const raw = localStorage.getItem(PENDING_COMMENTS_KEY);
    if (!raw) return [];
    const comments: PendingComment[] = JSON.parse(raw);
    const now = Date.now();
    const valid = comments.filter((c) => now - c.createdAt < PENDING_TTL);
    if (valid.length !== comments.length) {
      localStorage.setItem(PENDING_COMMENTS_KEY, JSON.stringify(valid));
    }
    return valid;
  } catch {
    return [];
  }
}

function addPendingComment(comment: PendingComment): void {
  const comments = getPendingComments();
  comments.unshift(comment);
  localStorage.setItem(PENDING_COMMENTS_KEY, JSON.stringify(comments));
}

function removePendingComment(id: string): void {
  const comments = getPendingComments().filter((c) => c.id !== id);
  localStorage.setItem(PENDING_COMMENTS_KEY, JSON.stringify(comments));
}

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (seconds < 60) return "только что";
  if (minutes < 60) return `${minutes} мин. назад`;
  if (hours < 24) return `${hours} ч. назад`;
  if (days < 30) return `${days} дн. назад`;
  return date.toLocaleDateString("ru-RU");
}

function createCommentHTML(
  comment: Comment | PendingComment,
  isPending: boolean = false,
): string {
  const displayName =
    "user_display_name" in comment
      ? comment.user_display_name
      : comment.userDisplayName;
  const avatarSeed =
    "user_avatar_seed" in comment
      ? comment.user_avatar_seed
      : comment.userAvatarSeed;
  const contentHtml =
    "content_html" in comment ? comment.content_html : comment.contentHtml;
  const createdAt =
    "created_at" in comment
      ? comment.created_at
      : new Date(comment.createdAt).toISOString();

  const avatarUrl = `https://api.dicebear.com/9.x/bottts-neutral/svg?seed=${avatarSeed}&backgroundType=solid,gradientLinear`;

  return `
    <div class="comment-item${isPending ? " comment-pending" : ""}" data-comment-id="${comment.id}">
      <img src="${avatarUrl}" alt="${displayName}" class="comment-avatar" loading="lazy"/>
      <div class="comment-body">
        <div class="comment-header">
          <span class="comment-author">${displayName}</span>
          <span class="comment-date">${formatRelativeTime(createdAt)}</span>
          ${isPending ? '<span class="comment-moderation-badge">На модерации</span>' : ""}
        </div>
        <div class="comment-content">${contentHtml}</div>
      </div>
    </div>
  `;
}

function renderComments(
  container: HTMLElement,
  comments: Comment[],
  pendingComments: PendingComment[],
  chapterId: string,
): void {
  const chapterPending = pendingComments.filter(
    (c) => c.chapterId === chapterId,
  );
  const pendingIds = new Set(chapterPending.map((c) => c.id));
  const filteredComments = comments.filter((c) => !pendingIds.has(c.id));

  let html = "";

  chapterPending.forEach((c) => {
    html += createCommentHTML(c, true);
  });

  filteredComments.forEach((c) => {
    html += createCommentHTML(c, false);
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
): Promise<void> {
  const listEl = container.querySelector(".comments-list");
  if (listEl) {
    listEl.innerHTML =
      '<div class="comments-loading">Загрузка комментариев...</div>';
  }

  try {
    const res = await fetch(
      `${API_URL}/chapters/${chapterId}/comments?page=${page}`,
    );
    if (!res.ok) throw new Error("Failed to load comments");

    const data: CommentsPage = await res.json();
    const pendingComments = getPendingComments();

    data.comments.forEach((c) => {
      removePendingComment(c.id);
    });

    renderComments(container, data.comments, getPendingComments(), chapterId);
    renderPagination(container, data.page, data.total_pages, chapterId);

    const countEl = container.querySelector(".comments-count");
    if (countEl) {
      const pendingCount = getPendingComments().filter(
        (c) => c.chapterId === chapterId,
      ).length;
      const totalDisplay = data.total_count + pendingCount;
      countEl.textContent = `${totalDisplay}`;
    }
  } catch (err) {
    console.error("Failed to load comments", err);
    if (listEl) {
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
    counter.classList.toggle("warning", len > 900);
    counter.classList.toggle("error", len >= 1000);
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
        <textarea
          id="comment-textarea"
          class="comment-textarea"
          placeholder="Написать комментарий..."
          maxlength="1000"
          rows="2"
        ></textarea>
        <div class="comment-form-footer">
          <span id="comment-char-counter" class="comment-char-counter">0/1000</span>
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

function initFormHandlers(container: HTMLElement): void {
  const textarea = container.querySelector(
    "#comment-textarea",
  ) as HTMLTextAreaElement;
  const submitBtn = container.querySelector(
    "#comment-submit",
  ) as HTMLButtonElement;
  const chapterId = container.dataset.chapterId;

  if (!chapterId) return;

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
        const profileId = profileManager.getProfileId();
        const secretToken = localStorage.getItem("kappalib_secret_token");

        const res = await fetch(`${API_URL}/chapters/${chapterId}/comments`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-Profile-ID": profileId || "",
            "X-Secret-Token": secretToken || "",
          },
          body: JSON.stringify({
            content: content,
            turnstile_token: token,
          }),
        });

        if (!res.ok) {
          const err = await res.json();
          throw new Error(err.detail || "Failed to create comment");
        }

        const comment: Comment = await res.json();

        addPendingComment({
          id: comment.id,
          chapterId: comment.chapter_id,
          contentHtml: comment.content_html,
          createdAt: Date.now(),
          userDisplayName: comment.user_display_name,
          userAvatarSeed: comment.user_avatar_seed,
        });

        textarea.value = "";
        updateCharCounter(textarea);
        autoResizeTextarea(textarea);

        if (turnstileWidgetId) {
          (window as any).turnstile.reset(turnstileWidgetId);
        }

        await loadComments(container, chapterId);
      } catch (err) {
        console.error("Failed to submit comment", err);
        alert("Не удалось отправить комментарий. Попробуйте ещё раз.");
      } finally {
        submitBtn.disabled = false;
        submitBtn.textContent = "Отправить";
      }
    });
  }
}
