import { loadCatalogPage } from "./catalog";
import { setKappalibCookie } from "./profile";
import { trackEvent } from "./analytics";

interface DropdownChangeEventDetail {
  value: string;
}

let sortingInProgress = false;

export function initChaptersSort(): void {
  const sortDropdown = document.getElementById("chapter-sort");
  const chaptersList = document.getElementById("chapters-list");

  if (sortDropdown && chaptersList) {
    sortDropdown.addEventListener("change", async (e: Event) => {
      if (sortingInProgress) {
        console.warn("Sort already in progress");
        return;
      }

      const customEvent = e as CustomEvent<DropdownChangeEventDetail>;
      const value = customEvent.detail.value;

      console.info(`Sort requested: ${value}`);

      sortingInProgress = true;
      setKappalibCookie("chapter_sort", value);
      trackEvent("chapter_sort", { sort: value });

      try {
        chaptersList.classList.add("is-sorting");
        chaptersList.style.pointerEvents = "none";

        await new Promise((resolve) => setTimeout(resolve, 50));

        await new Promise<void>((resolve) => {
          setTimeout(() => {
            requestAnimationFrame(() => {
              sortChapters(chaptersList, value);
              resolve();
            });
          }, 0);
        });

        await new Promise((resolve) => setTimeout(resolve, 100));

        console.info("Sort completed successfully");
      } catch (error) {
        console.error("Sort failed:", error);
      } finally {
        chaptersList.classList.remove("is-sorting");
        chaptersList.style.pointerEvents = "";
        setTimeout(() => {
          sortingInProgress = false;
        }, 150);
      }
    });
  }
}

export function initCatalogSort(): void {
  const sortDropdown = document.getElementById("catalog-sort");

  if (sortDropdown) {
    sortDropdown.addEventListener("change", (e: Event) => {
      const customEvent = e as CustomEvent<DropdownChangeEventDetail>;
      const value = customEvent.detail.value;

      console.info(`Changed catalog sort to: ${value}`);
      setKappalibCookie("catalog_sort", value);
      trackEvent("catalog_sort", { sort: value });
      loadCatalogPage(window.location.href, false);
    });
  }
}

function sortChapters(container: HTMLElement, direction: string): void {
  try {
    const items = Array.from(
      container.querySelectorAll<HTMLElement>(".chapter-item"),
    );

    if (items.length === 0) {
      console.warn("No chapter items found");
      return;
    }

    console.info(`Sorting ${items.length} chapters: ${direction}`);

    const itemsData = items.map((item) => {
      const sortAttr = item.getAttribute("data-sort-value");
      const value = sortAttr ? parseFloat(sortAttr) : 0;

      return {
        element: item,
        value: value,
      };
    });

    itemsData.sort((a, b) => {
      return direction === "asc" ? a.value - b.value : b.value - a.value;
    });

    const fragment = document.createDocumentFragment();
    itemsData.forEach((item) => {
      fragment.appendChild(item.element);
    });
    container.appendChild(fragment);

    void container.offsetHeight;
    console.info("Chapters reordered successfully");
  } catch (error) {
    console.error("Critical error in sortChapters:", error);
    throw error;
  }
}
