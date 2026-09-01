import { trackEvent } from "./analytics";
import { profileManager } from "./profile";

const API_URL = process.env.API_URL;

interface EnrichedListEntry {
  id: string;
  addedAt: number;
  title: string;
  author: string;
  coverUrl: string | null;
}

interface EnrichedListCategory {
  createdAt: number;
  updatedAt: number;
  novels: EnrichedListEntry[];
}

type UserList = Record<string, EnrichedListCategory>;

const FALLBACK_COVER =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='200' height='300'%3E%3Crect fill='%23ecf0f1' width='200' height='300'/%3E%3C/svg%3E";

const LIST_STATUSES: { slug: string; label: string }[] = [
  { slug: "favorite", label: "Избранное" },
  { slug: "reading", label: "Читаю" },
  { slug: "rereading", label: "Перечитываю" },
  { slug: "planned", label: "Запланировано" },
  { slug: "on_hold", label: "Отложено" },
  { slug: "completed", label: "Прочитано" },
  { slug: "dropped", label: "Брошено" },
];

let cachedList: UserList | null = null;
const hydrateFns: (() => void)[] = [];

function hydrateAll(): void {
  hydrateFns.forEach((fn) => fn());
}

async function fetchUserList(force = false): Promise<UserList> {
  if (cachedList && !force) return cachedList;
  try {
    const res = await fetch(`${API_URL}/profile/me/list`, {
      credentials: "include",
    });
    if (!res.ok) return {};
    const data: UserList = await res.json();
    cachedList = data;
    return cachedList;
  } catch {
    return {};
  }
}

function findStatusOfNovel(list: UserList, novelId: string): string | null {
  for (const [slug, cat] of Object.entries(list)) {
    if (cat.novels.some((entry) => entry.id === novelId)) return slug;
  }
  return null;
}

function cloneTemplate(id: string): DocumentFragment {
  const template = document.getElementById(id) as HTMLTemplateElement;
  return template.content.cloneNode(true) as DocumentFragment;
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

async function removeFromList(novelId: string): Promise<boolean> {
  try {
    const res = await fetch(`${API_URL}/list/${encodeURIComponent(novelId)}`, {
      method: "DELETE",
      credentials: "include",
    });
    if (res.ok) {
      cachedList = null;
      trackEvent("list_remove");
      return true;
    }
    const data = await res.json().catch(() => null);
    alert(data?.detail || "Не удалось удалить новеллу из списка");
    return false;
  } catch {
    alert("Ошибка сети при удалении из списка");
    return false;
  }
}

async function loadListPage(
  container: HTMLElement,
  empty: HTMLElement | null,
  loading: HTMLElement | null,
): Promise<void> {
  const openSlugs = new Set<string>();
  container.querySelectorAll("details.bm-category[open]").forEach((d) => {
    const slug = (d as HTMLDetailsElement).dataset.slug;
    if (slug) openSlugs.add(slug);
  });

  const list = await fetchUserList(true);
  if (loading) loading.style.display = "none";
  if (empty) empty.style.display = "none";

  container.innerHTML = "";

  const firstNonEmptySlug = LIST_STATUSES.find(
    ({ slug }) => (list[slug]?.novels.length ?? 0) > 0,
  )?.slug;

  const iconsTemplate = document.getElementById(
    "tpl-ls-icons",
  ) as HTMLTemplateElement | null;
  const cardTemplate = document.getElementById(
    "tpl-ls-novel-card",
  ) as HTMLTemplateElement;

  LIST_STATUSES.forEach(({ slug, label }) => {
    const novels = list[slug]?.novels ?? [];

    const catNode = cloneTemplate("tpl-ls-category");
    const detailsEl = catNode.querySelector(
      "details.bm-category",
    ) as HTMLDetailsElement;
    const nameEl = catNode.querySelector(
      "[data-field=\"name\"]",
    ) as HTMLElement;
    const countEl = catNode.querySelector(
      "[data-field=\"count\"]",
    ) as HTMLElement;
    const itemsWrap = catNode.querySelector(".bm-items") as HTMLElement;

    nameEl.textContent = label;
    nameEl.title = label;
    countEl.textContent = `(${novels.length})`;
    detailsEl.dataset.slug = slug;

    const iconSlot = catNode.querySelector(
      ".ls-cat-icon-slot",
    ) as HTMLElement | null;
    const iconSvg = iconsTemplate?.content.querySelector(
      `.ls-icon[data-slug="${slug}"]`,
    );
    if (iconSlot && iconSvg) iconSlot.appendChild(iconSvg.cloneNode(true));

    if (openSlugs.size > 0) {
      if (openSlugs.has(slug)) detailsEl.open = true;
    } else if (slug === firstNonEmptySlug) {
      detailsEl.open = true;
    }

    novels.forEach((entry) => {
      const cardNode = cardTemplate.content.cloneNode(true) as HTMLElement;
      const link = cardNode.querySelector("a.novel-card") as HTMLAnchorElement;
      const poster = cardNode.querySelector(".poster-wrapper") as HTMLElement;
      const img = cardNode.querySelector("img") as HTMLImageElement;
      const titleEl = cardNode.querySelector("h3") as HTMLElement;
      const authorEl = cardNode.querySelector(".author") as HTMLElement;
      const menu = cardNode.querySelector(".ls-card-menu") as HTMLElement;
      const menuBtn = cardNode.querySelector(
        ".comment-menu-btn",
      ) as HTMLElement;
      const deleteBtn = cardNode.querySelector(
        ".ls-item-delete",
      ) as HTMLButtonElement;

      link.href = `/${encodeURIComponent(entry.id)}`;
      const cover = entry.coverUrl || FALLBACK_COVER;
      img.src = cover;
      img.alt = entry.title;
      poster.style.setProperty("--bg-url", `url(${cover})`);
      titleEl.textContent = entry.title;
      authorEl.textContent = entry.author || "";

      menuBtn.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        toggleMenu(container, menu);
      });

      deleteBtn.addEventListener("click", async (e) => {
        e.preventDefault();
        e.stopPropagation();
        if (await removeFromList(entry.id)) {
          await loadListPage(container, empty, loading);
        }
      });

      itemsWrap.appendChild(cardNode);
    });

    if (novels.length === 0) {
      const note = document.createElement("div");
      note.className = "mc-empty";
      note.textContent = "Здесь пока ничего нет";
      itemsWrap.appendChild(note);
    }

    container.appendChild(catNode);
  });
}

