import Dropdown from "./dropdown";
import { profileManager } from "./profile";
import { uiManager } from "./ui";

const API_URL = process.env.API_URL;

interface Bookmark {
  id: string;
  novelId: string;
  chapterId: string;
  chapterNum: number;
  novelTitle: string;
  value: string;
  createdAt: number;
  updatedAt: number;
}

interface BookmarkCategory {
  createdAt: number;
  updatedAt: number;
  bookmarks: Bookmark[];
}

type BookmarkCategories = Record<string, BookmarkCategory>;

let cachedCategories: BookmarkCategories | null = null;

async function fetchBookmarkCategories(
  force = false,
): Promise<BookmarkCategories> {
  if (cachedCategories && !force) return cachedCategories;
  try {
    const res = await fetch(`${API_URL}/profile/me/bookmarks`, {
      credentials: "include",
    });
    if (!res.ok) return {};
    const data: BookmarkCategories = await res.json();
    cachedCategories = data;
    return cachedCategories;
  } catch {
    return {};
  }
}

function findBookmarkForChapter(
  categories: BookmarkCategories,
  chapterId: string,
): Bookmark | null {
  for (const cat of Object.values(categories)) {
    const found = cat.bookmarks.find((b) => b.chapterId === chapterId);
    if (found) return found;
  }
  return null;
}

function findCategoryOfBookmark(
  categories: BookmarkCategories,
  bookmarkId: string,
): string {
  for (const [name, cat] of Object.entries(categories)) {
    if (cat.bookmarks.some((b) => b.id === bookmarkId)) return name;
  }
  return "";
}

interface ChapterCtx {
  novelId: string;
  chapterId: string;
  chapterNum: number;
  novelTitle: string;
  chapterTitle: string;
}

let currentBookmark: Bookmark | null = null;
let currentCtx: ChapterCtx | null = null;

function cloneTemplate(id: string): DocumentFragment {
  const template = document.getElementById(id) as HTMLTemplateElement;
  return template.content.cloneNode(true) as DocumentFragment;
}

