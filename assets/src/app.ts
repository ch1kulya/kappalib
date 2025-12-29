import { initSearch } from "./modules/search";
import { initAgeGate } from "./modules/age";
import { initReadingProgressSaver } from "./modules/progress";
import { initStatusBadge } from "./modules/status";
import { initCatalogPagination } from "./modules/catalog";
import Dropdown from "./modules/dropdown";
import { initChaptersSort, initCatalogSort } from "./modules/sort";
import { initDescription } from "./modules/description";
import { initProfile, initProfileModal } from "./modules/profile";
import { initComments } from "./modules/comments";

declare global {
  interface Window {
    isAdultContent?: boolean;
  }
}

document.addEventListener("DOMContentLoaded", () => {
  try {
    const dropdowns = document.querySelectorAll<HTMLElement>(".dropdown");
    dropdowns.forEach((el) => new Dropdown(el));

    initProfile();
    initProfileModal();
    initChaptersSort();
    initCatalogSort();
    initSearch();
    initAgeGate();
    initDescription();
    initReadingProgressSaver();
    initStatusBadge();
    initCatalogPagination();
    initComments();

    console.info("All modules initialized successfully");
  } catch (err) {
    console.error("Critical error during module initialization", err);
  }
});
