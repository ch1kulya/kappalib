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

      const titleElement =
        document.getElementById("catalog-title") ||
        document.querySelector("h2");

      if (titleElement) {
        const header = document.getElementById("main-header");
        const headerHeight = header ? header.offsetHeight : 0;
        const extraMargin = 24;
        const elementPosition = titleElement.getBoundingClientRect().top;
        const offsetPosition =
          elementPosition + window.scrollY - headerHeight - extraMargin;

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
      !link ||
      !link.href ||
      link.classList.contains("active") ||
      link.classList.contains("disabled")
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

function getBaseParams(): URLSearchParams {
  const params = new URLSearchParams();

  params.set("sort", currentSort);

  const catalogContent = document.getElementById("catalog-content");
  const search = catalogContent?.dataset.search;
  if (search) {
    params.set("search", search);
  }

  return params;
}

async function loadMoreNovels(): Promise<void> {
  if (isLoading || currentPage >= totalPages) return;

  isLoading = true;
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
  }
}

async function applySort(): Promise<void> {
  const params = getBaseParams();
  params.set("page", "1");

  if (currentSort !== "relevance") {
    setKappalibCookie("catalog_sort", currentSort);
  }

  const catalogContent = document.getElementById("catalog-content");
  if (catalogContent) {
    catalogContent.classList.add("is-loading");
  }

  try {
    const response = await fetch(`/catalog?${params.toString()}`, {
      headers: { "X-Partial": "true" },
    });

    if (!response.ok) throw new Error(`HTTP error ${response.status}`);

    const html = await response.text();
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, "text/html");
    const newGrid = doc.getElementById("catalog-grid");

    if (newGrid && catalogContent) {
      const currentGrid = document.getElementById("catalog-grid");
      if (currentGrid) {
        currentGrid.innerHTML = newGrid.innerHTML;
      }

      const tempDiv = document.createElement("div");
      tempDiv.innerHTML = html;
      const emptyMessage = tempDiv.querySelector(".catalog-empty");
      const existingEmpty = catalogContent.querySelector(".catalog-empty");

      if (emptyMessage && !existingEmpty) {
        catalogContent.appendChild(emptyMessage.cloneNode(true));
      } else if (!emptyMessage && existingEmpty) {
        existingEmpty.remove();
      }
    }

    currentPage = 1;
    const newCatalogContent = doc.getElementById("catalog-content");
    if (newCatalogContent) {
      totalPages = parseInt(newCatalogContent.dataset.totalPages || "1", 10);
      if (catalogContent) {
        catalogContent.dataset.totalPages = String(totalPages);
      }
    }

    updateURL();

    const loader = document.getElementById("catalog-loader");
    if (loader) {
      loader.style.display = currentPage < totalPages ? "flex" : "none";
    }

    console.info("Sort applied successfully");
  } catch (err) {
    console.error("Failed to apply sort", err);
  } finally {
    if (catalogContent) {
      catalogContent.classList.remove("is-loading");
    }
  }
}

function updateURL(): void {
  const params = getBaseParams();

  const newUrl = new URL(window.location.origin + "/catalog");
  params.forEach((value, key) => {
    newUrl.searchParams.append(key, value);
  });
  window.history.replaceState({}, "", newUrl.toString());
}

export function initCatalogPage(): void {
  const catalogContent = document.getElementById("catalog-content");
  if (!catalogContent) return;

  const loader = document.getElementById("catalog-loader");
  if (!loader && !catalogContent.dataset.search) return;

  currentPage = 1;
  totalPages = parseInt(catalogContent.dataset.totalPages || "1", 10);
  currentSort = catalogContent.dataset.sort || "popular";

  console.info(`Catalog page initialized: ${totalPages} total pages`);

  const sortDropdown = document.getElementById("catalog-sort-dropdown");
  if (sortDropdown) {
    sortDropdown.addEventListener("change", (e: Event) => {
      const customEvent = e as CustomEvent<{ value: string }>;
      currentSort = customEvent.detail.value;
      applySort();
    });
  }

  if (loader) {
    const observer = new IntersectionObserver(
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

    observer.observe(loader);
  }

  window.addEventListener("popstate", () => {
    window.location.reload();
  });
}