function setupBookmarkFormInstance(
  rootEl: HTMLElement,
  options: {
    bookmark: Bookmark | null;
    defaultTitle: string;
    categories: BookmarkCategories;
    preselectedCategory: string;
    cancelLabel?: string;
    isDangerDelete?: boolean;
    onSave: (value: string, category: string) => Promise<void>;
    onCancel: () => void | Promise<void>;
  },
): void {
  const valueEl = rootEl.querySelector("#bm-form-value") as HTMLTextAreaElement;
  const saveBtn = rootEl.querySelector("#bm-form-save") as HTMLButtonElement;
  const deleteBtn = rootEl.querySelector("#bm-form-delete") as HTMLButtonElement;
  const counter = rootEl.querySelector("#bm-char-counter") as HTMLElement;
  const dropdownEl = rootEl.querySelector(".dropdown") as HTMLElement | null;
  const dropdownContainer = rootEl.querySelector(
    ".dropdown-menu-inner-bm",
  ) as HTMLElement | null;
  const dropdownLabel = rootEl.querySelector(
    ".js-dropdown-label",
  ) as HTMLElement | null;

  const maxLen = 100;
  let selectedCategory = options.preselectedCategory || "Избранное";
  const initialValue = options.bookmark
    ? options.bookmark.value
    : options.defaultTitle;
  const initialCategory = selectedCategory;

  const getFinalCategory = (): string => {
    const inputRow = rootEl.querySelector(
      ".bm-category-input-row",
    ) as HTMLElement | null;
    const inlineInput = rootEl.querySelector(
      ".bm-category-input",
    ) as HTMLInputElement | null;
    if (
      inputRow
      && inputRow.style.display !== "none"
      && inlineInput
      && inlineInput.value.trim()
    ) {
      return inlineInput.value.trim();
    }
    return selectedCategory;
  };

  const updateSaveState = () => {
    if (!options.bookmark) {
      saveBtn.disabled = false;
      return;
    }
    const curVal = valueEl.value;
    const curCat = getFinalCategory();
    const hasChanged = curVal !== initialValue || curCat !== initialCategory;
    saveBtn.disabled = !hasChanged;
  };

  const updateCounter = () => {
    const len = valueEl.value.length;
    counter.textContent = `${len}/${maxLen}`;
    counter.classList.toggle("count-warning", len >= maxLen * 0.8);
    counter.classList.toggle("count-error", len >= maxLen);
  };

  const autoResize = () => {
    valueEl.style.height = "auto";
    valueEl.style.height = valueEl.scrollHeight + "px";
  };

  valueEl.placeholder = "Ваша заметка...";

  if (options.bookmark) {
    valueEl.value = options.bookmark.value;
    deleteBtn.style.display = "inline-flex";
    deleteBtn.textContent = options.cancelLabel || "Удалить";
    if (options.isDangerDelete) {
      deleteBtn.classList.add("danger-hover");
    } else {
      deleteBtn.classList.remove("danger-hover");
    }
    saveBtn.disabled = true;
  } else {
    valueEl.value = options.defaultTitle;
    saveBtn.disabled = false;
    deleteBtn.style.display = "none";
  }

  let selectedOnFirstInteraction = false;
  valueEl.addEventListener("focus", () => {
    if (!selectedOnFirstInteraction) {
      setTimeout(() => {
        valueEl.select();
      }, 0);
      selectedOnFirstInteraction = true;
    }
  });
  valueEl.addEventListener("mouseup", (e) => {
    if (!selectedOnFirstInteraction) {
      e.preventDefault();
      valueEl.select();
      selectedOnFirstInteraction = true;
    }
  });

  updateCounter();
  autoResize();

  valueEl.addEventListener("input", () => {
    selectedOnFirstInteraction = true;
    updateCounter();
    autoResize();
    updateSaveState();
  });

  if (dropdownEl && dropdownContainer && dropdownLabel) {
    dropdownContainer.innerHTML = "";
    const names = Object.keys(options.categories);
    if (!names.includes("Избранное")) {
      names.unshift("Избранное");
    }
    const initial = names.includes(options.preselectedCategory)
      ? options.preselectedCategory
      : names[0] || "Избранное";

    names.forEach((name) => {
      const node = cloneTemplate("tpl-bookmark-category-option");
      const btn = node.querySelector(".dropdown-item") as HTMLElement;
      const label = node.querySelector("[data-field=\"label\"]") as HTMLElement;
      btn.dataset.value = name;
      label.textContent = name;
      const isSelected = name === initial;
      btn.classList.toggle("selected", isSelected);
      btn.setAttribute("aria-selected", String(isSelected));
      dropdownContainer.appendChild(node);
    });

    dropdownContainer.appendChild(cloneTemplate("tpl-bookmark-category-new"));

    selectedCategory = initial;
    dropdownLabel.textContent = initial;

    const dropdown = new Dropdown(dropdownEl);

    const newWrap = dropdownContainer.querySelector(".dropdown-item-new-wrap");
    const newBtn = newWrap?.querySelector(
      "button.dropdown-item-new",
    ) as HTMLElement | null;
    const inputRow = newWrap?.querySelector(
      ".bm-category-input-row",
    ) as HTMLElement | null;
    const newInput = newWrap?.querySelector(
      ".bm-category-input",
    ) as HTMLInputElement | null;

    if (newBtn && inputRow && newInput) {
      const showInput = () => {
        newBtn.style.display = "none";
        inputRow.style.display = "flex";
        newInput.value = "";
        newInput.focus();
      };

      const showButton = () => {
        inputRow.style.display = "none";
        newBtn.style.display = "";
        newInput.value = "";
      };

      newBtn.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        showInput();
      });

      inputRow.addEventListener("click", (e) => {
        e.stopPropagation();
        newInput.focus();
      });

      const commit = () => {
        const val = newInput.value.trim();
        if (val) {
          selectedCategory = val;
          dropdownLabel.textContent = val;
          dropdownContainer.querySelectorAll(".dropdown-item").forEach((i) => {
            i.classList.remove("selected");
            i.setAttribute("aria-selected", "false");
          });
          showButton();
          updateSaveState();
          dropdown.close();
        } else {
          showButton();
        }
      };

      newInput.addEventListener("keydown", (e) => {
        e.stopPropagation();
        if (e.key === "Enter") {
          e.preventDefault();
          commit();
        } else if (e.key === "Escape") {
          e.preventDefault();
          showButton();
        }
      });

      newInput.addEventListener("blur", () => {
        commit();
      });
    }

    dropdownEl.addEventListener("change", (e: Event) => {
      const value = (e as CustomEvent).detail.value as string;
      if (value && value !== "__new__") {
        selectedCategory = value;
        dropdownLabel.textContent = value;
        updateSaveState();
      }
    });
  }

  saveBtn.addEventListener("click", async () => {
    const category = getFinalCategory();
    const value = valueEl.value;
    await options.onSave(value, category);
  });

  deleteBtn.addEventListener("click", () => {
    options.onCancel();
  });
}

