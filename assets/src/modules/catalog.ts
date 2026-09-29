import { setKappalibCookie } from "./profile";

export async function loadCatalogPage(
  url: string,
  isHistoryPush: boolean = true,
): Promise<void> {
  const container = document.getElementById("catalog-content");
  if (!container) return;

  const urlObj = new URL(url, window.location.origin);
  const pageParam = urlObj.searchParams.get("page");
  const isFirstPage = !pageParam || pageParam === "1";

  if (isFirstPage) {
    urlObj.searchParams.delete("page");
  }
  const finalUrl = urlObj.toString();

  console.info(`Loading catalog content from: ${finalUrl}`);
  container.classList.add("is-loading");

  try {
    const [response] = await Promise.all([
      fetch(finalUrl),
      new Promise<void>((resolve) => setTimeout(resolve, 120)),
    ]);

    if (!response.ok) throw new Error(`HTTP error ${response.status}`);
    const htmlText = await response.text();

    const parser = new DOMParser();
    const doc = parser.parseFromString(htmlText, "text/html");
    const newContent = doc.getElementById("catalog-content");

    if (newContent) {
      container.innerHTML = newContent.innerHTML;

      const currentUpdates = container.querySelector(".updates-section");
      const incomingUpdates = newContent.querySelector(".updates-section");

      if (isFirstPage) {
        if (incomingUpdates && !currentUpdates) {
          container.appendChild(incomingUpdates);
        }
      } else {
        if (currentUpdates) {
          currentUpdates.remove();
        }
      }

      if (isHistoryPush) {
        window.history.pushState({}, "", finalUrl);
      }

      console.info("Catalog page updated successfully");

      const titleElement = document.getElementById("catalog-title")
        || document.querySelector("h2");

      if (titleElement) {
        const header = document.getElementById("main-header");
        const headerHeight = header ? header.offsetHeight : 0;
        const extraMargin = 24;
        const elementPosition = titleElement.getBoundingClientRect().top;
        const offsetPosition = elementPosition + window.scrollY - headerHeight - extraMargin;

        window.scrollTo({
          top: offsetPosition,
          behavior: "smooth",
        });
      }
    } else {
      console.warn("New content not found, falling back to reload");
      window.location.href = finalUrl;
    }
  } catch (err) {
    console.error("Catalog load failed", err);
    window.location.href = finalUrl;
  } finally {
    container.classList.remove("is-loading");
  }
}

export function initCatalogPagination(): void {
  const container = document.getElementById("catalog-content");
  if (!container) return;

  container.addEventListener("click", (e: Event) => {
    const target = e.target as HTMLElement;
    const link = target.closest(".page-link") as HTMLAnchorElement | null;

    if (
      !link
      || !link.href
      || link.classList.contains("active")
      || link.classList.contains("disabled")
    ) {
      return;
    }

    e.preventDefault();
    loadCatalogPage(link.href, true);
  });

  window.addEventListener("popstate", () => {
    loadCatalogPage(window.location.href, false);
  });
}

let isLoading = false;
let currentPage = 1;
let totalPages = 1;
let currentSort = "popular";
let generation = 0;
let reloadController: AbortController | null = null;
let reloadTimeout: number | undefined;
let loaderObserver: IntersectionObserver | null = null;

function getRangeValue(input: HTMLInputElement): string {
  const value = parseInt(input.value, 10);
  const min = parseInt(input.dataset.min || "0", 10);
  const max = parseInt(input.dataset.max || String(Number.MAX_SAFE_INTEGER), 10);
  if (Number.isNaN(value) || value < min || value > max) return "";
  return String(value);
}

function appendFilterParams(params: URLSearchParams): void {
  const filters = document.getElementById("catalog-filters");
  if (!filters) return;

  filters.querySelectorAll<HTMLElement>(".catalog-filter").forEach((filter) => {
    const param = filter.dataset.filter;
    if (!param) return;
    filter
      .querySelectorAll<HTMLElement>(".dropdown-item.selected")
      .forEach((item) => {
        if (item.dataset.value) params.append(param, item.dataset.value);
      });
  });

  filters
    .querySelectorAll<HTMLInputElement>("input[name]")
    .forEach((input) => {
      const value = getRangeValue(input);
      if (value) params.set(input.name, value);
    });
}

