import { profileManager, getAvatarUrl, updateProfileBadges } from "./profile";

const API_URL = process.env.API_URL;
const TURNSTILE_COMMENTS_SITE_KEY =
  process.env.TURNSTILE_COMMENTS_SITE_KEY || "";
const SMARTCAPTCHA_SITE_KEY = process.env.SMARTCAPTCHA_SITE_KEY || "";
const COMMENT_COOLDOWN = 30 * 1000;
const LAST_COMMENT_TIME_KEY = "kappalib_last_comment_time";

const initializedContainers = new WeakSet<HTMLElement>();

document.addEventListener("click", (e) => {
  const target = e.target as HTMLElement;
  if (!target.closest(".comment-menu")) {
    document.querySelectorAll(".comment-menu.active").forEach((m) => {
      m.classList.remove("active");
      m.querySelector(".comment-menu-btn")?.setAttribute("aria-expanded", "false");
    });
  }
});

document.addEventListener("keydown", (e) => {
  if (e.key === "Escape") {
    document.querySelectorAll(".comment-menu.active").forEach((m) => {
      m.classList.remove("active");
      m.querySelector(".comment-menu-btn")?.setAttribute("aria-expanded", "false");
    });
  }
});

interface CommentAnswer {
  id: string;
  comment_id: string;
  user_id: string;
  content_html: string;
  status: string;
  created_at: string;
  user_display_name: string;
  user_avatar_seed: string;
  user_has_custom_avatar: boolean;
  user_avatar_updated_at: number;
  is_new?: boolean;
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
  user_has_custom_avatar: boolean;
  user_avatar_updated_at: number;
  score: number;
  user_vote: number;
  answers?: CommentAnswer[];
  chapter_num?: number;
  novel_id?: string;
  novel_title?: string;
}

interface CommentsPage {
  comments: Comment[];
  page: number;
  page_size: number;
  total_count: number;
  total_pages: number;
}

interface CommentStatDay {
  day: string;
  rating: number;
  replies: number;
}

interface CommentStats {
  days: CommentStatDay[];
  rating: number;
  replies: number;
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
    if (!button.isConnected) return;
    const remaining = getRemainingCooldown();
    if (remaining <= 0) {
      button.disabled = false;
      button.textContent = "Отправить";
      return;
    }
    button.disabled = true;
    button.textContent = `Кулдаун ${Math.ceil(remaining / 1000)} сек.`;
    setTimeout(updateButton, 100);
  };
  updateButton();
}

function startAllCooldownTimers(): void {
  document
    .querySelectorAll<HTMLButtonElement>(
      "#comment-submit, .comment-reply-submit-btn",
    )
    .forEach((btn) => {
      startCooldownTimer(btn);
    });
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

function createCommentHTML(
  comment: Comment,
  options?: { chapterUrl?: string },
): string {
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
    case "deleted":
      statusBadge = '<span class="comment-deleted-badge">Удалено</span>';
      extraClass = " comment-rejected";
      break;
    default:
      statusBadge = `<span class="comment-date">${formatRelativeTime(comment.created_at)}</span>`;
  }

  const isApproved = comment.status === "approved";
  const isOwn = comment.user_id === profileManager.getProfileId();

  const upActive = comment.user_vote === 1 ? " vote-active" : "";
  const downActive = comment.user_vote === -1 ? " vote-active" : "";

  const voteHTML = isApproved
    ? `<div class="comment-votes">
        <button class="vote-btn vote-up${upActive}" data-vote="1" data-comment-id="${comment.id}" aria-label="Лайк">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 19a1 1 0 0 0 1 1h4a1 1 0 0 0 1-1v-6a1 1 0 0 1 1-1h3.293a.707.707 0 0 0 .5-1.207l-7.086-7.086a1 1 0 0 0-1.414 0l-7.086 7.086a.707.707 0 0 0 .5 1.207H8a1 1 0 0 1 1 1z"/>
          </svg>
        </button>
        <span class="vote-score" data-comment-id="${comment.id}">${comment.score}</span>
        <button class="vote-btn vote-down${downActive}" data-vote="-1" data-comment-id="${comment.id}" aria-label="Дизлайк">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v6a1 1 0 0 0 1 1h3.293a.707.707 0 0 1 .5 1.207l-7.086 7.086a1 1 0 0 1-1.414 0l-7.086-7.086a.707.707 0 0 1 .5-1.207H8a1 1 0 0 0 1-1z"/>
          </svg>
        </button>
      </div>`
    : `<div class="comment-votes">
        <button class="vote-btn vote-disabled" aria-label="Лайк" disabled>
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 19a1 1 0 0 0 1 1h4a1 1 0 0 0 1-1v-6a1 1 0 0 1 1-1h3.293a.707.707 0 0 0 .5-1.207l-7.086-7.086a1 1 0 0 0-1.414 0l-7.086 7.086a.707.707 0 0 0 .5 1.207H8a1 1 0 0 1 1 1z"/>
          </svg>
        </button>
        <span class="vote-score">0</span>
        <button class="vote-btn vote-disabled" aria-label="Дизлайк" disabled>
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v6a1 1 0 0 0 1 1h3.293a.707.707 0 0 1 .5 1.207l-7.086 7.086a1 1 0 0 1-1.414 0l-7.086-7.086a.707.707 0 0 1 .5-1.207H8a1 1 0 0 0 1-1z"/>
          </svg>
        </button>
      </div>`;

  let actionHTML = "";
  if (isOwn && isApproved) {
    actionHTML = `<div class="comment-menu">
        <button class="comment-menu-btn" aria-label="Действия" aria-haspopup="true" aria-expanded="false">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/></svg>
        </button>
        <div class="comment-dropdown-menu">
          <button class="comment-dropdown-item comment-delete-btn danger" data-comment-id="${comment.id}">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>
            <span>Удалить</span>
          </button>
        </div>
      </div>`;
  }

  const replyHTML = isApproved
    ? `<button class="comment-reply-btn" data-comment-id="${comment.id}" data-author="${comment.user_display_name}">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 17 4 12 9 7"/><path d="M20 18v-2a4 4 0 0 0-4-4H4"/></svg>
        <span>Ответить</span>
      </button>`
    : "";

  const footerHTML =
    voteHTML || replyHTML
      ? `<div class="comment-footer">${voteHTML}${replyHTML}</div>`
      : "";

  const chapterUrl =
    options?.chapterUrl ||
    (comment.novel_id && comment.chapter_id
      ? `/${comment.novel_id}/chapter/${comment.chapter_id}`
      : "");

  let jumpHTML = "";
  if (chapterUrl) {
    jumpHTML = `<a href="${chapterUrl}#${comment.id}" class="comment-jump-link" title="Перейти к комментарию в главе" aria-label="Перейти к комментарию">
      <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
        <polyline points="15 3 21 3 21 9"/>
        <line x1="10" x2="21" y1="14" y2="3"/>
      </svg>
    </a>`;
  }

  const actionsHTML =
    jumpHTML || actionHTML
      ? `<div class="comment-header-actions">${jumpHTML}${actionHTML}</div>`
      : "";

  let answersHTML = "";
  if (comment.answers && comment.answers.length > 0) {
    answersHTML = `<div class="comment-answers">`;
    for (const answer of comment.answers) {
      answersHTML += createAnswerHTML(answer);
    }
    answersHTML += `</div>`;
  }

  return `
    <div class="comment-item${extraClass}" id="${comment.id}" data-comment-id="${comment.id}" data-chapter-id="${comment.chapter_id}" data-comment-status="${comment.status}" tabindex="0">
      <div class="comment-main-row">
        <img src="${avatarUrl}" alt="${comment.user_display_name}" class="comment-avatar" loading="lazy"/>
        <div class="comment-main">
          <div class="comment-header">
            <span class="comment-author">${comment.user_display_name} ${statusBadge}</span>
            ${actionsHTML}
          </div>
          <div class="comment-body">
            <div class="comment-content">${comment.content_html}</div>
            ${footerHTML}
          </div>
        </div>
      </div>
      ${answersHTML}
    </div>
  `;
}