function renderBookmarkForm(): void {
  const card = document.getElementById("bookmark-card");
  if (!card || !currentCtx) return;

  card.innerHTML = "";
  card.appendChild(cloneTemplate("tpl-bookmark-form"));

  fetchBookmarkCategories().then((categories) => {
    const preselected = currentBookmark
      ? findCategoryOfBookmark(categories, currentBookmark.id)
      : "";
    setupBookmarkFormInstance(card, {
      bookmark: currentBookmark,
      defaultTitle: currentCtx!.chapterTitle || `Глава ${currentCtx!.chapterNum}`,
      categories,
      preselectedCategory: preselected,
      cancelLabel: "Удалить",
      isDangerDelete: true,
      onSave: async (value, category) => {
        if (currentBookmark) {
          const res = await fetch(
            `${API_URL}/bookmarks/${currentBookmark.id}`,
            {
              method: "PATCH",
              credentials: "include",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ value, category }),
            },
          );
          if (!res.ok) {
            const data = await res.json().catch(() => null);
            alert(data?.detail || "Не удалось сохранить закладку");
            return;
          }
        } else {
          const res = await fetch(`${API_URL}/bookmarks`, {
            method: "POST",
            credentials: "include",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              chapterId: currentCtx!.chapterId,
              category,
              value,
            }),
          });
          if (!res.ok) {
            const data = await res.json().catch(() => null);
            alert(data?.detail || "Не удалось добавить закладку");
            return;
          }
        }
        uiManager.closeAll();
        await refreshButtonState();
      },
      onCancel: async () => {
        if (!currentBookmark) return;
        await fetch(`${API_URL}/bookmarks/${currentBookmark.id}`, {
          method: "DELETE",
          credentials: "include",
        });
        uiManager.closeAll();
        await refreshButtonState();
      },
    });
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
    chapterTitle: tracker.dataset.chapterTitle || "",
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

let isEditingCategories = false;

async function saveCategoriesEdit(
  container: HTMLElement,
  empty: HTMLElement | null,
  loading: HTMLElement | null,
  btn: HTMLButtonElement,
): Promise<void> {
  btn.disabled = true;
  try {
    const inputs = container.querySelectorAll<HTMLInputElement>(
      ".bm-category-edit-input",
    );
    const promises: Promise<Response>[] = [];

    inputs.forEach((input) => {
      const oldName = input.dataset.originalName || "";
      const newName = input.value.trim();
      if (newName && newName !== oldName) {
        promises.push(
          fetch(`${API_URL}/bookmarks/category/${encodeURIComponent(oldName)}`, {
            method: "PATCH",
            credentials: "include",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ newName }),
          }),
        );
      }
    });

    if (promises.length > 0) {
      const results = await Promise.all(promises);
      const failed = results.filter((res) => !res.ok);
      if (failed.length > 0) {
        alert("Не удалось переименовать некоторые категории");
      }
    }
  } catch (err) {
    console.error("Failed to save categories:", err);
    alert("Ошибка сети при сохранении категорий");
  } finally {
    isEditingCategories = false;
    btn.disabled = false;
    btn.textContent = "Переименовать";
    await loadBookmarksPage(container, empty, loading);
  }
}

