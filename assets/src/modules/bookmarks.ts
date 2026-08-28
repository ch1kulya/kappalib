import { profileManager } from "./profile";
import { uiManager } from "./ui";
import Dropdown from "./dropdown";
import { formatRelativeTime } from "./comments";

const API_URL = process.env.API_URL;

interface Bookmark {
  id: string;
  novelId: string;
  chapterId: string;
  chapterNum: number;
  novelTitle: string;
  novelCoverUrl: string;
  name: string;
  createdAt: number;
}

interface BookmarkCategory {
  name: string;
  bookmarks: Bookmark[];
}

let cachedCategories: BookmarkCategory[] | null = null;

async function fetchBookmarkCategories(
  force = false,
): Promise<BookmarkCategory[]> {
  if (cachedCategories && !force) return cachedCategories;
  try {
    const res = await fetch(`${API_URL}/profile/me/bookmarks`, {
      credentials: "include",
    });
    if (!res.ok) return [];
    const data: { categories: BookmarkCategory[] } = await res.json();
    cachedCategories = data.categories;
    return cachedCategories;
  } catch {
    return [];
  }
}

function findBookmarkForChapter(
  categories: BookmarkCategory[],
  chapterId: string,
): Bookmark | null {
  for (const cat of categories) {
    const found = cat.bookmarks.find((b) => b.chapterId === chapterId);
    if (found) return found;
  }
  return null;
}

function findCategoryOfBookmark(
  categories: BookmarkCategory[],
  bookmarkId: string,
): string {
  for (const cat of categories) {
    if (cat.bookmarks.some((b) => b.id === bookmarkId)) return cat.name;
  }
  return "";
}

interface ChapterCtx {
  novelId: string;
  chapterId: string;
  chapterNum: number;
  novelTitle: string;
  novelCover: string;
}

let currentBookmark: Bookmark | null = null;
let currentCtx: ChapterCtx | null = null;
let selectedCategory = "";

function cloneTemplate(id: string): DocumentFragment {
  const template = document.getElementById(id) as HTMLTemplateElement;
  return template.content.cloneNode(true) as DocumentFragment;
}

function getFinalCategoryValue(): string {
  const newInput = document.getElementById(
    "bm-category-new-input",
  ) as HTMLInputElement | null;
  if (newInput && newInput.style.display !== "none") {
    return newInput.value.trim();
  }
  return selectedCategory;
}

function populateCategoryDropdown(
  categories: BookmarkCategory[],
  preselected: string,
): void {
  const dropdownEl = document.getElementById(
    "bm-category-dropdown",
  ) as HTMLElement | null;
  const container = document.getElementById("bm-category-options");
  const labelEl = document.getElementById(
    "bm-category-label",
  ) as HTMLElement | null;
  const newInput = document.getElementById(
    "bm-category-new-input",
  ) as HTMLInputElement | null;
  if (!dropdownEl || !container || !labelEl || !newInput) return;

  container.innerHTML = "";
  newInput.style.display = "none";
  newInput.value = "";
  selectedCategory = "";

  const names = categories.map((c) => c.name);
  const initial = names.includes(preselected) ? preselected : names[0] || "";

  names.forEach((name) => {
    const node = cloneTemplate("tpl-bookmark-category-option");
    const btn = node.querySelector(".dropdown-item") as HTMLElement;
    const label = node.querySelector('[data-field="label"]') as HTMLElement;
    btn.dataset.value = name;
    label.textContent = name;
    const isSelected = name === initial;
    btn.classList.toggle("selected", isSelected);
    btn.setAttribute("aria-selected", String(isSelected));
    container.appendChild(node);
  });

  container.appendChild(cloneTemplate("tpl-bookmark-category-new"));

  if (initial) {
    selectedCategory = initial;
    labelEl.textContent = initial;
  } else {
    labelEl.textContent = "Новая категория";
    newInput.style.display = "block";
    newInput.placeholder = "Избранное";
  }

  const dropdown = new Dropdown(dropdownEl);

  dropdownEl.addEventListener("change", (e: Event) => {
    const value = (e as CustomEvent).detail.value as string;
    if (value === "__new__") {
      selectedCategory = "";
      labelEl.textContent = "Новая категория";
      newInput.style.display = "block";
      newInput.value = "";
      newInput.placeholder = "Название категории";
      newInput.focus();
    } else {
      selectedCategory = value;
      labelEl.textContent = value;
      newInput.style.display = "none";
      newInput.value = "";
      dropdown.close();
    }
  });
}

