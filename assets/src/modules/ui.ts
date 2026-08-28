type UIState = "idle" | "profile" | "settings" | "search" | "bookmark";

class UIManager {
  private state: UIState = "idle";
  private backdrop = document.getElementById("header-backdrop");
  private profileCard = document.getElementById("profile-card");
  private settingsCard = document.getElementById("settings-card");
  private bookmarkCard = document.getElementById("bookmark-card"); // добавь
  private searchResults = document.getElementById("search-results");
  private searchInput = document.getElementById(
    "search-input",
  ) as HTMLInputElement | null;
  private profileBtn = document.getElementById("header-profile-btn");
  private settingsBtn = document.getElementById("header-settings-btn");
  private bookmarkBtn = document.getElementById("header-bookmark-btn"); // добавь
  private header = document.getElementById("main-header");

  constructor() {
    this.initBackdrop();
    this.initGlobalKeys();
  }

  public toggleBookmark(): void {
    if (this.state === "bookmark") {
      this.closeAll();
    } else {
      this.activateState("bookmark");
    }
  }

  private initBackdrop() {
    this.backdrop?.addEventListener("click", () => this.closeAll());
  }

  private initGlobalKeys() {
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape" && this.state !== "idle") {
        this.closeAll();
      }
    });
  }

  public toggleProfile(onOpen?: () => void): void {
    if (this.state === "profile") {
      this.closeAll();
    } else {
      this.activateState("profile");
      if (onOpen) onOpen();
    }
  }

  public toggleSettings(): void {
    if (this.state === "settings") {
      this.closeAll();
    } else {
      this.activateState("settings");
    }
  }

  public openSearch(): void {
    if (this.state !== "search") {
      this.activateState("search");
    }
  }

  public closeAll(): void {
    if (this.state === "idle") return;
    this.state = "idle";
    if (this.searchInput) this.searchInput.value = "";
    this.updateDOM();
  }

  public isSearchActive(): boolean {
    return this.state === "search";
  }

  private activateState(newState: UIState): void {
    this.state = newState;
    this.updateDOM();
  }

  private updateDOM(): void {
    const isIdle = this.state === "idle";

    if (this.backdrop) {
      this.backdrop.classList.toggle("active", !isIdle);
    }
    document.body.style.overflow = isIdle ? "" : "hidden";

    if (this.profileCard) {
      this.profileCard.style.display =
        this.state === "profile" ? "block" : "none";
    }
    if (this.profileBtn) {
      this.profileBtn.classList.toggle("active", this.state === "profile");
    }

    if (this.settingsCard) {
      this.settingsCard.style.display =
        this.state === "settings" ? "block" : "none";
    }
    if (this.settingsBtn) {
      this.settingsBtn.classList.toggle("active", this.state === "settings");
    }
    if (this.bookmarkCard) {
      this.bookmarkCard.style.display =
        this.state === "bookmark" ? "block" : "none";
    }
    if (this.bookmarkBtn) {
      this.bookmarkBtn.classList.toggle("active", this.state === "bookmark");
    }

    if (this.searchResults) {
      if (this.state !== "search") {
        this.searchResults.style.display = "none";
        this.searchResults.innerHTML = "";
      } else {
        this.searchResults.style.display = "block";
      }
    }

    if (this.header) {
      if (this.state === "search") {
        this.header.classList.add("search-expanded");
      } else {
        this.header.classList.remove("search-expanded");
      }
    }
  }
}

export const uiManager = new UIManager();
