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
    titleHtml: el.querySelector(".chapter-title")?.innerHTML || "",
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
        item.element.classList.remove("toc-hidden", "toc-last-visible");
        item.numEl.textContent = item.numText;
        item.titleEl.innerHTML = item.titleHtml;
      });
      hideNoResults();
      return;
    }

    let lastVisible: HTMLAnchorElement | null = null;
    originalData.forEach((item) => {
      item.element.classList.remove("toc-last-visible");
      const numMatch = item.numText.toLowerCase().includes(q);
      const titleMatch = item.titleText.toLowerCase().includes(q);

      if (numMatch || titleMatch) {
        item.element.classList.remove("toc-hidden");
        item.numEl.innerHTML = highlight(item.numText, q);
        item.titleEl.innerHTML = highlight(item.titleHtml, q);
        lastVisible = item.element;
      } else {
        item.element.classList.add("toc-hidden");
        item.numEl.textContent = item.numText;
        item.titleEl.innerHTML = item.titleHtml;
      }
    });

    if (lastVisible) {
      hideNoResults();
      (lastVisible as HTMLAnchorElement).classList.add("toc-last-visible");
    } else {
      showNoResults();
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

  function highlight(html: string, query: string): string {
    const queryEscaped = escapeRegex(query);
    const regex = new RegExp(`(${queryEscaped})`, "gi");
    const parts = html.split(/(<[^>]*>)/);
    return parts
      .map((part) => {
        if (part.startsWith("<")) return part;
        return part.replace(regex, "<mark>$1</mark>");
      })
      .join("");
  }

  function escapeRegex(str: string): string {
    return str.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  }
}