function createAnswerHTML(answer: CommentAnswer): string {
  const avatarUrl = getAvatarUrl(
    answer.user_id,
    answer.user_has_custom_avatar,
    answer.user_avatar_seed,
    answer.user_avatar_updated_at,
  );

  let statusBadge = "";
  let extraClass = "";

  switch (answer.status) {
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
      statusBadge = `<span class="comment-date">${formatRelativeTime(answer.created_at)}</span>`;
  }

  if (answer.is_new) {
    extraClass += " is-new";
  }

  const isOwn = answer.user_id === profileManager.getProfileId();

  let actionHTML = "";
  if (isOwn && answer.status === "approved") {
    actionHTML = `<div class="comment-menu">
        <button class="comment-menu-btn" aria-label="Действия" aria-haspopup="true" aria-expanded="false">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/></svg>
        </button>
        <div class="comment-dropdown-menu">
          <button class="comment-dropdown-item comment-delete-btn danger" data-answer-id="${answer.id}">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>
            <span>Удалить</span>
          </button>
        </div>
      </div>`;
  }

  const newBadge = answer.is_new ? ' <span class="comment-new-badge">Новый ответ</span>' : '';

  return `
    <div class="comment-answer${extraClass}" id="${answer.id}" data-answer-id="${answer.id}" tabindex="0">
      <img src="${avatarUrl}" alt="${answer.user_display_name}" class="comment-answer-avatar" loading="lazy"/>
      <div class="comment-main">
        <div class="comment-header">
          <span class="comment-author">${answer.user_display_name} ${statusBadge}${newBadge}</span>
          ${actionHTML}
        </div>
        <div class="comment-body"><div class="comment-content">${answer.content_html}</div></div>
      </div>
    </div>
  `;
}

function renderComments(
  container: HTMLElement,
  comments: Comment[],
  append: boolean = false,
): void {
  const listEl = container.querySelector(".comments-list");
  if (!listEl) return;

  if (!append) {
    if (comments.length === 0) {
      listEl.innerHTML =
        '<div class="comments-empty">Комментариев пока нет. Будьте первым!</div>';
      return;
    }
    listEl.innerHTML = "";
  }

  let html = "";
  comments.forEach((c) => {
    html += createCommentHTML(c);
  });

  if (append) {
    const temp = document.createElement("div");
    temp.innerHTML = html;
    while (temp.firstChild) {
      listEl.appendChild(temp.firstChild);
    }
  } else {
    listEl.innerHTML = html;
  }
}

let isCommentsLoading = false;
let commentsCurrentPage = 1;
let commentsTotalPages = 1;
let commentsObserver: IntersectionObserver | null = null;

async function loadComments(
  container: HTMLElement,
  chapterId: string,
  page: number = 1,
  noCache: boolean = false,
  showLoader: boolean = true,
  commentId?: string,
): Promise<void> {
  if (isCommentsLoading) return;
  isCommentsLoading = true;

  const listEl = container.querySelector(".comments-list");
  const loader = container.querySelector<HTMLElement>("#comments-loader");

  if (page === 1 && showLoader && listEl) {
    listEl.innerHTML =
      '<div class="comments-loading">Загрузка комментариев...</div>';
  }

  if (page > 1 && loader) {
    loader.style.display = "flex";
  }

  try {
    const url = commentId
      ? `${API_URL}/chapters/${chapterId}/comments?comment_id=${commentId}`
      : `${API_URL}/chapters/${chapterId}/comments?page=${page}`;

    const res = await fetch(
      url,
      noCache
        ? { credentials: "include", cache: "no-cache" }
        : { credentials: "include" },
    );
    if (!res.ok) throw new Error("Failed to load comments");

    const data: CommentsPage = await res.json();

    commentsTotalPages = data.total_pages;
    commentsCurrentPage = data.page;

    renderComments(container, data.comments, page > 1);

    const countEl = container.querySelector(".comments-count");
    if (countEl) {
      countEl.textContent = `${data.total_count}`;
    }

    if (loader) {
      loader.style.display =
        commentsCurrentPage < commentsTotalPages ? "flex" : "none";
    }

    if (page === 1 && window.location.hash && window.location.hash !== "#comments-section") {
      try {
        const targetEl = document.querySelector(window.location.hash) as HTMLElement | null;
        if (targetEl) {
          smoothScrollToTarget(
            () => targetEl,
            () => targetEl,
            (el) => {
              highlightTargetElement(el);
            },
          );
        }
      } catch {}
    }
  } catch (err) {
    console.error("Failed to load comments", err);
    if (listEl && page === 1 && showLoader) {
      listEl.innerHTML =
        '<div class="comments-error">Не удалось загрузить комментарии</div>';
    }
  } finally {
    isCommentsLoading = false;
    if (loader && commentsCurrentPage < commentsTotalPages) {
      loader.style.display = "flex";
    }
  }
}

