import Dropdown from "./dropdown";
import { setKappalibCookie } from "./profile";
import { uiManager } from "./ui";

const SETTINGS_COOKIE_KEY = "kappalib_reader_settings";

interface ReaderSettings {
  theme: "auto" | "light" | "dark";
  colorScheme:
    | "default"
    | "catppuccin"
    | "gruvbox"
    | "nord"
    | "rosepine"
    | "tokyonight"
    | "everforest"
    | "flexoki"
    | "sonokai"
    | "oxocarbon";
  fontSize: number;
  fontFamily: string;
  indent: number;
  density: "compact" | "normal" | "relaxed";
  justify: boolean;
  showComments: boolean;
}

const DEFAULT_SETTINGS: ReaderSettings = {
  theme: "auto",
  colorScheme: "default",
  fontSize: 18,
  fontFamily: "default",
  indent: 0,
  density: "normal",
  justify: false,
  showComments: true,
};

const FONT_OPTIONS: { value: string; label: string; family: string }[] = [
  { value: "default", label: "Стандартный", family: "inherit" },
  { value: "literata", label: "Literata", family: "Literata, serif" },
  {
    value: "nunito",
    label: "Nunito",
    family: "Nunito, sans-serif",
  },
  {
    value: "merriweather",
    label: "Merriweather",
    family: "Merriweather, serif",
  },
  { value: "lora", label: "Lora", family: "Lora, serif" },
  { value: "pt-serif", label: "PT Serif", family: "PT Serif, serif" },
  { value: "open-sans", label: "Open Sans", family: "Open Sans, sans-serif" },
  { value: "roboto", label: "Roboto", family: "Roboto, sans-serif" },
];

const COLOR_SCHEME_OPTIONS: {
  value: string;
  label: string;
  colors: { light: string[]; dark: string[] };
}[] = [
  {
    value: "default",
    label: "Стандартная",
    colors: {
      light: ["#f9f9f9", "#1b1b1b", "#757575", "#f97316"],
      dark: ["#131313", "#e7e7e7", "#a5a5a5", "#f97316"],
    },
  },
  {
    value: "catppuccin",
    label: "Catppuccin",
    colors: {
      light: ["#eff1f5", "#4c4f69", "#6c6f85", "#fe640b"],
      dark: ["#1e1e2e", "#cdd6f4", "#a6adc8", "#fab387"],
    },
  },
  {
    value: "gruvbox",
    label: "Gruvbox",
    colors: {
      light: ["#fbf1c7", "#3c3836", "#665c54", "#d65d0e"],
      dark: ["#282828", "#ebdbb2", "#a89984", "#fe8019"],
    },
  },
  {
    value: "nord",
    label: "Nord",
    colors: {
      light: ["#eceff4", "#2e3440", "#4c566a", "#d08770"],
      dark: ["#2e3440", "#eceff4", "#d8dee9", "#d08770"],
    },
  },
  {
    value: "rosepine",
    label: "Rosé Pine",
    colors: {
      light: ["#faf4ed", "#575279", "#797593", "#d7827e"],
      dark: ["#232136", "#e0def4", "#908caa", "#ea9a97"],
    },
  },
  {
    value: "tokyonight",
    label: "Tokyo Night",
    colors: {
      light: ["#e1e2e7", "#3760bf", "#6172b0", "#ff9e64"],
      dark: ["#1a1b26", "#c0caf5", "#a9b1d6", "#ff9e64"],
    },
  },
  {
    value: "everforest",
    label: "Everforest",
    colors: {
      light: ["#fdf6e3", "#5c6a72", "#829181", "#e69875"],
      dark: ["#2d353b", "#d3c6aa", "#9da9a0", "#e69875"],
    },
  },
  {
    value: "flexoki",
    label: "Flexoki",
    colors: {
      light: ["#fffcf0", "#1c1b1a", "#6f6e69", "#da702c"],
      dark: ["#100f0f", "#cecdc3", "#878580", "#da702c"],
    },
  },
  {
    value: "sonokai",
    label: "Sonokai",
    colors: {
      light: ["#fbfaf9", "#2d2a2e", "#69676c", "#f85e84"],
      dark: ["#2d2a2e", "#e3e1e4", "#9f9ca6", "#f85e84"],
    },
  },
  {
    value: "oxocarbon",
    label: "Oxocarbon",
    colors: {
      light: ["#f2f4f8", "#161616", "#525252", "#ee5396"],
      dark: ["#161616", "#f2f4f8", "#dde1e6", "#ee5396"],
    },
  },
];