export function initListPage(): void {
  const container = document.getElementById("ls-categories");
  if (!container) return;

  container.addEventListener("click", (e) => {
    const target = e.target as HTMLElement;
    if (!target.closest(".comment-menu")) {
      closeAllMenus(container);
    }
  });

  const empty = document.getElementById("ls-empty");
  const loading = document.getElementById("ls-loading");

  if (!profileManager.isLoggedIn()) {
    if (loading) loading.style.display = "none";
    if (empty) {
      empty.style.display = "block";
      empty.querySelector("p")!.textContent = "Войдите в аккаунт, чтобы видеть свой список";
    }
    return;
  }

  void loadListPage(container, empty, loading);
}

function initNovelListDropdownInstance(dropdownEl: HTMLElement): void {
  const novelId = dropdownEl.dataset.novelId;
  if (!novelId) return;

  const iconSlot = dropdownEl.querySelector(
    ".ls-btn-icon",
  ) as HTMLElement | null;
  const defaultIcon = iconSlot?.querySelector("svg")?.cloneNode(true) ?? null;
  const removeWrap = dropdownEl.querySelector(
    ".ls-remove-wrap",
  ) as HTMLElement | null;
  const removeItem = dropdownEl.querySelector(
    ".dropdown-item-remove",
  ) as HTMLElement | null;

  let currentStatus: string | null = null;
  let inFlight = false;

  const applyStatus = (slug: string | null) => {
    currentStatus = slug;
    dropdownEl
      .querySelectorAll<HTMLElement>(".dropdown-item")
      .forEach((item) => {
        const isSelected = slug !== null && item.dataset.value === slug;
        item.classList.toggle("selected", isSelected);
        item.setAttribute("aria-selected", String(isSelected));
      });
    if (removeWrap) removeWrap.style.display = slug ? "" : "none";
    if (iconSlot) {
      iconSlot.replaceChildren();
      if (slug) {
        const statusIcon = dropdownEl.querySelector(
          `.dropdown-item[data-value="${slug}"] .ls-icon`,
        );
        if (statusIcon) iconSlot.appendChild(statusIcon.cloneNode(true));
      }
      if (!iconSlot.firstChild && defaultIcon) {
        iconSlot.appendChild(defaultIcon.cloneNode(true));
      }
    }
  };

  const closeDropdown = () => {
    dropdownEl.classList.remove("active");
    dropdownEl
      .querySelector(".dropdown-btn")
      ?.setAttribute("aria-expanded", "false");
  };

  const hydrate = async () => {
    if (!profileManager.isLoggedIn()) return;
    const list = await fetchUserList();
    applyStatus(findStatusOfNovel(list, novelId));
  };

  const moveToList = async (slug: string) => {
    const previous = currentStatus;
    inFlight = true;
    try {
      const res = await fetch(`${API_URL}/list`, {
        method: "PUT",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ novelId, status: slug }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => null);
        alert(data?.detail || "Не удалось обновить список");
        applyStatus(previous);
        return;
      }
      trackEvent(previous ? "list_move" : "list_add");
      cachedList = null;
      hydrateAll();
    } catch {
      alert("Ошибка сети при обновлении списка");
      applyStatus(previous);
    } finally {
      inFlight = false;
    }
  };

  const removeFromCurrent = async () => {
    const previous = currentStatus;
    if (previous === null) return;
    applyStatus(null);
    const ok = await removeFromList(novelId);
    if (!ok) applyStatus(previous);
    hydrateAll();
  };

  dropdownEl.addEventListener("change", async (e: Event) => {
    const value = (e as CustomEvent).detail.value as string;
    if (!value || value === "__remove__" || inFlight) return;

    if (!profileManager.isLoggedIn()) {
      alert("Войдите в аккаунт, чтобы добавить новеллу в список");
      applyStatus(currentStatus);
      return;
    }

    if (value === currentStatus) {
      closeDropdown();
      await removeFromCurrent();
      return;
    }

    await moveToList(value);
  });

  if (removeItem) {
    removeItem.addEventListener("click", async (e) => {
      e.preventDefault();
      e.stopPropagation();
      if (inFlight) return;
      if (!profileManager.isLoggedIn()) {
        alert("Войдите в аккаунт, чтобы добавить новеллу в список");
        return;
      }
      closeDropdown();
      await removeFromCurrent();
    });
  }

  hydrateFns.push(() => {
    void hydrate();
  });
  profileManager.onLogin(() => {
    void hydrate();
  });
  void hydrate();
}

export function initNovelListDropdown(): void {
  document
    .querySelectorAll<HTMLElement>(".novel-list-dropdown[data-novel-id]")
    .forEach((el) => initNovelListDropdownInstance(el));
}