async function loadMoreComments(
  container: HTMLElement,
  chapterId: string,
): Promise<void> {
  if (isCommentsLoading || commentsCurrentPage >= commentsTotalPages) return;
  await loadComments(
    container,
    chapterId,
    commentsCurrentPage + 1,
    false,
    false,
  );
}

let activeScrollAnimationId: number | null = null;

function cancelActiveScrollAnimation(): void {
  if (activeScrollAnimationId !== null) {
    cancelAnimationFrame(activeScrollAnimationId);
    activeScrollAnimationId = null;
  }
}

function highlightTargetElement(el: HTMLElement): void {
  el.classList.add("comment-highlight");
  setTimeout(() => {
    el.classList.remove("comment-highlight");
  }, 2500);
}

function smoothScrollToTarget(
  getTargetElement: () => HTMLElement | null,
  fallbackTarget: () => HTMLElement | null,
  onComplete?: (targetEl: HTMLElement) => void,
): void {
  cancelActiveScrollAnimation();

  const startY = window.scrollY;
  const initialEl = getTargetElement() || fallbackTarget();
  if (!initialEl) return;

  const getTargetY = (): number => {
    const el = getTargetElement();
    if (el) {
      const rect = el.getBoundingClientRect();
      const offset = (window.innerHeight - rect.height) / 2;
      return Math.max(0, window.scrollY + rect.top - Math.max(20, offset));
    }
    const fallback = fallbackTarget();
    if (fallback) {
      return Math.max(
        0,
        window.scrollY + fallback.getBoundingClientRect().top - 20,
      );
    }
    return window.scrollY;
  };

  const initialTargetY = getTargetY();
  const distance = Math.abs(initialTargetY - startY);
  const duration = Math.min(1000, Math.max(400, distance * 0.25));
  let startTime: number | null = null;

  const onUserInterrupt = () => {
    cancelActiveScrollAnimation();
    window.removeEventListener("wheel", onUserInterrupt);
    window.removeEventListener("touchstart", onUserInterrupt);
  };
  window.addEventListener("wheel", onUserInterrupt, {
    passive: true,
    once: true,
  });
  window.addEventListener("touchstart", onUserInterrupt, {
    passive: true,
    once: true,
  });

  function step(timestamp: number): void {
    if (!startTime) startTime = timestamp;
    const elapsed = timestamp - startTime;
    const progress = Math.min(elapsed / duration, 1);

    const ease =
      progress < 0.5
        ? 4 * progress * progress * progress
        : 1 - Math.pow(-2 * progress + 2, 3) / 2;

    const currentTargetY = getTargetY();
    window.scrollTo(0, startY + (currentTargetY - startY) * ease);

    if (progress < 1) {
      activeScrollAnimationId = requestAnimationFrame(step);
    } else {
      activeScrollAnimationId = null;
      window.removeEventListener("wheel", onUserInterrupt);
      window.removeEventListener("touchstart", onUserInterrupt);
      const finalEl = getTargetElement();
      if (finalEl && onComplete) {
        onComplete(finalEl);
      }
    }
  }

  activeScrollAnimationId = requestAnimationFrame(step);
}