function getBaseParams(): URLSearchParams {
  const params = new URLSearchParams();

  params.set("sort", currentSort);

  const catalogContent = document.getElementById("catalog-content");
  const search = catalogContent?.dataset.search;
  if (search) {
    params.set("search", search);
  }

  appendFilterParams(params);

  return params;
}

function formatRangeBadge(from: string, to: string): string {
  if (from && to) return `${from}–${to}`;
  if (from) return `от ${from}`;
  if (to) return `до ${to}`;
  return "";
}

function getFilterBadgeText(filter: HTMLElement): string {
  const inputs = filter.querySelectorAll<HTMLInputElement>("input[name]");
  if (inputs.length === 2) {
    return formatRangeBadge(getRangeValue(inputs[0]), getRangeValue(inputs[1]));
  }

  const selected = filter.querySelectorAll(".dropdown-item.selected").length;
  return selected > 0 ? String(selected) : "";
}

function updateFilterBadges(): void {
  const filters = document.getElementById("catalog-filters");
  if (!filters) return;

  let hasActive = false;

  filters.querySelectorAll<HTMLElement>(".catalog-filter").forEach((filter) => {
    const badgeText = getFilterBadgeText(filter);

    const badge = filter.querySelector<HTMLElement>(".catalog-filter-badge");
    if (badge) {
      badge.textContent = badgeText;
      badge.hidden = badgeText === "";
    }
    filter
      .querySelector(".dropdown-btn")
      ?.classList.toggle("has-value", badgeText !== "");

    if (badgeText) hasActive = true;
  });

  const reset = document.getElementById("catalog-filters-reset");
  if (reset) reset.hidden = !hasActive;
}

function scheduleReload(delay: number): void {
  clearTimeout(reloadTimeout);
  reloadTimeout = window.setTimeout(() => {
    reloadCatalog();
  }, delay);
}

function observeLoader(): void {
  const loader = document.getElementById("catalog-loader");
  if (!loader || !loaderObserver) return;
  loaderObserver.unobserve(loader);
  loaderObserver.observe(loader);
}

async function loadMoreNovels(): Promise<void> {
  if (isLoading || reloadController || currentPage >= totalPages) return;

  isLoading = true;
  const requestGeneration = generation;
  const loader = document.getElementById("catalog-loader");
  if (loader) loader.style.display = "flex";

  try {
    const params = getBaseParams();
    params.set("page", String(currentPage + 1));

    const response = await fetch(`/catalog?${params.toString()}`, {
      headers: { "X-Partial": "true" },
    });

    if (!response.ok) throw new Error(`HTTP error ${response.status}`);

    const html = await response.text();
    if (requestGeneration !== generation) return;

    const parser = new DOMParser();
    const doc = parser.parseFromString(html, "text/html");
    const newGrid = doc.getElementById("catalog-grid");

    if (newGrid) {
      const currentGrid = document.getElementById("catalog-grid");
      if (currentGrid) {
        const newCards = newGrid.querySelectorAll(".novel-card");
        newCards.forEach((card) => {
          currentGrid.appendChild(card.cloneNode(true));
        });
      }
      currentPage++;
    }

    if (currentPage >= totalPages && loader) {
      loader.style.display = "none";
    }

    console.info(`Loaded catalog page ${currentPage}/${totalPages}`);
  } catch (err) {
    console.error("Failed to load more novels", err);
  } finally {
    isLoading = false;
    const loader = document.getElementById("catalog-loader");
    if (loader && currentPage < totalPages) {
      loader.style.display = "flex";
    }
    if (requestGeneration !== generation) observeLoader();
  }
}

async function reloadCatalog(): Promise<void> {
  const catalogContent = document.getElementById("catalog-content");
  if (!catalogContent) return;

  clearTimeout(reloadTimeout);
  reloadController?.abort();
  const controller = new AbortController();
  reloadController = controller;
  generation++;

  const params = getBaseParams();
  catalogContent.classList.add("is-loading");

  try {
    const response = await fetch(`/catalog?${params.toString()}`, {
      headers: { "X-Partial": "results" },
      signal: controller.signal,
    });

    if (!response.ok) throw new Error(`HTTP error ${response.status}`);

    const html = await response.text();
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, "text/html");
    const newContent = doc.getElementById("catalog-content");
    if (!newContent) throw new Error("Catalog content not found in response");

    catalogContent.innerHTML = newContent.innerHTML;
    updateURL(params);

    currentPage = 1;
    totalPages = parseInt(newContent.dataset.totalPages || "1", 10);
    catalogContent.dataset.totalPages = String(totalPages);

    const loader = document.getElementById("catalog-loader");
    if (loader) {
      loader.style.display = currentPage < totalPages ? "flex" : "none";
    }
    observeLoader();

    console.info("Catalog results updated");
  } catch (err) {
    if (controller.signal.aborted) return;
    console.error("Failed to reload catalog, falling back to navigation", err);
    window.location.href = catalogURL(params);
  } finally {
    if (reloadController === controller) {
      reloadController = null;
      catalogContent.classList.remove("is-loading");
    }
  }
}

