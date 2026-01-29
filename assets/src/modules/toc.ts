export function initTocFilter(): void {
  const input = document.getElementById(
    "toc-search-input",
  ) as HTMLInputElement | null;
  const chaptersList = document.getElementById("chapters-list");

  if (!input || !chaptersList) return;

  const chapters = chaptersList.querySelectorAll<HTMLAnchorElement>(
    ".chapter-item[data-chapter-id]",
  );
  if (chapters.length === 0) return;

  const originalData = Array.from(chapters).map((el) => ({
    element: el,
    numEl: el.querySelector(".chapter-num") as HTMLElement,
    titleEl: el.querySelector(".chapter-title") as HTMLElement,
    numText: el.querySelector(".chapter-num")?.textContent || "",
    titleText: el.querySelector(".chapter-title")?.textContent || "",
  }));

  let noResultsEl: HTMLElement | null = null;
  let debounceTimer: ReturnType<typeof setTimeout>;

  input.addEventListener("input", () => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => filterChapters(input.value), 80);
  });

  function filterChapters(query: string): void {
    const q = query.trim().toLowerCase();

    if (!q) {
      originalData.forEach((item) => {
        item.element.classList.remove("toc-hidden");
        item.numEl.textContent = item.numText;
        item.titleEl.innerHTML = escapeHtml(item.titleText);
      });
      hideNoResults();
      return;
    }

    let matchCount = 0;
    originalData.forEach((item) => {
      const numMatch = item.numText.toLowerCase().includes(q);
      const titleMatch = item.titleText.toLowerCase().includes(q);

      if (numMatch || titleMatch) {
        item.element.classList.remove("toc-hidden");
        item.numEl.innerHTML = highlightMatch(item.numText, q);
        item.titleEl.innerHTML = highlightMatch(item.titleText, q);
        matchCount++;
      } else {
        item.element.classList.add("toc-hidden");
        item.numEl.textContent = item.numText;
        item.titleEl.innerHTML = escapeHtml(item.titleText);
      }
    });

    if (matchCount === 0) {
      showNoResults();
    } else {
      hideNoResults();
    }
  }

  function showNoResults(): void {
    if (noResultsEl || !chaptersList) return;
    noResultsEl = document.createElement("div");
    noResultsEl.className = "no-results toc-no-results";
    noResultsEl.textContent = "Ничего не найдено";
    chaptersList.appendChild(noResultsEl);
  }

  function hideNoResults(): void {
    if (noResultsEl) {
      noResultsEl.remove();
      noResultsEl = null;
    }
  }

  function highlightMatch(text: string, query: string): string {
    const escaped = escapeHtml(text);
    const queryEscaped = escapeRegex(query);
    const regex = new RegExp(`(${queryEscaped})`, "gi");
    return escaped.replace(regex, "<mark>$1</mark>");
  }

  function escapeHtml(str: string): string {
    return str
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function escapeRegex(str: string): string {
    return str.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  }
}
