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
  const header = document.getElementById("main-header");

  if (!input || !results) return;

  let timeout: number | undefined;
  const API_URL = process.env.API_URL;

  const PLACEHOLDER_IMG =
    "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='200' height='300'%3E%3Crect fill='%23ecf0f1' width='200' height='300'/%3E%3C/svg%3E";

  input.onfocus = () => {
    if (window.innerWidth <= 600 && header) {
      header.classList.add("search-expanded");
    }
  };

  input.onblur = () => {
    setTimeout(() => {
      const focusMovedToResults = results.contains(document.activeElement);

      if (!focusMovedToResults) {
        if (header) header.classList.remove("search-expanded");
        input.value = "";
        results.style.display = "none";
      }
    }, 150);
  };

  input.oninput = (e: Event) => {
    const target = e.target as HTMLInputElement;
    const query = target.value.trim();
    clearTimeout(timeout);

    if (query.length < 2) {
      results.style.display = "none";
      return;
    }

    timeout = window.setTimeout(async () => {
      try {
        console.info(`Searching for: "${query}"`);
        results.style.display = "block";

        results.innerHTML = "";
        const loadingDiv = document.createElement("div");
        loadingDiv.className = "search-loading";
        loadingDiv.textContent = "Поиск...";
        results.appendChild(loadingDiv);

        const res = await fetch(
          `${API_URL}/novels/search?q=${encodeURIComponent(query)}`,
        );
        const data: SearchResponse = await res.json();

        results.innerHTML = "";

        if (!data.novels || data.novels.length === 0) {
          console.info(`No results found for: "${query}"`);
          const noResultsDiv = document.createElement("div");
          noResultsDiv.className = "no-results";
          noResultsDiv.textContent = "Ничего не найдено";
          results.appendChild(noResultsDiv);
          return;
        }

        console.info(`Found ${data.novels.length} results for: "${query}"`);

        const fragment = document.createDocumentFragment();

        data.novels.forEach((novel) => {
          const a = document.createElement("a");
          a.href = `/${novel.id}`;
          a.className = "search-result-card";

          const img = document.createElement("img");
          img.src = novel.cover_url || PLACEHOLDER_IMG;
          img.alt = novel.title;

          const infoDiv = document.createElement("div");
          infoDiv.className = "search-result-info";

          const h3 = document.createElement("h3");
          h3.textContent = novel.title;

          const metaDiv = document.createElement("div");
          metaDiv.className = "search-result-meta";

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
          } else {
            const authorP = document.createElement("p");
            authorP.className = "author";
            authorP.textContent = novel.author;
            infoDiv.appendChild(authorP);
          }

          a.appendChild(img);
          a.appendChild(infoDiv);
          fragment.appendChild(a);
        });

        results.appendChild(fragment);
      } catch (err) {
        console.error("Search API request failed", err);
        results.innerHTML = "";
        const errorDiv = document.createElement("div");
        errorDiv.className = "no-results";
        errorDiv.textContent = "Ошибка поиска";
        results.appendChild(errorDiv);
      }
    }, 350);
  };

  document.addEventListener("click", (e: Event) => {
    if (
      !input.contains(e.target as Node) &&
      !results.contains(e.target as Node)
    ) {
      results.style.display = "none";
      input.value = "";
      if (header) {
        header.classList.remove("search-expanded");
      }
    }
  });
}

function mapStatus(status: string): string {
  const statusMap: Record<string, string> = {
    ongoing: "Выходит",
    completed: "Завершено",
    hiatus: "Приостановлено",
    dropped: "Заброшено",
  };
  return statusMap[status] || status;
}