function renderBookmarkForm(): void {
  const card = document.getElementById("bookmark-card");
  const template = document.getElementById(
    "tpl-bookmark-form",
  ) as HTMLTemplateElement | null;
  if (!card || !template || !currentCtx) return;

  card.innerHTML = "";
  const node = template.content.cloneNode(true) as DocumentFragment;
  card.appendChild(node);

  const titleEl = card.querySelector("#bm-form-title") as HTMLElement;
  const nameInput = card.querySelector("#bm-form-name") as HTMLInputElement;
  const saveBtn = card.querySelector("#bm-form-save") as HTMLButtonElement;
  const deleteBtn = card.querySelector("#bm-form-delete") as HTMLButtonElement;

  const defaultName = `Глава ${currentCtx.chapterNum} — ${currentCtx.novelTitle}`;

  if (currentBookmark) {
    nameInput.value = currentBookmark.name;
    deleteBtn.style.display = "inline-block";
  } else {
    nameInput.placeholder = defaultName;
  }

  fetchBookmarkCategories().then((categories) => {
    const preselected = currentBookmark
      ? findCategoryOfBookmark(categories, currentBookmark.id)
      : "";
    populateCategoryDropdown(categories, preselected);
  });

  saveBtn.addEventListener("click", async () => {
    const nameEl = card.querySelector("#bm-form-name") as HTMLInputElement;
    const category = getFinalCategoryValue();
    const name = nameEl.value;

    if (currentBookmark) {
      await fetch(`${API_URL}/bookmarks/${currentBookmark.id}`, {
        method: "PATCH",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, category }),
      });
    } else {
      await fetch(`${API_URL}/bookmarks`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          novelId: currentCtx!.novelId,
          chapterId: currentCtx!.chapterId,
          chapterNum: currentCtx!.chapterNum,
          novelTitle: currentCtx!.novelTitle,
          novelCoverUrl: currentCtx!.novelCover,
          category,
          name,
        }),
      });
    }
    uiManager.closeAll();
    await refreshButtonState();
  });

  deleteBtn.addEventListener("click", async () => {
    if (!currentBookmark) return;
    await fetch(`${API_URL}/bookmarks/${currentBookmark.id}`, {
      method: "DELETE",
      credentials: "include",
    });
    uiManager.closeAll();
    await refreshButtonState();
  });
}

async function refreshButtonState(): Promise<void> {
  const btn = document.getElementById("header-bookmark-btn");
  if (!btn || !currentCtx || !profileManager.isLoggedIn()) return;
  const categories = await fetchBookmarkCategories(true);
  currentBookmark = findBookmarkForChapter(categories, currentCtx.chapterId);
  btn.classList.toggle("bm-active", !!currentBookmark);
}

export function initBookmarkButton(): void {
  const btn = document.getElementById(
    "header-bookmark-btn",
  ) as HTMLButtonElement | null;
  const tracker = document.getElementById("reading-tracker");
  if (!btn || !tracker) return;

  currentCtx = {
    novelId: tracker.dataset.novelId!,
    chapterId: tracker.dataset.chapterId!,
    chapterNum: Number(tracker.dataset.chapterNum),
    novelTitle: tracker.dataset.novelTitle!,
    novelCover: tracker.dataset.novelCover || "",
  };

  refreshButtonState();
  profileManager.onLogin(refreshButtonState);

  btn.addEventListener("click", (e) => {
    e.preventDefault();
    e.stopPropagation();

    if (!profileManager.isLoggedIn()) {
      alert("Войдите в аккаунт, чтобы добавить закладку");
      return;
    }

    uiManager.toggleBookmark();
    renderBookmarkForm();
  });
}

