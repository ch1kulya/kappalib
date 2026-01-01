import Dropdown from "./dropdown";

const SETTINGS_COOKIE_KEY = "kappalib_reader_settings";
const SETTINGS_TIMESTAMP_KEY = "kappalib_reader_settings_updated_at";

interface ReaderSettings {
  theme: "auto" | "light" | "dark";
  fontSize: number;
  fontFamily: string;
  indent: number;
  density: "compact" | "normal" | "relaxed";
  justify: boolean;
}

const DEFAULT_SETTINGS: ReaderSettings = {
  theme: "auto",
  fontSize: 18,
  fontFamily: "default",
  indent: 0,
  density: "normal",
  justify: false,
};

const FONT_OPTIONS: { value: string; label: string; family: string }[] = [
  { value: "default", label: "Стандартный", family: "inherit" },
  { value: "literata", label: "Literata", family: "'Literata', serif" },
  {
    value: "source-serif",
    label: "Source Serif",
    family: "'Source Serif 4', serif",
  },
  {
    value: "merriweather",
    label: "Merriweather",
    family: "'Merriweather', serif",
  },
  { value: "lora", label: "Lora", family: "'Lora', serif" },
  { value: "pt-serif", label: "PT Serif", family: "'PT Serif', serif" },
  { value: "open-sans", label: "Open Sans", family: "'Open Sans', sans-serif" },
  { value: "roboto", label: "Roboto", family: "'Roboto', sans-serif" },
];

const FONT_URLS: Record<string, string> = {
  literata: "https://cdn.jsdelivr.net/npm/@fontsource/literata@5/index.min.css",
  "source-serif":
    "https://cdn.jsdelivr.net/npm/@fontsource/source-serif-4@5/index.min.css",
  merriweather:
    "https://cdn.jsdelivr.net/npm/@fontsource/merriweather@5/index.min.css",
  lora: "https://cdn.jsdelivr.net/npm/@fontsource/lora@5/index.min.css",
  "pt-serif":
    "https://cdn.jsdelivr.net/npm/@fontsource/pt-serif@5/index.min.css",
  "open-sans":
    "https://cdn.jsdelivr.net/npm/@fontsource/open-sans@5/index.min.css",
  roboto: "https://cdn.jsdelivr.net/npm/@fontsource/roboto@5/index.min.css",
};

const loadedFonts = new Set<string>();

function loadFont(fontKey: string): void {
  if (fontKey === "default" || loadedFonts.has(fontKey)) return;

  const url = FONT_URLS[fontKey];
  if (!url) return;

  const link = document.createElement("link");
  link.rel = "stylesheet";
  link.href = url;
  link.media = "print";
  link.onload = () => {
    link.media = "all";
  };
  document.head.appendChild(link);
  loadedFonts.add(fontKey);
}

function getCookie(name: string): string | null {
  const match = document.cookie.match(new RegExp("(^| )" + name + "=([^;]+)"));
  return match ? decodeURIComponent(match[2]) : null;
}

function setCookie(name: string, value: string): void {
  const timestamp = Date.now();
  document.cookie = `${name}=${encodeURIComponent(value)}; path=/; max-age=31536000; SameSite=Lax`;
  localStorage.setItem(SETTINGS_TIMESTAMP_KEY, timestamp.toString());
}

function getSettings(): ReaderSettings {
  try {
    const raw = getCookie(SETTINGS_COOKIE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw);
      return { ...DEFAULT_SETTINGS, ...parsed };
    }
  } catch {
    // ignore
  }
  return { ...DEFAULT_SETTINGS };
}

function saveSettings(settings: ReaderSettings): void {
  setCookie(SETTINGS_COOKIE_KEY, JSON.stringify(settings));
  applySettings(settings);
}

function enableThemeTransition(): void {
  const root = document.documentElement;
  root.classList.add("theme-transitioning");

  const handleTransitionEnd = () => {
    root.classList.remove("theme-transitioning");
    root.removeEventListener("transitionend", handleTransitionEnd);
  };

  root.addEventListener("transitionend", handleTransitionEnd);
  setTimeout(() => {
    root.classList.remove("theme-transitioning");
  }, 100);
}