function startSmoothScrollToHash(): void {
  const hash = window.location.hash;
  if (!hash) return;

  if (hash === "#comments-section") {
    const section = document.getElementById("comments-section");
    if (section) {
      smoothScrollToTarget(
        () => section,
        () => section,
      );
    }
    return;
  }

  const container = document.getElementById("comments-section");
  smoothScrollToTarget(
    () => {
      try {
        return document.querySelector(hash) as HTMLElement | null;
      } catch {
        return null;
      }
    },
    () => container,
    (targetEl) => {
      highlightTargetElement(targetEl);
    },
  );
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

    const timeout = setTimeout(() => {
      resolve();
    }, 4000);

    const existing = document.querySelector(
      'script[src*="challenges.cloudflare.com/turnstile"]',
    );
    if (existing) {
      const checkLoaded = setInterval(() => {
        if ((window as any).turnstile) {
          clearInterval(checkLoaded);
          clearTimeout(timeout);
          resolve();
        }
      }, 100);
      return;
    }

    const script = document.createElement("script");
    script.src =
      "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";
    script.async = true;
    script.onerror = () => {
      clearTimeout(timeout);
      resolve();
    };
    script.onload = () => {
      const checkLoaded = setInterval(() => {
        if ((window as any).turnstile) {
          clearInterval(checkLoaded);
          clearTimeout(timeout);
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

  let turnstileContainer =
    container.querySelector<HTMLElement>("#comments-turnstile-container") ||
    document.getElementById("comments-turnstile-container");
  if (!turnstileContainer) {
    turnstileContainer = document.createElement("div");
    turnstileContainer.id = "comments-turnstile-container";
    turnstileContainer.style.display = "none";
    document.body.appendChild(turnstileContainer);
  }
  if (turnstileLoaded || !(window as any).turnstile) return;

  turnstileLoaded = true;

  try {
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
  } catch (err) {
    console.warn("Turnstile render failed:", err);
  }
}

let tokenPromise: Promise<string | null> | null = null;

async function getTurnstileToken(): Promise<string | null> {
  if (!turnstileWidgetId || !(window as any).turnstile) return null;

  if (tokenPromise) {
    return tokenPromise;
  }

  tokenPromise = new Promise((resolve) => {
    if (turnstileToken) {
      const token = turnstileToken;
      turnstileToken = null;
      try {
        (window as any).turnstile.reset(turnstileWidgetId);
      } catch {}
      resolve(token);
      return;
    }

    let resolved = false;
    let checkToken: any = null;
    let failTimeout: any = null;

    const finish = (tok: string | null) => {
      if (resolved) return;
      resolved = true;
      if (checkToken) clearInterval(checkToken);
      if (failTimeout) clearTimeout(failTimeout);
      turnstileToken = null;
      resolve(tok);
    };

    failTimeout = setTimeout(() => {
      finish(null);
    }, 4000);

    try {
      (window as any).turnstile.reset(turnstileWidgetId);
      (window as any).turnstile.execute(turnstileWidgetId);
    } catch {
      finish(null);
      return;
    }

    checkToken = setInterval(() => {
      if (turnstileToken) {
        finish(turnstileToken);
      }
    }, 100);
  });

  const result = await tokenPromise;
  tokenPromise = null;
  return result;
}

let smartCaptchaWidgetId: string | null = null;
let smartCaptchaModalResolve: ((token: string | null) => void) | null = null;

function loadSmartCaptchaScript(): Promise<void> {
  return new Promise((resolve) => {
    if ((window as any).smartCaptcha) {
      resolve();
      return;
    }

    const timeout = setTimeout(() => {
      resolve();
    }, 4000);

    const existing = document.querySelector(
      'script[src*="smartcaptcha.yandexcloud.net/captcha.js"]',
    );
    if (existing) {
      const checkLoaded = setInterval(() => {
        if ((window as any).smartCaptcha) {
          clearInterval(checkLoaded);
          clearTimeout(timeout);
          resolve();
        }
      }, 100);
      return;
    }

    const script = document.createElement("script");
    script.src =
      "https://smartcaptcha.yandexcloud.net/captcha.js?render=onload&onload=kappalibSmartCaptchaOnload";
    script.async = true;
    script.defer = true;
    (window as any).kappalibSmartCaptchaOnload = () => {
      clearTimeout(timeout);
      resolve();
    };
    script.onerror = () => {
      clearTimeout(timeout);
      resolve();
    };
    document.head.appendChild(script);
  });
}

function closeSmartCaptchaModal(): void {
  const overlay = document.getElementById("comments-captcha-modal");
  if (overlay) {
    overlay.remove();
  }
  document.body.style.overflow = "";
  if (smartCaptchaModalResolve) {
    const resolve = smartCaptchaModalResolve;
    smartCaptchaModalResolve = null;
    resolve(null);
  }
}

async function getSmartCaptchaToken(): Promise<string | null> {
  if (!SMARTCAPTCHA_SITE_KEY) {
    console.warn("SMARTCAPTCHA_SITE_KEY not set");
    return null;
  }

  await loadSmartCaptchaScript();
  if (!(window as any).smartCaptcha) {
    return null;
  }

  const existingOverlay = document.getElementById("comments-captcha-modal");
  if (existingOverlay) {
    existingOverlay.remove();
  }

  return new Promise((resolve) => {
    smartCaptchaModalResolve = resolve;

    const overlay = document.createElement("div");
    overlay.id = "comments-captcha-modal";
    overlay.className = "modal-overlay";

    const container = document.createElement("div");
    container.id = "comments-smartcaptcha-container";
    container.className = "smart-captcha";
    container.addEventListener("click", (e) => e.stopPropagation());

    overlay.appendChild(container);
    document.body.style.overflow = "hidden";

    overlay.addEventListener("click", () => {
      closeSmartCaptchaModal();
    });

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        document.removeEventListener("keydown", handleKeyDown);
        closeSmartCaptchaModal();
      }
    };
    document.addEventListener("keydown", handleKeyDown);

    document.body.appendChild(overlay);

    try {
      smartCaptchaWidgetId = (window as any).smartCaptcha.render(container, {
        sitekey: SMARTCAPTCHA_SITE_KEY,
        hl: "ru",
        callback: (token: string) => {
          document.removeEventListener("keydown", handleKeyDown);
          const currentResolve = smartCaptchaModalResolve;
          smartCaptchaModalResolve = null;
          document.body.style.overflow = "";
          overlay.remove();
          if (currentResolve) {
            currentResolve(token);
          }
        },
      });
    } catch (err) {
      console.error("Failed to render smartcaptcha", err);
      closeSmartCaptchaModal();
    }
  });
}

function updateCharCounter(textarea: HTMLTextAreaElement): void {
  const form = textarea.closest(".comment-form");
  const counter = form?.querySelector(".comment-char-counter");
  if (counter) {
    const isReply = textarea.classList.contains("comment-reply-textarea");
    const maxLen = isReply ? 500 : 3000;
    const warnThreshold = isReply ? 400 : 2500;
    const len = textarea.value.length;
    counter.textContent = `${len}/${maxLen}`;
    counter.classList.toggle("count-warning", len > warnThreshold);
    counter.classList.toggle("count-error", len >= maxLen);
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

function getTargetCommentIdFromHash(): string | undefined {
  const hash = window.location.hash;
  if (!hash) return undefined;
  const match = hash.match(/^#((?:cmt|ans)_[a-z0-9]{8})$/);
  return match ? match[1] : undefined;
}

export function initComments(): void {
  const container = document.getElementById("comments-section");
  if (!container) return;

  const chapterId = container.dataset.chapterId;
  if (!chapterId) return;

  renderCommentForm(container);

  if (initializedContainers.has(container)) return;
  initializedContainers.add(container);

  const loader = container.querySelector<HTMLElement>("#comments-loader");
  if (loader) {
    commentsObserver?.disconnect();
    commentsObserver = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (
            entry.isIntersecting &&
            !isCommentsLoading &&
            commentsCurrentPage < commentsTotalPages
          ) {
            loadMoreComments(container, chapterId);
          }
        });
      },
      { rootMargin: "200px" },
    );
    commentsObserver.observe(loader);
  }

  window.addEventListener("hashchange", () => {
    const targetId = getTargetCommentIdFromHash();
    if (targetId && !document.getElementById(targetId)) {
      startSmoothScrollToHash();
      loadComments(container, chapterId, 1, false, true, targetId);
    } else {
      startSmoothScrollToHash();
    }
  });

  const initialCommentId = getTargetCommentIdFromHash();
  if (initialCommentId || window.location.hash === "#comments-section") {
    startSmoothScrollToHash();
  }
  loadComments(container, chapterId, 1, false, true, initialCommentId);

  profileManager.onLogin(() => {
    renderCommentForm(container);
  });

  container.addEventListener("click", (e) => {
    const target = e.target as HTMLElement;

    const spoiler = target.closest(".spoiler") as HTMLElement | null;
    if (spoiler) {
      spoiler.classList.toggle("revealed");
      return;
    }

    const voteBtn = target.closest(".vote-btn") as HTMLElement | null;
    if (voteBtn) {
      e.preventDefault();
      handleVote(voteBtn);
      return;
    }

    const replyBtn = target.closest(".comment-reply-btn") as HTMLElement | null;
    if (replyBtn) {
      e.preventDefault();
      handleReply(replyBtn, container);
      return;
    }

    const menuBtn = target.closest(".comment-menu-btn") as HTMLElement | null;
    if (menuBtn) {
      e.preventDefault();
      const menu = menuBtn.closest(".comment-menu");
      const wasActive = menu?.classList.contains("active");
      document.querySelectorAll(".comment-menu.active").forEach((m) => {
        m.classList.remove("active");
        m.querySelector(".comment-menu-btn")?.setAttribute("aria-expanded", "false");
      });
      if (!wasActive && menu) {
        menu.classList.add("active");
        menuBtn.setAttribute("aria-expanded", "true");
      }
      return;
    }

    const deleteBtn = target.closest(".comment-delete-btn") as HTMLElement | null;
    if (deleteBtn) {
      e.preventDefault();
      deleteBtn.closest(".comment-menu")?.classList.remove("active");
      const answerId = deleteBtn.dataset.answerId;
      if (answerId) {
        handleDeleteAnswer(answerId, container, chapterId);
      } else {
        handleDeleteComment(deleteBtn, container, chapterId);
      }
      return;
    }
  });
}

async function handleVote(
  btn: HTMLElement,
): Promise<void> {
  if (!profileManager.isLoggedIn()) return;

  const commentId = btn.dataset.commentId;
  if (!commentId) return;

  const requestedValue = parseInt(btn.dataset.vote || "0", 10);
  const commentItem = btn.closest(".comment-item") as HTMLElement;
  if (!commentItem) return;

  const upBtn = commentItem.querySelector('.vote-btn[data-vote="1"]');
  const downBtn = commentItem.querySelector('.vote-btn[data-vote="-1"]');
  const scoreEl = commentItem.querySelector(".vote-score");
  if (!upBtn || !downBtn || !scoreEl) return;

  const wasActive = btn.classList.contains("vote-active");
  const value = wasActive ? 0 : requestedValue;

  try {
    const res = await fetch(`${API_URL}/comments/${commentId}/vote`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ value }),
    });
    if (!res.ok) return;

    const data: { score: number; user_vote: number } = await res.json();
    scoreEl.textContent = String(data.score);
    upBtn.classList.toggle("vote-active", data.user_vote === 1);
    downBtn.classList.toggle("vote-active", data.user_vote === -1);
  } catch {}
}

