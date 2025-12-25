export async function loadCatalogPage(
  url: string,
  isHistoryPush: boolean = true,
): Promise<void> {
  const container = document.getElementById("catalog-content");
  if (!container) return;

  const urlObj = new URL(url, window.location.origin);
  if (urlObj.searchParams.get("page") === "1") {
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