function catalogURL(params: URLSearchParams): string {
  const url = new URL(window.location.origin + "/catalog");
  params.forEach((value, key) => {
    url.searchParams.append(key, value);
  });
  return url.toString();
}

function updateURL(params: URLSearchParams): void {
  window.history.replaceState({}, "", catalogURL(params));
}

function initCatalogFilters(): void {
  const filters = document.getElementById("catalog-filters");
  if (!filters) return;

  filters.addEventListener("click", (e: Event) => {
    const target = e.target as HTMLElement;

    const option = target.closest<HTMLElement>(".dropdown-item");
    if (option && filters.contains(option)) {
      const selected = option.classList.toggle("selected");
      option.setAttribute("aria-selected", String(selected));
      updateFilterBadges();
      scheduleReload(250);
      return;
    }

    if (target.closest("#catalog-filters-reset")) {
      filters
        .querySelectorAll<HTMLElement>(".dropdown-item.selected")
        .forEach((item) => {
          item.classList.remove("selected");
          item.setAttribute("aria-selected", "false");
        });
      filters.querySelectorAll<HTMLInputElement>("input").forEach((input) => {
        input.value = "";
      });
      filterTagOptions(filters, "");
      updateFilterBadges();
      reloadCatalog();
    }
  });

  filters.addEventListener("input", (e: Event) => {
    const input = e.target as HTMLInputElement;

    if (input.classList.contains("catalog-filter-search")) {
      filterTagOptions(filters, input.value);
      return;
    }

    if (!input.name) return;
    const maxDigits = (input.dataset.max || "").length || undefined;
    const digits = input.value.replace(/\D/g, "").slice(0, maxDigits);
    if (digits !== input.value) input.value = digits;
    updateFilterBadges();
    scheduleReload(600);
  });

  filters.addEventListener("keydown", (e: KeyboardEvent) => {
    const input = e.target as HTMLInputElement;
    if (e.key === "Enter" && input.name) {
      e.preventDefault();
      reloadCatalog();
    }
  });
}

function filterTagOptions(filters: HTMLElement, query: string): void {
  const normalized = query.trim().toLowerCase();
  let visible = 0;

  filters
    .querySelectorAll<HTMLElement>(".catalog-filter-options .dropdown-item")
    .forEach((item) => {
      const matches = !normalized
        || (item.textContent || "").toLowerCase().includes(normalized);
      item.hidden = !matches;
      if (matches) visible++;
    });

  const empty = filters.querySelector<HTMLElement>(".catalog-filter-empty");
  if (empty) empty.hidden = visible > 0;
}

export function initCatalogPage(): void {
  const catalogContent = document.getElementById("catalog-content");
  if (!catalogContent || !catalogContent.closest(".catalog-page")) return;

  currentPage = 1;
  totalPages = parseInt(catalogContent.dataset.totalPages || "1", 10);
  currentSort = catalogContent.dataset.sort || "popular";

  console.info(`Catalog page initialized: ${totalPages} total pages`);

  const sortDropdown = document.getElementById("catalog-sort-dropdown");
  if (sortDropdown) {
    sortDropdown.addEventListener("change", (e: Event) => {
      const customEvent = e as CustomEvent<{ value: string }>;
      currentSort = customEvent.detail.value;
      if (currentSort !== "relevance") {
        setKappalibCookie("catalog_sort", currentSort);
      }
      reloadCatalog();
    });
  }

  initCatalogFilters();

  loaderObserver = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting && !isLoading && currentPage < totalPages) {
          loadMoreNovels();
        }
      });
    },
    {
      rootMargin: "200px",
    },
  );
  observeLoader();

  window.addEventListener("popstate", () => {
    window.location.reload();
  });
}