export function initBookmarksPage(): void {
  const container = document.getElementById("bm-categories");
  if (!container) return;

  const empty = document.getElementById("bm-empty");
  const loading = document.getElementById("bm-loading");

  if (!profileManager.isLoggedIn()) {
    if (loading) loading.style.display = "none";
    if (empty) {
      empty.style.display = "block";
      empty.querySelector("p")!.textContent =
        "Войдите в аккаунт, чтобы видеть свои закладки";
    }
    return;
  }

  loadBookmarksPage(container, empty, loading);
}

function toggleMenu(container: HTMLElement, menu: HTMLElement): void {
  const btn = menu.querySelector(".comment-menu-btn") as HTMLElement;
  const isActive = menu.classList.contains("active");

  container.querySelectorAll(".comment-menu.active").forEach((m) => {
    m.classList.remove("active");
    m.querySelector(".comment-menu-btn")?.setAttribute(
      "aria-expanded",
      "false",
    );
  });

  if (!isActive) {
    menu.classList.add("active");
    btn.setAttribute("aria-expanded", "true");
  }
}

function closeAllMenus(container: HTMLElement): void {
  container.querySelectorAll(".comment-menu.active").forEach((m) => {
    m.classList.remove("active");
    m.querySelector(".comment-menu-btn")?.setAttribute(
      "aria-expanded",
      "false",
    );
  });
}

function editInline(
  displayEl: HTMLElement,
  onSave: (newValue: string) => Promise<void>,
): void {
  const currentValue = displayEl.textContent || "";

  const input = document.createElement("input");
  input.type = "text";
  input.className = displayEl.className + " bm-inline-input";
  input.value = currentValue;
  input.maxLength = 100;

  let finished = false;

  const restore = () => {
    input.replaceWith(displayEl);
  };

  const commit = async () => {
    if (finished) return;
    finished = true;
    const newValue = input.value.trim();
    restore();
    if (newValue && newValue !== currentValue) {
      await onSave(newValue);
    }
  };

  const cancel = () => {
    if (finished) return;
    finished = true;
    restore();
  };

  input.addEventListener("mousedown", (e) => {
    e.stopPropagation();
  });
  input.addEventListener("click", (e) => {
    e.preventDefault();
    e.stopPropagation();
  });

  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter") {
      e.preventDefault();
      input.blur();
    } else if (e.key === "Escape") {
      e.preventDefault();
      cancel();
    }
  });

  input.addEventListener("blur", () => {
    commit();
  });

  displayEl.replaceWith(input);
  input.focus();
  input.select();
}

