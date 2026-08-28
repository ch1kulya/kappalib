import { initSearch } from "./modules/search";
import { initAgeGate } from "./modules/age";
import {
  initReadingProgressSaver,
  refreshLastReadTotalChapters,
} from "./modules/progress";
import { initStatusBadge } from "./modules/status";
import { initCatalogPagination, initCatalogPage } from "./modules/catalog";
import Dropdown from "./modules/dropdown";
import { initChaptersSort, initCatalogSort } from "./modules/sort";
import { initTocFilter } from "./modules/toc";
import { initDescription } from "./modules/description";
import { initProfile, initProfileModal } from "./modules/profile";
import { initComments, initMyCommentsPage } from "./modules/comments";
import { initSettings, initSettingsModal } from "./modules/settings";
import { initHistoryPage } from "./modules/history";
import { initTimeTracker } from "./modules/time";
import { initBookmarkButton, initBookmarksPage } from "./modules/bookmarks";

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
    initBookmarkButton();
    initBookmarksPage();

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