const FONT_URLS: Record<string, string> = {
  literata: "https://cdn.jsdelivr.net/npm/@fontsource/literata@5/index.min.css",
  nunito: "https://cdn.jsdelivr.net/npm/@fontsource/nunito@5/index.min.css",
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
  setKappalibCookie(SETTINGS_COOKIE_KEY, JSON.stringify(settings));
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

  if (settings.colorScheme === "default") {
    root.removeAttribute("data-scheme");
  } else {
    root.setAttribute("data-scheme", settings.colorScheme);
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

  const commentsSection = document.getElementById(
    "comments-section",
  ) as HTMLElement | null;
  if (commentsSection) {
    commentsSection.style.display = settings.showComments ? "" : "none";
  }

  document
    .querySelectorAll<HTMLElement>(".comment-jump-link")
    .forEach((el) => {
      el.style.display = settings.showComments ? "" : "none";
    });
}

function applyGlobalSettings(): void {
  const settings = getSettings();
  const root = document.documentElement;

  if (settings.theme === "auto") {
    root.removeAttribute("data-theme");
  } else {
    root.setAttribute("data-theme", settings.theme);
  }

  if (settings.colorScheme === "default") {
    root.removeAttribute("data-scheme");
  } else {
    root.setAttribute("data-scheme", settings.colorScheme);
  }
}

function isDarkMode(): boolean {
  const root = document.documentElement;
  const theme = root.getAttribute("data-theme");
  if (theme === "dark") return true;
  if (theme === "light") return false;
  return window.matchMedia("(prefers-color-scheme: dark)").matches;
}

function createColorPalette(colors: {
  light: string[];
  dark: string[];
}): HTMLElement {
  const palette = document.createElement("div");
  palette.className = "scheme-palette";

  const currentColors = isDarkMode() ? colors.dark : colors.light;

  currentColors.forEach((color) => {
    const swatch = document.createElement("span");
    swatch.className = "scheme-swatch";
    swatch.style.backgroundColor = color;
    palette.appendChild(swatch);
  });

  return palette;
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
    this.settings = getSettings();
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

  if (!settingsCard || !settingsBtn) return;

  settingsBtn.addEventListener("click", (e) => {
    e.preventDefault();
    e.stopPropagation();

    uiManager.toggleSettings();
    if (settingsCard.style.display === "block") {
      renderSettingsView();
    }
  });

  document.addEventListener("click", (e) => {
    if (
      settingsCard.style.display === "block" &&
      !settingsCard.contains(e.target as Node) &&
      !settingsBtn.contains(e.target as Node)
    ) {
      uiManager.closeAll();
    }
  });

  settingsManager.applyAll();
}

function cloneTemplate(id: string): DocumentFragment {
  const template = document.getElementById(id) as HTMLTemplateElement | null;
  if (!template) {
    console.error(`Template #${id} not found`);
    return document.createDocumentFragment();
  }
  return template.content.cloneNode(true) as DocumentFragment;
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

function createToggleButton(
  value: string,
  label: string,
  isActive: boolean,
): HTMLButtonElement {
  const btn = document.createElement("button");
  btn.className = "settings-toggle-btn" + (isActive ? " active" : "");
  btn.dataset.value = value;
  btn.innerHTML = label;
  return btn;
}

function renderSettingsView(): void {
  const content = document.getElementById("settings-card");
  if (!content) return;

  const settings = settingsManager.getSettings();
  const currentFont =
    FONT_OPTIONS.find((f) => f.value === settings.fontFamily) ||
    FONT_OPTIONS[0];
  const currentScheme =
    COLOR_SCHEME_OPTIONS.find((s) => s.value === settings.colorScheme) ||
    COLOR_SCHEME_OPTIONS[0];

  content.innerHTML = "";
  content.appendChild(cloneTemplate("tpl-settings"));

  const fontLabel = content.querySelector('[data-field="currentFontLabel"]');
  if (fontLabel) fontLabel.textContent = currentFont.label;

  const schemeLabel = content.querySelector(
    '[data-field="currentSchemeLabel"]',
  );
  if (schemeLabel) schemeLabel.textContent = currentScheme.label;

  const schemePaletteBtn = content.querySelector(
    "#scheme-dropdown .dropdown-btn",
  );
  if (schemePaletteBtn) {
    const existingPalette = schemePaletteBtn.querySelector(".scheme-palette");
    if (existingPalette) existingPalette.remove();
    const palette = createColorPalette(currentScheme.colors);
    schemePaletteBtn.insertBefore(
      palette,
      schemePaletteBtn.querySelector(".chevron"),
    );
  }

  const fontSizeValue = content.querySelector('[data-field="fontSize"]');
  if (fontSizeValue) fontSizeValue.textContent = String(settings.fontSize);

  const fontOptionsContainer = document.getElementById(
    "font-options-container",
  );
  if (fontOptionsContainer) {
    FONT_OPTIONS.forEach((f) => {
      const optionTemplate = cloneTemplate("tpl-font-option");
      const btn = optionTemplate.querySelector(".dropdown-item") as HTMLElement;
      if (btn) {
        btn.dataset.value = f.value;
        btn.setAttribute(
          "aria-selected",
          String(settings.fontFamily === f.value),
        );
        if (settings.fontFamily === f.value) {
          btn.classList.add("selected");
        }
        const label = btn.querySelector('[data-field="label"]');
        if (label) label.textContent = f.label;
      }
      fontOptionsContainer.appendChild(optionTemplate);
    });
  }

  const schemeOptionsContainer = document.getElementById(
    "scheme-options-container",
  );
  if (schemeOptionsContainer) {
    COLOR_SCHEME_OPTIONS.forEach((s) => {
      const btn = document.createElement("button");
      btn.className = "dropdown-item";
      if (settings.colorScheme === s.value) {
        btn.classList.add("selected");
      }
      btn.dataset.value = s.value;
      btn.setAttribute("role", "option");
      btn.setAttribute(
        "aria-selected",
        String(settings.colorScheme === s.value),
      );

      const labelSpan = document.createElement("span");
      labelSpan.textContent = s.label;
      btn.appendChild(labelSpan);

      const palette = createColorPalette(s.colors);
      btn.appendChild(palette);

      const checkIcon = document.createElementNS(
        "http://www.w3.org/2000/svg",
        "svg",
      );
      checkIcon.setAttribute("class", "check-icon");
      checkIcon.setAttribute("width", "16");
      checkIcon.setAttribute("height", "16");
      checkIcon.setAttribute("viewBox", "0 0 24 24");
      checkIcon.setAttribute("fill", "none");
      checkIcon.setAttribute("stroke", "currentColor");
      checkIcon.setAttribute("stroke-width", "3");
      checkIcon.setAttribute("stroke-linecap", "round");
      checkIcon.setAttribute("stroke-linejoin", "round");
      const polyline = document.createElementNS(
        "http://www.w3.org/2000/svg",
        "polyline",
      );
      polyline.setAttribute("points", "20 6 9 17 4 12");
      checkIcon.appendChild(polyline);
      btn.appendChild(checkIcon);

      schemeOptionsContainer.appendChild(btn);
    });
  }

  const themeToggle = document.getElementById("theme-toggle");
  if (themeToggle) {
    themeToggle.appendChild(
      createToggleButton("light", "Светлая", settings.theme === "light"),
    );
    themeToggle.appendChild(
      createToggleButton("dark", "Тёмная", settings.theme === "dark"),
    );
    themeToggle.appendChild(
      createToggleButton("auto", "Авто", settings.theme === "auto"),
    );
  }

  const justifyToggle = document.getElementById("justify-toggle");
  if (justifyToggle) {
    justifyToggle.appendChild(
      createToggleButton(
        "false",
        '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 6H3"/><path d="M21 12H3"/><path d="M15 18H3"/></svg>',
        !settings.justify,
      ),
    );
    justifyToggle.appendChild(
      createToggleButton(
        "true",
        '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M3 12h18"/><path d="M3 18h18"/></svg>',
        settings.justify,
      ),
    );
  }

  const indentToggle = document.getElementById("indent-toggle");
  if (indentToggle) {
    for (let i = 0; i <= 3; i++) {
      indentToggle.appendChild(
        createToggleButton(String(i), String(i), settings.indent === i),
      );
    }
  }

  const densityToggle = document.getElementById("density-toggle");
  if (densityToggle) {
    densityToggle.appendChild(
      createToggleButton(
        "compact",
        "Компактно",
        settings.density === "compact",
      ),
    );
    densityToggle.appendChild(
      createToggleButton("normal", "Обычно", settings.density === "normal"),
    );
    densityToggle.appendChild(
      createToggleButton("relaxed", "Свободно", settings.density === "relaxed"),
    );
  }

  const commentsToggle = document.getElementById("comments-toggle");
  if (commentsToggle) {
    commentsToggle.appendChild(
      createToggleButton("true", "Показывать", settings.showComments),
    );
    commentsToggle.appendChild(
      createToggleButton("false", "Скрывать", !settings.showComments),
    );
  }

  updateFontSizeButtons(settings.fontSize);
  initSettingsInteractions();
}

function updateSchemePalettes(): void {
  const schemeDropdown = document.getElementById("scheme-dropdown");
  if (!schemeDropdown) return;

  schemeDropdown.querySelectorAll(".scheme-palette").forEach((palette) => {
    const btn = palette.closest("[data-value]") as HTMLElement;
    if (!btn) return;

    const scheme = COLOR_SCHEME_OPTIONS.find(
      (s) => s.value === btn.dataset.value,
    );
    if (!scheme) return;

    const currentColors = isDarkMode()
      ? scheme.colors.dark
      : scheme.colors.light;
    const swatches = palette.querySelectorAll(".scheme-swatch");
    swatches.forEach((swatch, i) => {
      if (currentColors[i]) {
        (swatch as HTMLElement).style.backgroundColor = currentColors[i];
      }
    });
  });
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

  const schemeDropdownEl = document.getElementById("scheme-dropdown");
  if (schemeDropdownEl) {
    new Dropdown(schemeDropdownEl);
    schemeDropdownEl.addEventListener("change", (e: Event) => {
      const customEvent = e as CustomEvent<{ value: string }>;
      enableThemeTransition();
      settingsManager.updateSetting(
        "colorScheme",
        customEvent.detail.value as ReaderSettings["colorScheme"],
      );

      const scheme = COLOR_SCHEME_OPTIONS.find(
        (s) => s.value === customEvent.detail.value,
      );
      if (scheme) {
        const btnPalette = schemeDropdownEl.querySelector(
          ".dropdown-btn .scheme-palette",
        );
        if (btnPalette) {
          const currentColors = isDarkMode()
            ? scheme.colors.dark
            : scheme.colors.light;
          const swatches = btnPalette.querySelectorAll(".scheme-swatch");
          swatches.forEach((swatch, i) => {
            if (currentColors[i]) {
              (swatch as HTMLElement).style.backgroundColor = currentColors[i];
            }
          });
        }
      }
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
      setTimeout(updateSchemePalettes, 50);
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

  const commentsToggle = document.querySelector('[data-setting="showComments"]');
  commentsToggle?.addEventListener("click", (e) => {
    const btn = (e.target as HTMLElement).closest(
      ".settings-toggle-btn",
    ) as HTMLElement;
    if (!btn) return;
    const value = btn.dataset.value === "true";
    settingsManager.updateSetting("showComments", value);
    updateActiveToggle(commentsToggle as HTMLElement, btn.dataset.value!);
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

export { FONT_OPTIONS, COLOR_SCHEME_OPTIONS, getSettings };