async function loadBookmarksPage(
  container: HTMLElement,
  empty: HTMLElement | null,
  loading: HTMLElement | null,
): Promise<void> {
  const categories = await fetchBookmarkCategories(true);
  if (loading) loading.style.display = "none";

  const visibleCategories = categories.filter((c) => c.bookmarks.length >= 0);

  if (visibleCategories.length === 0) {
    if (empty) empty.style.display = "block";
    return;
  }
  if (empty) empty.style.display = "none";

  const catTemplate = document.getElementById(
    "tpl-bm-category",
  ) as HTMLTemplateElement;
  const itemTemplate = document.getElementById(
    "tpl-bm-item",
  ) as HTMLTemplateElement;

  container.innerHTML = "";

  container.addEventListener("click", (e) => {
    const target = e.target as HTMLElement;
    if (!target.closest(".comment-menu")) {
      closeAllMenus(container);
    }
  });

  visibleCategories.forEach((cat) => {
    const catNode = catTemplate.content.cloneNode(true) as HTMLElement;
    const nameEl = catNode.querySelector('[data-field="name"]') as HTMLElement;
    const countEl = catNode.querySelector(
      '[data-field="count"]',
    ) as HTMLElement;
    const menu = catNode.querySelector(".bm-category-menu") as HTMLElement;
    const menuBtn = menu.querySelector(".comment-menu-btn") as HTMLElement;
    const renameBtn = catNode.querySelector(
      ".bm-category-rename",
    ) as HTMLButtonElement;
    const deleteBtn = catNode.querySelector(
      ".bm-category-delete-item",
    ) as HTMLButtonElement;
    const itemsWrap = catNode.querySelector(".bm-items") as HTMLElement;

    nameEl.textContent = cat.name;
    nameEl.title = cat.name;
    countEl.textContent = String(cat.bookmarks.length);

    menuBtn.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      toggleMenu(container, menu);
    });

    renameBtn.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      closeAllMenus(container);

      editInline(nameEl, async (newValue) => {
        const res = await fetch(
          `${API_URL}/bookmarks/category/${encodeURIComponent(cat.name)}`,
          {
            method: "PATCH",
            credentials: "include",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ newName: newValue }),
          },
        );

        if (!res.ok) {
          console.error(
            "Failed to rename category:",
            res.status,
            await res.text(),
          );
          alert("Не удалось переименовать категорию");
          return;
        }

        await loadBookmarksPage(container, empty, loading);
      });
    });

    deleteBtn.addEventListener("click", async (e) => {
      e.preventDefault();
      e.stopPropagation();
      closeAllMenus(container);

      const confirmed = confirm(
        `Удалить категорию "${cat.name}" и все закладки в ней (${cat.bookmarks.length})?`,
      );
      if (!confirmed) return;

      const res = await fetch(
        `${API_URL}/bookmarks/category/${encodeURIComponent(cat.name)}`,
        { method: "DELETE", credentials: "include" },
      );

      if (!res.ok) {
        console.error(
          "Failed to delete category:",
          res.status,
          await res.text(),
        );
        alert("Не удалось удалить категорию");
        return;
      }

      await loadBookmarksPage(container, empty, loading);
    });

    cat.bookmarks.forEach((bm) => {
      const itemNode = itemTemplate.content.cloneNode(true) as HTMLElement;
      const sourceEl = itemNode.querySelector(
        '[data-field="source"]',
      ) as HTMLElement;
      const dateEl = itemNode.querySelector(
        '[data-field="date"]',
      ) as HTMLElement;
      const nameEl = itemNode.querySelector(
        '[data-field="name"]',
      ) as HTMLElement;
      const link = itemNode.querySelector(
        ".bm-item-name-link",
      ) as HTMLAnchorElement;
      const itemMenu = itemNode.querySelector(".bm-item-menu") as HTMLElement;
      const itemMenuBtn = itemMenu.querySelector(
        ".comment-menu-btn",
      ) as HTMLElement;
      const editBtn = itemNode.querySelector(
        ".bm-item-edit",
      ) as HTMLButtonElement;
      const itemDeleteBtn = itemNode.querySelector(
        ".bm-item-delete",
      ) as HTMLButtonElement;

      let nvTitle = bm.novelTitle;
      if (nvTitle.endsWith(".")) {
        nvTitle = nvTitle.slice(0, -1);
      }

      sourceEl.innerHTML = `${nvTitle}, глава <span class="chapter-num-highlight">${bm.chapterNum}</span>`;
      sourceEl.title = `${nvTitle}, глава ${bm.chapterNum}`;
      dateEl.textContent = formatRelativeTime(
        new Date(bm.createdAt * 1000).toISOString(),
      );
      nameEl.textContent = bm.name;
      nameEl.title = bm.name;
      link.href = `/${bm.novelId}/chapter/${bm.chapterId}`;

      itemMenuBtn.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        toggleMenu(container, itemMenu);
      });

      editBtn.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        closeAllMenus(container);

        editInline(nameEl, async (newValue) => {
          const res = await fetch(`${API_URL}/bookmarks/${bm.id}`, {
            method: "PATCH",
            credentials: "include",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ name: newValue }),
          });

          if (!res.ok) {
            console.error("Failed to update bookmark:", res.status);
            alert("Не удалось изменить закладку");
            return;
          }

          await loadBookmarksPage(container, empty, loading);
        });
      });

      itemDeleteBtn.addEventListener("click", async (e) => {
        e.preventDefault();
        e.stopPropagation();
        closeAllMenus(container);

        const res = await fetch(`${API_URL}/bookmarks/${bm.id}`, {
          method: "DELETE",
          credentials: "include",
        });
        if (!res.ok) {
          console.error("Failed to delete bookmark:", res.status);
          alert("Не удалось удалить закладку");
          return;
        }
        await loadBookmarksPage(container, empty, loading);
      });

      itemsWrap.appendChild(itemNode);
    });

    container.appendChild(catNode);
  });
}