function renderToolbarHTML(imageInputAttr: string): string {
  return `
    <div class="comment-toolbar">
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
      <input type="file" ${imageInputAttr} accept="image/jpeg,image/png,image/gif" style="display:none"/>
    </div>
  `;
}

function initToolbarHandlers(
  toolbar: HTMLElement,
  textarea: HTMLTextAreaElement,
  imageInput: HTMLInputElement | null,
  container: HTMLElement,
): void {
  imageInput?.addEventListener("change", async () => {
    const file = imageInput.files?.[0];
    if (!file || !textarea) return;
    imageInput.value = "";
    initTurnstileForComments(container);
    await uploadCommentImage(file, textarea);
  });

  toolbar.querySelectorAll(".toolbar-btn").forEach((btn) => {
    btn.addEventListener("mousedown", (e) => {
      e.preventDefault();
      const action = (btn as HTMLElement).dataset.action;
      switch (action) {
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
}

async function postCommentPayload(
  url: string,
  content: string,
  submitBtn: HTMLButtonElement,
): Promise<boolean> {
  submitBtn.disabled = true;
  submitBtn.textContent = "Проверка...";

  let turnstileTok: string | null = await getTurnstileToken();
  let smartCaptchaTok: string | null = null;

  if (!turnstileTok) {
    submitBtn.textContent = "Проверка...";
    smartCaptchaTok = await getSmartCaptchaToken();
  }

  if (!turnstileTok && !smartCaptchaTok) {
    submitBtn.disabled = false;
    submitBtn.textContent = "Отправить";
    return false;
  }

  submitBtn.textContent = "Отправка...";

  const payload: {
    content: string;
    turnstile_token?: string;
    smart_captcha_token?: string;
  } = {
    content: content,
  };

  if (smartCaptchaTok) {
    payload.smart_captcha_token = smartCaptchaTok;
  } else if (turnstileTok) {
    payload.turnstile_token = turnstileTok;
  }

  let res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    const err = await res.json();
    if (
      res.status === 400 &&
      err.detail === "Captcha verification failed" &&
      turnstileTok &&
      !smartCaptchaTok
    ) {
      submitBtn.textContent = "Проверка...";
      smartCaptchaTok = await getSmartCaptchaToken();
      if (smartCaptchaTok) {
        submitBtn.textContent = "Отправка...";
        payload.turnstile_token = undefined;
        payload.smart_captcha_token = smartCaptchaTok;
        res = await fetch(url, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify(payload),
        });
        if (!res.ok) {
          const retryErr = await res.json();
          throw new Error(retryErr.detail || "Failed to submit");
        }
      } else {
        submitBtn.disabled = false;
        submitBtn.textContent = "Отправить";
        return false;
      }
    } else {
      throw new Error(err.detail || "Failed to submit");
    }
  }

  return true;
}

function handleReply(btn: HTMLElement, container: HTMLElement): void {
  if (!profileManager.isLoggedIn()) {
    alert("Войдите в аккаунт, чтобы ответить");
    return;
  }

  const commentId = btn.dataset.commentId;
  const author = btn.dataset.author || "";
  if (!commentId) return;

  const commentItem = btn.closest(".comment-item") as HTMLElement | null;
  if (!commentItem) return;

  const existingForm = commentItem.querySelector(".comment-reply-form-wrapper");
  if (existingForm) {
    existingForm.remove();
    return;
  }

  document.querySelectorAll(".comment-reply-form-wrapper").forEach((el) => el.remove());

  const formWrapper = document.createElement("div");
  formWrapper.className = "comment-reply-form-wrapper";
  formWrapper.innerHTML = `
    <div class="comment-form comment-reply-form">
      ${renderToolbarHTML('class="comment-reply-image-input"')}
      <textarea
        class="comment-textarea comment-reply-textarea"
        placeholder="Ответ пользователю ${author}..."
        maxlength="500"
        rows="2"
      ></textarea>
      <div class="comment-form-footer">
        <span class="comment-char-counter">0/500</span>
        <div class="comment-reply-actions">
          <button type="button" class="comment-reply-cancel-btn">Отмена</button>
          <button type="button" class="action-btn btn-primary comment-submit-btn comment-reply-submit-btn">Отправить</button>
        </div>
      </div>
    </div>
  `;

  const answersEl = commentItem.querySelector(".comment-answers");
  if (answersEl) {
    commentItem.insertBefore(formWrapper, answersEl);
  } else {
    commentItem.appendChild(formWrapper);
  }

  initReplyFormHandlers(formWrapper, commentId, container);

  const textarea = formWrapper.querySelector(".comment-reply-textarea") as HTMLTextAreaElement | null;
  textarea?.focus();
}

function initReplyFormHandlers(
  formWrapper: HTMLElement,
  commentId: string,
  container: HTMLElement,
): void {
  const textarea = formWrapper.querySelector(".comment-reply-textarea") as HTMLTextAreaElement | null;
  const submitBtn = formWrapper.querySelector(".comment-reply-submit-btn") as HTMLButtonElement | null;
  const cancelBtn = formWrapper.querySelector(".comment-reply-cancel-btn") as HTMLButtonElement | null;
  const toolbar = formWrapper.querySelector(".comment-toolbar") as HTMLElement | null;
  const imageInput = formWrapper.querySelector(".comment-reply-image-input") as HTMLInputElement | null;

  if (!textarea || !submitBtn) return;

  if (getRemainingCooldown() > 0) {
    startCooldownTimer(submitBtn);
  }

  if (toolbar) {
    initToolbarHandlers(toolbar, textarea, imageInput, container);
  }

  cancelBtn?.addEventListener("click", () => {
    formWrapper.remove();
  });

  textarea.addEventListener("input", () => {
    updateCharCounter(textarea);
    autoResizeTextarea(textarea);
  });

  textarea.addEventListener("focus", () => {
    initTurnstileForComments(container);
  });

  submitBtn.addEventListener("click", async () => {
    if (getRemainingCooldown() > 0) {
      startCooldownTimer(submitBtn);
      return;
    }

    const content = textarea.value.trim();
    if (!content) return;

    if (content.length > 500) {
      alert("Ответ слишком длинный (максимум 500 символов)");
      return;
    }

    if (!profileManager.isLoggedIn()) {
      alert("Войдите в аккаунт, чтобы оставить ответ");
      return;
    }

    try {
      const ok = await postCommentPayload(
        `${API_URL}/comments/${commentId}/answers`,
        content,
        submitBtn,
      );
      if (!ok) return;

      setLastCommentTime();
      startAllCooldownTimers();
      formWrapper.remove();

      if (turnstileWidgetId && (window as any).turnstile) {
        (window as any).turnstile.reset(turnstileWidgetId);
      }

      const chapterId = container.dataset.chapterId;
      if (chapterId) {
        await loadComments(container, chapterId, 1, true, false, commentId);
      } else {
        await loadMyComments(1);
      }
    } catch (err: any) {
      console.error("Failed to submit answer", err);
      const msg =
        err && typeof err === "object" && "message" in err && err.message
          ? err.message === "Failed to submit"
            ? "Не удалось отправить. Попробуйте ещё раз."
            : err.message
          : "Не удалось отправить. Попробуйте ещё раз.";
      alert(msg);
      submitBtn.disabled = false;
      submitBtn.textContent = "Отправить";
    }
  });
}

async function handleDeleteComment(
  btn: HTMLElement,
  container?: HTMLElement,
  chapterId?: string,
): Promise<void> {
  const commentId = btn.dataset.commentId;
  if (!commentId) return;

  if (!confirm("Вы уверены, что хотите удалить этот комментарий?")) return;

  try {
    const res = await fetch(`${API_URL}/comments/${commentId}`, {
      method: "DELETE",
      credentials: "include",
    });

    if (!res.ok) return;

    const wrapper = btn.closest(".mc-comment-wrapper") as HTMLElement | null;
    if (wrapper) {
      const separator = wrapper.querySelector(".mc-chapter-separator");
      const nextWrapper = wrapper.nextElementSibling as HTMLElement | null;
      if (
        separator &&
        nextWrapper &&
        nextWrapper.classList.contains("mc-comment-wrapper")
      ) {
        const nextSeparator = nextWrapper.querySelector(".mc-chapter-separator");
        if (!nextSeparator) {
          nextWrapper.insertBefore(separator, nextWrapper.firstChild);
        }
      }
      wrapper.remove();
    } else {
      const item = btn.closest(".comment-item");
      if (item) item.remove();
    }

    if (container && chapterId) {
      const countEl = container.querySelector(".comments-count");
      if (countEl) {
        const current = parseInt(countEl.textContent || "0", 10);
        if (current > 0) countEl.textContent = String(current - 1);
      }
      const listEl = container.querySelector(".comments-list");
      if (listEl && listEl.querySelectorAll(".comment-item").length === 0) {
        listEl.innerHTML =
          '<div class="comments-empty">Комментариев пока нет. Будьте первым!</div>';
      }
    } else {
      const countEl = document.getElementById("mc-count");
      if (countEl) {
        const current = parseInt(countEl.textContent || "0", 10);
        if (current > 0) countEl.textContent = String(current - 1);
      }
      const listEl = document.getElementById("mc-list");
      const emptyEl = document.getElementById("mc-empty");
      if (listEl && listEl.querySelectorAll(".comment-item").length === 0 && emptyEl) {
        emptyEl.style.display = "block";
      }
    }
  } catch {}
}

async function handleDeleteAnswer(
  answerId: string,
  container: HTMLElement,
  chapterId?: string,
): Promise<void> {
  if (!confirm("Вы уверены, что хотите удалить этот ответ?")) return;

  try {
    const res = await fetch(`${API_URL}/comment-answers/${answerId}`, {
      method: "DELETE",
      credentials: "include",
    });

    if (!res.ok) return;

    const answerEl = container.querySelector(
      `.comment-answer[data-answer-id="${answerId}"]`,
    );
    if (answerEl) answerEl.remove();
  } catch {}
}

async function handleDeleteMyAnswer(
  answerId: string,
): Promise<void> {
  if (!confirm("Вы уверены, что хотите удалить этот ответ?")) return;

  try {
    const res = await fetch(`${API_URL}/comment-answers/${answerId}`, {
      method: "DELETE",
      credentials: "include",
    });

    if (!res.ok) return;

    const answerEl = document.querySelector(
      `.comment-answer[data-answer-id="${answerId}"]`,
    );
    if (answerEl) answerEl.remove();
  } catch {}
}

function renderCommentForm(container: HTMLElement): void {
  const formWrapper = container.querySelector(".comment-form-wrapper");
  if (!formWrapper) return;

  if (profileManager.isLoggedIn()) {
    formWrapper.innerHTML = `
      <div class="comment-form">
        ${renderToolbarHTML('id="comment-image-input"')}
        <textarea
          id="comment-textarea"
          class="comment-textarea"
          placeholder="Ваш комментарий..."
          maxlength="3000"
          rows="2"
        ></textarea>
        <div class="comment-form-footer">
          <span id="comment-char-counter" class="comment-char-counter">0/3000</span>
          <a href="/markdown" target="_blank" class="comment-markdown-hint">Формат</a>
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

function insertLinePrefix(
  textarea: HTMLTextAreaElement,
  prefix: string,
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
  } else {
    const before = textarea.value.substring(0, lineStart);
    const after = textarea.value.substring(lineStart);
    textarea.value = `${before}${prefix}${after}`;
    textarea.selectionStart = textarea.selectionEnd = start + prefix.length;
  }

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

interface UserCommentsPage {
  comments: Comment[];
  page: number;
  page_size: number;
  total_count: number;
  total_pages: number;
}

function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength) + "…";
}

function renderMyComments(
  listEl: HTMLElement,
  comments: Comment[],
  append: boolean = false,
): void {
  if (!append && comments.length === 0) {
    listEl.innerHTML = "";
    return;
  }

  let lastChapterId: string | null = null;
  if (append) {
    const lastItem = listEl.querySelector<HTMLElement>(
      ".mc-comment-wrapper:last-child .comment-item",
    );
    lastChapterId = lastItem?.dataset.chapterId || null;
  }

  let html = "";
  comments.forEach((c) => {
    const showSeparator = c.chapter_id !== lastChapterId;
    lastChapterId = c.chapter_id;

    const novelTitle = truncateText(c.novel_title || "", 25);
    const chapterUrl = `/${c.novel_id || ""}/chapter/${c.chapter_id}`;

    if (showSeparator) {
      html += `<div class="mc-comment-wrapper">
        <div class="mc-chapter-separator">
          <a href="${chapterUrl}" class="mc-chapter-link">
            <span class="mc-novel-title">${novelTitle}</span>
            <span class="mc-chapter-num">Глава ${c.chapter_num || ""}</span>
          </a>
        </div>
        ${createCommentHTML(c, { chapterUrl })}
      </div>`;
    } else {
      html += `<div class="mc-comment-wrapper">
        ${createCommentHTML(c, { chapterUrl })}
      </div>`;
    }
  });

  if (append) {
    const temp = document.createElement("div");
    temp.innerHTML = html;
    while (temp.firstChild) {
      listEl.appendChild(temp.firstChild);
    }
  } else {
    listEl.innerHTML = html;
  }
}

function buildSparklinePath(
  values: number[],
  width: number,
  height: number,
): string {
  if (values.length === 0) return "";

  const all = [...values];
  const min = Math.min(...all);
  const max = Math.max(...all);

  if (min === max) {
    const y = height / 2;
    return `M0,${y} L${width},${y}`;
  }

  const padding = 2;
  const range = max - min;
  const stepX = values.length > 1 ? width / (values.length - 1) : width;

  const points = values.map((v, i) => {
    const x = values.length > 1 ? i * stepX : width / 2;
    const y = height - padding - ((v - min) / range) * (height - padding * 2);
    return `${x},${y}`;
  });

  return `M${points.join(" L")}`;
}

function buildSparklineAreaPath(
  values: number[],
  width: number,
  height: number,
): string {
  if (values.length === 0) return "";

  const line = buildSparklinePath(values, width, height);
  return `${line} L${width},${height} L0,${height} Z`;
}

function renderStatCard(
  label: string,
  value: number,
  values: number[],
  color: string,
  prefix: string = "",
  id: string = "0",
): string {
  const width = 240;
  const height = 40;
  const linePath = buildSparklinePath(values, width, height);
  const areaPath = buildSparklineAreaPath(values, width, height);

  return `<div class="mc-stat-card">
    <div class="mc-stat-header">
      <span class="mc-stat-label">${label}</span>
      <span class="mc-stat-value" style="color: ${color}">${prefix}${value}</span>
    </div>
    <svg class="mc-stat-sparkline" viewBox="0 0 ${width} ${height}" preserveAspectRatio="none">
      <defs>
        <linearGradient id="grad-${id}" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="${color}" stop-opacity="0.25"/>
          <stop offset="100%" stop-color="${color}" stop-opacity="0"/>
        </linearGradient>
      </defs>
      <path d="${areaPath}" fill="url(#grad-${id})"/>
      <path d="${linePath}" fill="none" stroke="${color}" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
    </svg>
  </div>`;
}

async function loadCommentStats(): Promise<void> {
  const container = document.getElementById("mc-stats");
  if (!container) return;

  try {
    const res = await fetch(`${API_URL}/profile/me/comment-stats`, {
      credentials: "include",
    });

    if (!res.ok || res.status === 401) {
      container.style.display = "none";
      return;
    }

    const data: CommentStats = await res.json();

    const ratingValues = data.days.map((d) => d.rating);
    const repliesValues = data.days.map((d) => d.replies);

    const ratingPrefix = data.rating > 0 ? "+" : "";
    const ratingColor =
      data.rating === 0
        ? "var(--secondary)"
        : data.rating > 0
          ? "var(--accent-primary)"
          : "var(--color-danger)";

    container.innerHTML =
      renderStatCard(
        "Рейтинг",
        data.rating,
        ratingValues,
        ratingColor,
        ratingPrefix,
        "rating",
      ) +
      renderStatCard(
        "Ответы",
        data.replies,
        repliesValues,
        "var(--secondary)",
        "",
        "replies",
      );

    container.style.display = "flex";
  } catch {
    container.style.display = "none";
  }
}

let isMyCommentsLoading = false;
let myCommentsCurrentPage = 1;
let myCommentsTotalPages = 1;
let myCommentsObserver: IntersectionObserver | null = null;

async function loadMyComments(page: number = 1): Promise<void> {
  if (isMyCommentsLoading) return;
  isMyCommentsLoading = true;

  if (!profileManager.getProfileCache()) {
    await profileManager.fetchProfile();
  }

  const listEl = document.getElementById("mc-list");
  const emptyEl = document.getElementById("mc-empty");
  const loadingEl = document.getElementById("mc-loading");
  const countEl = document.getElementById("mc-count");
  const loader = document.getElementById("mc-loader");

  if (!listEl || !emptyEl) {
    isMyCommentsLoading = false;
    return;
  }

  if (page === 1 && loadingEl) {
    loadingEl.style.display = "block";
  }
  if (page > 1 && loader) {
    loader.style.display = "flex";
  }

  try {
    const res = await fetch(`${API_URL}/profile/me/comments?page=${page}`, {
      credentials: "include",
    });

    if (res.status === 401) {
      if (loadingEl) loadingEl.style.display = "none";
      emptyEl.style.display = "block";
      emptyEl.querySelector("p")!.textContent =
        "Войдите в аккаунт, чтобы видеть свои комментарии";
      return;
    }

    if (!res.ok) throw new Error("Failed to load comments");

    const data: UserCommentsPage = await res.json();

    profileManager.markNotificationsAsRead();
    updateProfileBadges();

    if (loadingEl) loadingEl.style.display = "none";

    myCommentsTotalPages = data.total_pages;
    myCommentsCurrentPage = data.page;

    if (data.total_count === 0 && page === 1) {
      emptyEl.style.display = "block";
      listEl.innerHTML = "";
      if (countEl) countEl.textContent = "";
      if (loader) loader.style.display = "none";
      return;
    }

    emptyEl.style.display = "none";
    if (countEl) countEl.textContent = `${data.total_count}`;

    renderMyComments(listEl, data.comments, page > 1);

    if (loader) {
      loader.style.display =
        myCommentsCurrentPage < myCommentsTotalPages ? "flex" : "none";
    }
  } catch (err) {
    console.error("Failed to load user comments", err);
    if (loadingEl) loadingEl.style.display = "none";
    if (page === 1) {
      listEl.innerHTML =
        '<div class="comments-error">Не удалось загрузить комментарии</div>';
    }
  } finally {
    isMyCommentsLoading = false;
    if (loader && myCommentsCurrentPage < myCommentsTotalPages) {
      loader.style.display = "flex";
    }
  }
}

async function loadMoreMyComments(): Promise<void> {
  if (isMyCommentsLoading || myCommentsCurrentPage >= myCommentsTotalPages) return;
  await loadMyComments(myCommentsCurrentPage + 1);
}

export function initMyCommentsPage(): void {
  const content = document.getElementById("mc-content");
  if (!content) return;

  if (initializedContainers.has(content)) return;
  initializedContainers.add(content);

  profileManager.onLogin(() => {
    loadCommentStats();
    loadMyComments(1);
  });

  if (!profileManager.isLoggedIn()) {
    const emptyEl = document.getElementById("mc-empty");
    if (emptyEl) {
      emptyEl.style.display = "block";
      emptyEl.querySelector("p")!.textContent =
        "Войдите в аккаунт, чтобы видеть свои комментарии";
    }
    return;
  }

  const loader = document.getElementById("mc-loader");
  if (loader) {
    myCommentsObserver?.disconnect();
    myCommentsObserver = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (
            entry.isIntersecting &&
            !isMyCommentsLoading &&
            myCommentsCurrentPage < myCommentsTotalPages
          ) {
            loadMoreMyComments();
          }
        });
      },
      { rootMargin: "200px" },
    );
    myCommentsObserver.observe(loader);
  }

  loadCommentStats();
  loadMyComments(1);

  content.addEventListener("click", (e) => {
    const target = e.target as HTMLElement;

    const spoiler = target.closest(".spoiler") as HTMLElement | null;
    if (spoiler) {
      spoiler.classList.toggle("revealed");
      return;
    }

    const voteBtn = target.closest(".vote-btn") as HTMLElement | null;
    if (voteBtn) {
      e.preventDefault();
      handleVote(voteBtn);
      return;
    }

    const replyBtn = target.closest(".comment-reply-btn") as HTMLElement | null;
    if (replyBtn) {
      e.preventDefault();
      handleReply(replyBtn, content);
      return;
    }

    const menuBtn = target.closest(".comment-menu-btn") as HTMLElement | null;
    if (menuBtn) {
      e.preventDefault();
      const menu = menuBtn.closest(".comment-menu");
      const wasActive = menu?.classList.contains("active");
      document.querySelectorAll(".comment-menu.active").forEach((m) => {
        m.classList.remove("active");
        m.querySelector(".comment-menu-btn")?.setAttribute("aria-expanded", "false");
      });
      if (!wasActive && menu) {
        menu.classList.add("active");
        menuBtn.setAttribute("aria-expanded", "true");
      }
      return;
    }

    const deleteBtn = target.closest(
      ".comment-delete-btn",
    ) as HTMLElement | null;
    if (deleteBtn) {
      e.preventDefault();
      deleteBtn.closest(".comment-menu")?.classList.remove("active");

      const answerId = deleteBtn.dataset.answerId;
      if (answerId) {
        handleDeleteMyAnswer(answerId);
      } else {
        handleDeleteComment(deleteBtn);
      }
      return;
    }
  });
}

function initFormHandlers(container: HTMLElement): void {
  const textarea = container.querySelector(
    "#comment-textarea",
  ) as HTMLTextAreaElement;
  const submitBtn = container.querySelector(
    "#comment-submit",
  ) as HTMLButtonElement;
  const chapterId = container.dataset.chapterId;
  const toolbar = container.querySelector(".comment-toolbar") as HTMLElement | null;
  const imageInput = container.querySelector(
    "#comment-image-input",
  ) as HTMLInputElement;

  if (!chapterId) return;

  if (submitBtn && getRemainingCooldown() > 0) {
    startCooldownTimer(submitBtn);
  }

  if (toolbar && textarea) {
    initToolbarHandlers(toolbar, textarea, imageInput, container);
  }

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
      if (getRemainingCooldown() > 0) {
        startCooldownTimer(submitBtn);
        return;
      }

      const content = textarea.value.trim();
      if (!content) return;

      if (content.length > 3000) {
        alert("Комментарий слишком длинный (максимум 3000 символов)");
        return;
      }

      if (!profileManager.isLoggedIn()) {
        alert("Войдите в аккаунт, чтобы оставить комментарий");
        return;
      }

      try {
        const ok = await postCommentPayload(
          `${API_URL}/chapters/${chapterId}/comments`,
          content,
          submitBtn,
        );
        if (!ok) return;

        setLastCommentTime();
        startAllCooldownTimers();

        textarea.value = "";
        updateCharCounter(textarea);
        autoResizeTextarea(textarea);

        if (turnstileWidgetId && (window as any).turnstile) {
          (window as any).turnstile.reset(turnstileWidgetId);
        }

        await loadComments(container, chapterId, 1, true, false);
      } catch (err: any) {
        console.error("Failed to submit", err);
        const msg =
          err && typeof err === "object" && "message" in err && err.message
            ? err.message === "Failed to submit"
              ? "Не удалось отправить. Попробуйте ещё раз."
              : err.message
            : "Не удалось отправить. Попробуйте ещё раз.";
        alert(msg);
        submitBtn.disabled = false;
        submitBtn.textContent = "Отправить";
      }
    });
  }
}