function applySettings(settings: ReaderSettings): void {
  const root = document.documentElement;

  if (settings.theme === "auto") {
    root.removeAttribute("data-theme");
  } else {
    root.setAttribute("data-theme", settings.theme);
  }

  const chapterContent = document.querySelector(
    ".chapter-content",
  ) as HTMLElement | null;
  const chapterTitle = document.querySelector(
    ".chapter-reader > h1",
  ) as HTMLElement | null;

  const baseFontSize = settings.fontSize;
  const titleRatio = 1.5 / 1.125;
  const titleFontSize = baseFontSize * titleRatio;

  const baseMarginRem = 2;
  const marginRatio = baseFontSize / 18;
  const titleMargin = baseMarginRem * marginRatio;

  if (chapterContent) {
    chapterContent.style.fontSize = `${baseFontSize / 16}rem`;

    const fontOption = FONT_OPTIONS.find(
      (f) => f.value === settings.fontFamily,
    );
    if (fontOption) {
      loadFont(settings.fontFamily);
      chapterContent.style.fontFamily = fontOption.family;
    }

    chapterContent.style.setProperty(
      "--reader-indent",
      settings.indent > 0 ? `${settings.indent}em` : "0",
    );
    chapterContent.classList.remove(
      "density-compact",
      "density-normal",
      "density-relaxed",
    );
    chapterContent.classList.add(`density-${settings.density}`);
    chapterContent.classList.toggle("justify-text", settings.justify);
  }

  if (chapterTitle) {
    chapterTitle.style.fontSize = `${titleFontSize / 16}rem`;
    chapterTitle.style.marginBottom = `${titleMargin}rem`;

    const fontOption = FONT_OPTIONS.find(
      (f) => f.value === settings.fontFamily,
    );
    if (fontOption) {
      loadFont(settings.fontFamily);
      chapterTitle.style.fontFamily = fontOption.family;
    }

    chapterTitle.classList.toggle("justify-text", settings.justify);
  }
}

function applyGlobalSettings(): void {
  const settings = getSettings();
  const root = document.documentElement;

  if (settings.theme === "auto") {
    root.removeAttribute("data-theme");
  } else {
    root.setAttribute("data-theme", settings.theme);
  }
}

class SettingsManager {
  private settings: ReaderSettings;

  constructor() {
    this.settings = getSettings();
  }

  getSettings(): ReaderSettings {
    return { ...this.settings };
  }

  updateSetting<K extends keyof ReaderSettings>(
    key: K,
    value: ReaderSettings[K],
  ): void {
    this.settings[key] = value;
    saveSettings(this.settings);
  }

  applyAll(): void {
    applySettings(this.settings);
  }
}

export const settingsManager = new SettingsManager();

export function initSettings(): void {
  applyGlobalSettings();
}

export function initSettingsModal(): void {
  const settingsCard = document.getElementById("settings-card");
  const settingsBtn = document.getElementById("header-settings-btn");
  const backdrop = document.getElementById("header-backdrop");
  const profileCard = document.getElementById("profile-card");

  if (!settingsCard || !settingsBtn) return;

  settingsBtn.addEventListener("click", (e) => {
    e.preventDefault();
    e.stopPropagation();

    if (profileCard?.style.display === "block") {
      profileCard.style.display = "none";
    }

    if (settingsCard.style.display === "block") {
      closeSettingsCard();
    } else {
      openSettingsCard();
    }
  });

  backdrop?.addEventListener("click", () => {
    closeSettingsCard();
  });

  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && settingsCard.style.display === "block") {
      closeSettingsCard();
    }
  });

  document.addEventListener("click", (e) => {
    if (
      settingsCard.style.display === "block" &&
      !settingsCard.contains(e.target as Node) &&
      !settingsBtn.contains(e.target as Node)
    ) {
      closeSettingsCard();
    }
  });

  settingsManager.applyAll();
}

function openSettingsCard(): void {
  const settingsCard = document.getElementById("settings-card");
  const backdrop = document.getElementById("header-backdrop");
  if (!settingsCard) return;

  settingsCard.style.display = "block";
  backdrop?.classList.add("active");
  document.body.style.overflow = "hidden";

  renderSettingsView();
}

