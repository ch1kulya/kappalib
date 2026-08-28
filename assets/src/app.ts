import { initAgeGate } from "./modules/age";
import { initCatalogPage, initCatalogPagination } from "./modules/catalog";
import { initComments, initMyCommentsPage } from "./modules/comments";
import { initDescription } from "./modules/description";
import Dropdown from "./modules/dropdown";
import { initHistoryPage } from "./modules/history";
import { initProfile, initProfileModal } from "./modules/profile";
import { initReadingProgressSaver, refreshLastReadTotalChapters } from "./modules/progress";
import { initSearch } from "./modules/search";
import { initSettings, initSettingsModal } from "./modules/settings";
import { initCatalogSort, initChaptersSort } from "./modules/sort";
import { initStatusBadge } from "./modules/status";
import { initTimeTracker } from "./modules/time";
import { initTocFilter } from "./modules/toc";

declare global {
  interface Window {
    isAdultContent?: boolean;
  }
}

document.addEventListener("DOMContentLoaded", () => {
  try {
    document
      .querySelectorAll<HTMLLinkElement>("link.lazy-font")
      .forEach((link) => {
        link.media = "all";
      });

    const retryBtn = document.getElementById("retry-btn");
    if (retryBtn) {
      retryBtn.addEventListener("click", () => window.location.reload());
    }

    const dropdowns = document.querySelectorAll<HTMLElement>(".dropdown");
    dropdowns.forEach((el) => new Dropdown(el));

    initSettings();
    initProfile();
    initProfileModal();
    initSettingsModal();
    initChaptersSort();
    initCatalogSort();
    initTocFilter();
    initSearch();
    initAgeGate();
    initDescription();
    initReadingProgressSaver();
    initStatusBadge();
    initCatalogPagination();
    initCatalogPage();
    initComments();
    initHistoryPage();
    initMyCommentsPage();
    initTimeTracker();

    if (document.querySelector(".cr-wrapper")) {
      refreshLastReadTotalChapters();
    }

    document.querySelector(".markdown-help")?.addEventListener("click", (e) => {
      const target = e.target as HTMLElement;
      if (target.classList.contains("spoiler")) {
        target.classList.toggle("revealed");
      }
    });

    console.info("All modules initialized successfully");
  } catch (err) {
    console.error("Critical error during module initialization", err);
  }
});