export function initBookmarksPage(): void {
  const container = document.getElementById("bm-categories");
  if (!container) return;

  container.addEventListener("click", (e) => {
    const target = e.target as HTMLElement;
    if (!target.closest(".comment-menu")) {
      closeAllMenus(container);
    }
  });

  const empty = document.getElementById("bm-empty");
  const loading = document.getElementById("bm-loading");
  const editCategoriesBtn = document.getElementById(
    "bm-edit-categories-btn",
  ) as HTMLButtonElement | null;

  if (!profileManager.isLoggedIn()) {
    if (loading) loading.style.display = "none";
    if (empty) {
      empty.style.display = "block";
      empty.querySelector("p")!.textContent = "Войдите в аккаунт, чтобы видеть свои закладки";
    }
    if (editCategoriesBtn) editCategoriesBtn.style.display = "none";
    return;
  }

  isEditingCategories = false;

  if (editCategoriesBtn) {
    editCategoriesBtn.addEventListener("click", async () => {
      if (!isEditingCategories) {
        isEditingCategories = true;
        editCategoriesBtn.textContent = "Сохранить";
        container.querySelectorAll(".bm-category-wrapper").forEach((wrap, i) => {
          const details = wrap.querySelector("details.bm-category") as HTMLDetailsElement | null;
          const editWrap = wrap.querySelector(".bm-category-edit-wrap") as HTMLElement | null;
          const input = wrap.querySelector(".bm-category-edit-input") as HTMLInputElement | null;
          if (details) {
            details.open = false;
            details.style.display = "none";
          }
          if (editWrap) {
            editWrap.style.display = "block";
          }
          if (i === 0 && input) {
            input.focus();
          }
        });
      } else {
        await saveCategoriesEdit(container, empty, loading, editCategoriesBtn);
      }
    });
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

async function loadBookmarksPage(
  container: HTMLElement,
  empty: HTMLElement | null,
  loading: HTMLElement | null,
  keepOpenCategoryNames?: Set<string>,
): Promise<void> {
  const editCategoriesBtn = document.getElementById(
    "bm-edit-categories-btn",
  ) as HTMLButtonElement | null;

  const openCategoryNames = keepOpenCategoryNames || new Set<string>();
  if (!keepOpenCategoryNames) {
    container.querySelectorAll("details.bm-category[open]").forEach((d) => {
      const name = d.querySelector("[data-field=\"name\"]")?.textContent?.trim();
      if (name) openCategoryNames.add(name);
    });
  }

  const categories = await fetchBookmarkCategories(true);
  if (loading) loading.style.display = "none";

  const categoryEntries = Object.entries(categories);

  container.innerHTML = "";

  if (categoryEntries.length === 0) {
    if (empty) empty.style.display = "block";
    if (editCategoriesBtn) editCategoriesBtn.style.display = "none";
    return;
  }
  if (empty) empty.style.display = "none";
  if (editCategoriesBtn) {
    editCategoriesBtn.style.display = "inline-flex";
    editCategoriesBtn.textContent = "Переименовать";
  }

  const catTemplate = document.getElementById(
    "tpl-bm-category",
  ) as HTMLTemplateElement;
  const itemTemplate = document.getElementById(
    "tpl-bm-item",
  ) as HTMLTemplateElement;

  categoryEntries.forEach(([catName, cat], index) => {
    const catNode = catTemplate.content.cloneNode(true) as HTMLElement;
    const detailsEl = catNode.querySelector("details.bm-category") as HTMLDetailsElement;
    const nameEl = catNode.querySelector("[data-field=\"name\"]") as HTMLElement;
    const countEl = catNode.querySelector(
      "[data-field=\"count\"]",
    ) as HTMLElement;
    const editInput = catNode.querySelector(
      "[data-field=\"edit-input\"]",
    ) as HTMLInputElement | null;
    const itemsWrap = catNode.querySelector(".bm-items") as HTMLElement;

    nameEl.textContent = catName;
    nameEl.title = catName;
    countEl.textContent = `(${cat.bookmarks.length})`;

    if (editInput) {
      editInput.value = catName;
      editInput.dataset.originalName = catName;
      editInput.addEventListener("keydown", (e) => {
        if (e.key === "Enter") {
          e.preventDefault();
          if (editCategoriesBtn) {
            saveCategoriesEdit(container, empty, loading, editCategoriesBtn);
          }
        }
      });
    }

    if (openCategoryNames.size > 0) {
      if (openCategoryNames.has(catName)) {
        detailsEl.open = true;
      }
    } else if (index === 0) {
      detailsEl.open = true;
    }

    cat.bookmarks.forEach((bm) => {
      const itemNode = itemTemplate.content.cloneNode(true) as HTMLElement;
      const viewEl = itemNode.querySelector(".bm-item-view") as HTMLElement;
      const slotEl = itemNode.querySelector(
        ".bm-item-edit-slot",
      ) as HTMLElement;
      const sourceEl = itemNode.querySelector(
        "[data-field=\"source\"]",
      ) as HTMLElement;
      const nameEl = (itemNode.querySelector("[data-field=\"value\"]")
        || itemNode.querySelector("[data-field=\"name\"]")) as HTMLElement;
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

      const titleSpan = sourceEl.querySelector("[data-field=\"source-title\"]") as HTMLElement;
      const chapterSpan = sourceEl.querySelector("[data-field=\"source-chapter\"]") as HTMLElement;
      titleSpan.textContent = nvTitle;
      chapterSpan.textContent = String(bm.chapterNum);
      sourceEl.title = `${nvTitle}, глава ${bm.chapterNum}`;
      nameEl.textContent = bm.value;
      link.href = bm.novelId && bm.chapterId
        ? `/${encodeURIComponent(bm.novelId)}/chapter/${encodeURIComponent(bm.chapterId)}`
        : "#";

      itemMenuBtn.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        toggleMenu(container, itemMenu);
      });

      editBtn.addEventListener("click", async (e) => {
        e.preventDefault();
        e.stopPropagation();
        closeAllMenus(container);

        container.querySelectorAll(".bm-item-edit-slot").forEach((s) => {
          s.innerHTML = "";
          (s as HTMLElement).style.display = "none";
        });
        container.querySelectorAll(".bm-item-view").forEach((v) => {
          (v as HTMLElement).style.display = "";
        });

        if (!viewEl || !slotEl) return;

        viewEl.style.display = "none";
        slotEl.style.display = "block";
        slotEl.innerHTML = "";
        slotEl.appendChild(cloneTemplate("tpl-bookmark-form"));

        const latestCategories = await fetchBookmarkCategories(true);
        const preselected = findCategoryOfBookmark(latestCategories, bm.id);

        setupBookmarkFormInstance(slotEl, {
          bookmark: bm,
          defaultTitle: `Глава ${bm.chapterNum}`,
          categories: latestCategories,
          preselectedCategory: preselected,
          cancelLabel: "Отмена",
          isDangerDelete: false,
          onSave: async (value, category) => {
            const res = await fetch(`${API_URL}/bookmarks/${bm.id}`, {
              method: "PATCH",
              credentials: "include",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ value, category }),
            });
            if (!res.ok) {
              const data = await res.json().catch(() => null);
              alert(data?.detail || "Не удалось сохранить закладку");
              return;
            }
            const keepOpen = new Set<string>();
            container.querySelectorAll("details.bm-category[open]").forEach((d) => {
              const n = d.querySelector("[data-field=\"name\"]")?.textContent?.trim();
              if (n) keepOpen.add(n);
            });
            keepOpen.add(catName);
            if (category) keepOpen.add(category);
            await loadBookmarksPage(container, empty, loading, keepOpen);
          },
          onCancel: () => {
            slotEl.innerHTML = "";
            slotEl.style.display = "none";
            viewEl.style.display = "";
          },
        });
      });

      itemDeleteBtn.addEventListener("click", async (e) => {
        e.preventDefault();
        e.stopPropagation();
        closeAllMenus(container);

        const keepOpen = new Set<string>();
        container.querySelectorAll("details.bm-category[open]").forEach((d) => {
          const n = d.querySelector("[data-field=\"name\"]")?.textContent?.trim();
          if (n) keepOpen.add(n);
        });
        keepOpen.add(catName);

        const res = await fetch(`${API_URL}/bookmarks/${bm.id}`, {
          method: "DELETE",
          credentials: "include",
        });
        if (!res.ok) {
          console.error("Failed to delete bookmark:", res.status);
          alert("Не удалось удалить закладку");
          return;
        }
        await loadBookmarksPage(container, empty, loading, keepOpen);
      });

      itemsWrap.appendChild(itemNode);
    });

    container.appendChild(catNode);
  });
}