function closeSettingsCard(): void {
  const settingsCard = document.getElementById("settings-card");
  const backdrop = document.getElementById("header-backdrop");
  const profileCard = document.getElementById("profile-card");
  if (!settingsCard) return;

  settingsCard.style.display = "none";

  if (profileCard?.style.display !== "block") {
    backdrop?.classList.remove("active");
    document.body.style.overflow = "";
  }
}

function updateFontSizeButtons(fontSize: number): void {
  const fontDecrease = document.getElementById(
    "font-decrease",
  ) as HTMLButtonElement | null;
  const fontIncrease = document.getElementById(
    "font-increase",
  ) as HTMLButtonElement | null;

  if (fontDecrease) {
    fontDecrease.disabled = fontSize <= 14;
  }
  if (fontIncrease) {
    fontIncrease.disabled = fontSize >= 26;
  }
}

function renderSettingsView(): void {
  const content = document.getElementById("settings-card");
  if (!content) return;

  const settings = settingsManager.getSettings();
  const currentFont =
    FONT_OPTIONS.find((f) => f.value === settings.fontFamily) ||
    FONT_OPTIONS[0];

  content.innerHTML = `
    <div class="settings-body">
      <div class="settings-row settings-row-font">
        <div class="settings-col settings-col-font">
          <div class="settings-label">Шрифт</div>
          <div class="dropdown settings-dropdown" id="font-dropdown">
            <button class="dropdown-btn" type="button" aria-haspopup="listbox" aria-expanded="false">
              <span class="js-dropdown-label">${currentFont.label}</span>
              <svg class="chevron" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>
            </button>
            <div class="dropdown-menu" role="listbox">
              <div class="dropdown-menu-inner">
                ${FONT_OPTIONS.map(
                  (f) => `
                  <button class="dropdown-item${settings.fontFamily === f.value ? " selected" : ""}" data-value="${f.value}" role="option" aria-selected="${settings.fontFamily === f.value}">
                    <span>${f.label}</span>
                    <svg class="check-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
                  </button>
                `,
                ).join("")}
              </div>
            </div>
          </div>
        </div>
        <div class="settings-col settings-col-size">
          <div class="settings-font-size">
            <button class="settings-font-btn" id="font-decrease"${settings.fontSize <= 14 ? " disabled" : ""}>−</button>
            <span class="settings-font-value" id="font-size-value">${settings.fontSize}</span>
            <button class="settings-font-btn" id="font-increase"${settings.fontSize >= 26 ? " disabled" : ""}>+</button>
          </div>
        </div>
      </div>

      <div class="settings-section">
        <div class="settings-label">Тема</div>
        <div class="settings-toggle settings-toggle-3" data-setting="theme">
          <button class="settings-toggle-btn${settings.theme === "light" ? " active" : ""}" data-value="light">Светлая</button>
          <button class="settings-toggle-btn${settings.theme === "dark" ? " active" : ""}" data-value="dark">Тёмная</button>
          <button class="settings-toggle-btn${settings.theme === "auto" ? " active" : ""}" data-value="auto">Авто</button>
        </div>
      </div>

      <div class="settings-row">
        <div class="settings-col">
          <div class="settings-label">Выравнивание</div>
          <div class="settings-toggle settings-toggle-2" data-setting="justify">
            <button class="settings-toggle-btn${!settings.justify ? " active" : ""}" data-value="false">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 6H3"/><path d="M21 12H3"/><path d="M15 18H3"/></svg>
            </button>
            <button class="settings-toggle-btn${settings.justify ? " active" : ""}" data-value="true">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M3 12h18"/><path d="M3 18h18"/></svg>
            </button>
          </div>
        </div>
        <div class="settings-col">
          <div class="settings-label">Отступ</div>
          <div class="settings-toggle settings-toggle-4" data-setting="indent">
            <button class="settings-toggle-btn${settings.indent === 0 ? " active" : ""}" data-value="0">0</button>
            <button class="settings-toggle-btn${settings.indent === 1 ? " active" : ""}" data-value="1">1</button>
            <button class="settings-toggle-btn${settings.indent === 2 ? " active" : ""}" data-value="2">2</button>
            <button class="settings-toggle-btn${settings.indent === 3 ? " active" : ""}" data-value="3">3</button>
          </div>
        </div>
      </div>

      <div class="settings-section">
        <div class="settings-label">Плотность текста</div>
        <div class="settings-toggle settings-toggle-3" data-setting="density">
          <button class="settings-toggle-btn${settings.density === "compact" ? " active" : ""}" data-value="compact">Компактно</button>
          <button class="settings-toggle-btn${settings.density === "normal" ? " active" : ""}" data-value="normal">Обычно</button>
          <button class="settings-toggle-btn${settings.density === "relaxed" ? " active" : ""}" data-value="relaxed">Свободно</button>
        </div>
      </div>
    </div>
  `;

  initSettingsInteractions();
}

