import { trackEvent } from "./analytics";
import { uiManager } from "./ui";

interface NovelSearchResult {
  id: string;
  title: string;
  author: string;
  cover_url?: string;
  year_start?: number;
  status?: string;
  description?: string;
}

interface SearchResponse {
  novels: NovelSearchResult[];
}

export function initSearch(): void {
  const input = document.getElementById(
    "search-input",
  ) as HTMLInputElement | null;
  const results = document.getElementById("search-results");

  if (!input || !results) return;

  let timeout: number | undefined;
  let firstResultUrl: string | null = null;
  const API_URL = process.env.API_URL;

  const PLACEHOLDER_IMG =
    "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='200' height='300'%3E%3Crect fill='%23ecf0f1' width='200' height='300'/%3E%3C/svg%3E";

  const clearResults = () => {
    results.innerHTML = "";
    firstResultUrl = null;
  };

  input.onfocus = () => {
    uiManager.openSearch();
  };

  input.onkeydown = (e: KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      const query = input.value.trim();
      if (query.length >= 2) {
        trackEvent("search", { query });
        window.location.href = `/catalog?search=${encodeURIComponent(query)}`;
      } else if (firstResultUrl) {
        window.location.href = firstResultUrl;
      }
    }
  };

  input.oninput = (e: Event) => {
    const target = e.target as HTMLInputElement;
    const query = target.value.trim();
    clearTimeout(timeout);

    if (query.length < 2) {
      results.style.display = "none";
      firstResultUrl = null;
      clearResults();
      return;
    }

    uiManager.openSearch();

    timeout = window.setTimeout(async () => {
      try {
        console.info(`Searching for: "${query}"`);
        results.style.display = "block";

        let loadingTimeout: number | null = setTimeout(() => {
          if (!uiManager.isSearchActive()) return;
          loadingTimeout = null;
          results.innerHTML = "";
          const loadingDiv = document.createElement("div");
          loadingDiv.className = "search-loading";
          loadingDiv.textContent = "Поиск...";
          results.appendChild(loadingDiv);
        }, 150);

        const res = await fetch(
          `${API_URL}/novels/search?q=${encodeURIComponent(query)}`,
        );
        const data: SearchResponse = await res.json();

        if (loadingTimeout !== null) {
          clearTimeout(loadingTimeout);
        }

        results.innerHTML = "";
        firstResultUrl = null;

        trackEvent("search", { query });

        if (!data.novels || data.novels.length === 0) {
          console.info(`No results found for: "${query}"`);
          const noResultsDiv = document.createElement("div");
          noResultsDiv.className = "no-results";
          noResultsDiv.textContent = "Ничего не найдено";
          results.appendChild(noResultsDiv);
          return;
        }

        console.info(`Found ${data.novels.length} results for: "${query}"`);

        firstResultUrl = `/${data.novels[0].id}`;

        const fragment = document.createDocumentFragment();

        data.novels.forEach((novel) => {
          const a = document.createElement("a");
          a.href = `/${novel.id}`;
          a.className = "search-result-card";
          a.addEventListener("click", () => {
            trackEvent("search_select", { novel_id: novel.id });
          });

          const img = document.createElement("img");
          img.src = novel.cover_url || PLACEHOLDER_IMG;
          img.alt = novel.title;

          const infoDiv = document.createElement("div");
          infoDiv.className = "search-result-info";

          const h3 = document.createElement("h3");
          h3.textContent = novel.title;

          const metaDiv = document.createElement("div");
          metaDiv.className = "search-result-meta";

          if (novel.author) {
            const authorBadge = document.createElement("span");
            authorBadge.className = "badge";
            authorBadge.textContent = novel.author.toString();
            metaDiv.appendChild(authorBadge);
          }

          if (novel.year_start) {
            const yearBadge = document.createElement("span");
            yearBadge.className = "badge";
            yearBadge.textContent = novel.year_start.toString();
            metaDiv.appendChild(yearBadge);
          }

          if (novel.status) {
            const statusBadge = document.createElement("span");
            statusBadge.className = "badge";
            statusBadge.textContent = mapStatus(novel.status);
            metaDiv.appendChild(statusBadge);
          }

          infoDiv.appendChild(h3);
          infoDiv.appendChild(metaDiv);

          if (novel.description) {
            const descP = document.createElement("p");
            descP.className = "search-result-desc";
            descP.textContent = novel.description;
            infoDiv.appendChild(descP);
          }

          a.appendChild(img);
          a.appendChild(infoDiv);
          fragment.appendChild(a);
        });

        results.appendChild(fragment);
      } catch (err) {
        console.error("Search API request failed", err);
        if (!uiManager.isSearchActive()) return;
        results.innerHTML = "";
        firstResultUrl = null;
        const errorDiv = document.createElement("div");
        errorDiv.className = "no-results";
        errorDiv.textContent = "Ошибка поиска";
        results.appendChild(errorDiv);
      }
    }, 350);
  };

  document.addEventListener("click", (e: Event) => {
    if (!uiManager.isSearchActive()) return;
    if (
      !input.contains(e.target as Node)
      && !results.contains(e.target as Node)
    ) {
      uiManager.closeAll();
      input.value = "";
      firstResultUrl = null;
      clearResults();
    }
  });
}

function mapStatus(status: string): string {
  const statusMap: Record<string, string> = {
    ongoing: "Онгоинг",
    completed: "Завершено",
    announced: "Анонс",
  };
  return statusMap[status] || status;
}