function initSettingsInteractions(): void {
  const fontDropdownEl = document.getElementById("font-dropdown");
  if (fontDropdownEl) {
    new Dropdown(fontDropdownEl);
    fontDropdownEl.addEventListener("change", (e: Event) => {
      const customEvent = e as CustomEvent<{ value: string }>;
      settingsManager.updateSetting("fontFamily", customEvent.detail.value);
    });
  }

  const themeToggle = document.querySelector('[data-setting="theme"]');
  themeToggle?.addEventListener("click", (e) => {
    const btn = (e.target as HTMLElement).closest(
      ".settings-toggle-btn",
    ) as HTMLElement;
    if (!btn) return;
    const value = btn.dataset.value as "auto" | "light" | "dark";
    const currentTheme = settingsManager.getSettings().theme;
    if (value !== currentTheme) {
      enableThemeTransition();
      settingsManager.updateSetting("theme", value);
      updateActiveToggle(themeToggle as HTMLElement, value);
    }
  });

  const justifyToggle = document.querySelector('[data-setting="justify"]');
  justifyToggle?.addEventListener("click", (e) => {
    const btn = (e.target as HTMLElement).closest(
      ".settings-toggle-btn",
    ) as HTMLElement;
    if (!btn) return;
    const value = btn.dataset.value === "true";
    settingsManager.updateSetting("justify", value);
    updateActiveToggle(justifyToggle as HTMLElement, btn.dataset.value!);
  });

  const indentToggle = document.querySelector('[data-setting="indent"]');
  indentToggle?.addEventListener("click", (e) => {
    const btn = (e.target as HTMLElement).closest(
      ".settings-toggle-btn",
    ) as HTMLElement;
    if (!btn) return;
    const value = parseInt(btn.dataset.value || "0", 10);
    settingsManager.updateSetting("indent", value);
    updateActiveToggle(indentToggle as HTMLElement, btn.dataset.value!);
  });

  const densityToggle = document.querySelector('[data-setting="density"]');
  densityToggle?.addEventListener("click", (e) => {
    const btn = (e.target as HTMLElement).closest(
      ".settings-toggle-btn",
    ) as HTMLElement;
    if (!btn) return;
    const value = btn.dataset.value as "compact" | "normal" | "relaxed";
    settingsManager.updateSetting("density", value);
    updateActiveToggle(densityToggle as HTMLElement, value);
  });

  const fontDecrease = document.getElementById("font-decrease");
  const fontIncrease = document.getElementById("font-increase");
  const fontSizeValue = document.getElementById("font-size-value");

  fontDecrease?.addEventListener("click", () => {
    const current = settingsManager.getSettings().fontSize;
    if (current > 14) {
      const newSize = current - 1;
      settingsManager.updateSetting("fontSize", newSize);
      if (fontSizeValue) fontSizeValue.textContent = String(newSize);
      updateFontSizeButtons(newSize);
    }
  });

  fontIncrease?.addEventListener("click", () => {
    const current = settingsManager.getSettings().fontSize;
    if (current < 26) {
      const newSize = current + 1;
      settingsManager.updateSetting("fontSize", newSize);
      if (fontSizeValue) fontSizeValue.textContent = String(newSize);
      updateFontSizeButtons(newSize);
    }
  });
}

function updateActiveToggle(container: HTMLElement, activeValue: string): void {
  container.querySelectorAll(".settings-toggle-btn").forEach((btn) => {
    btn.classList.toggle(
      "active",
      (btn as HTMLElement).dataset.value === activeValue,
    );
  });
}

export { FONT_OPTIONS, getSettings };
