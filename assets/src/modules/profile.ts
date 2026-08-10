import { uiManager } from "./ui";
import { settingsManager } from "./settings";
import { initComments } from "./comments";
import { refreshHistory } from "./history";
import { refreshLastReadTotalChapters } from "./progress";

const API_URL = process.env.API_URL;
const PROFILE_ID_KEY = "kappalib_profile_id";
const PROFILE_PROVIDER_KEY = "kappalib_oauth_provider";
const NOTIFICATIONS_KEY = "kappalib_unread_notifications";
const S3_URL = process.env.S3_PUBLIC_URL;

interface CookieValue {
  value: string;
  updated_at: number;
}

interface ProfilePublic {
  id: string;
  display_name: string;
  avatar_seed: string;
  has_custom_avatar: boolean;
  avatar_updated_at: number;
  created_at: string;
  unread_notifications?: number;
}

export function getAvatarUrl(
  userId: string,
  hasCustomAvatar: boolean,
  avatarSeed: string,
  avatarUpdatedAt: number = 0,
): string {
  if (hasCustomAvatar) {
    return `${S3_URL}/avatars/${userId}.jpg?v=${avatarUpdatedAt}`;
  }
  return `https://api.dicebear.com/9.x/bottts-neutral/svg?seed=${avatarSeed}&backgroundType=solid,gradientLinear`;
}

class ProfileManager {
  private profileId: string | null = null;
  private cachedProfile: ProfilePublic | null = null;

  constructor() {
    this.profileId = localStorage.getItem(PROFILE_ID_KEY);
  }

  isLoggedIn(): boolean {
    return !!this.profileId;
  }

  getProfileId(): string | null {
    return this.profileId;
  }

  getProvider(): string | null {
    return localStorage.getItem(PROFILE_PROVIDER_KEY);
  }

  setProvider(provider: string): void {
    localStorage.setItem(PROFILE_PROVIDER_KEY, provider);
  }

  getAvatarUrl(profile: ProfilePublic): string {
    if (profile.has_custom_avatar) {
      return `${S3_URL}/avatars/${profile.id}.jpg?v=${profile.avatar_updated_at}`;
    }
    return `https://api.dicebear.com/9.x/bottts-neutral/svg?seed=${profile.avatar_seed}&backgroundType=solid,gradientLinear`;
  }

  getProfileCache(): ProfilePublic | null {
    return this.cachedProfile;
  }

  async fetchProfile(): Promise<ProfilePublic | null> {
    try {
      const url = `${API_URL}/profile/me`;
      const res = await fetch(url, { credentials: "include" });
      if (res.ok) {
        const profile = await res.json();
        this.profileId = profile.id;
        this.cachedProfile = profile;
        localStorage.setItem(PROFILE_ID_KEY, profile.id);
        localStorage.setItem(NOTIFICATIONS_KEY, String(profile.unread_notifications || 0));
        this.notifyLogin();
        return profile;
      }
      if (res.status === 401 || res.status === 404) {
        this.clearLocal();
      }
    } catch (err) {
      console.error("Fetch profile failed", err);
    }
    return null;
  }

  getInitialUnreadCount(): number {
    if (!this.isLoggedIn()) return 0;
    const stored = localStorage.getItem(NOTIFICATIONS_KEY);
    return stored ? parseInt(stored, 10) : 0;
  }

  async syncCookiesToServer(): Promise<void> {
    if (!this.profileId) return;
    const cookies = this.getKappalibCookies();
    try {
      const res = await fetch(`${API_URL}/profile/sync-cookies`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ cookies }),
      });
      if (res.ok) {
        const merged: Record<string, CookieValue> = await res.json();
        if (this.applyCookiesIfChanged(merged)) {
          this.notifySync();
        }
      }
    } catch (err) {
      console.error("Sync cookies failed", err);
    }
  }

  async updateDisplayName(newName: string): Promise<{ error?: string; profile?: ProfilePublic }> {
    if (!this.profileId) return { error: "Not logged in" };
    try {
      const res = await fetch(`${API_URL}/profile/${this.profileId}/name`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ display_name: newName }),
      });
      if (res.ok) {
        return { profile: await res.json() };
      }
      const error = await res.json().catch(() => ({ detail: "Failed to update name" }));
      return { error: error.detail || "Failed to update name" };
    } catch (err) {
      return { error: "Network error" };
    }
  }

  async uploadAvatar(file: File): Promise<ProfilePublic | null> {
    if (!this.profileId) return null;
    try {
      const base64 = await this.fileToBase64(file);

      const res = await fetch(`${API_URL}/profile/${this.profileId}/avatar`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ image: base64 }),
      });

      if (res.ok) {
        return await res.json();
      }

      const error = await res.json().catch(() => null);
      if (error?.detail) {
        alert(error.detail);
      }
      return null;
    } catch (err) {
      console.error("Upload avatar failed", err);
    }
    return null;
  }

  private fileToBase64(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result as string;
        const base64 = result.split(",")[1];
        resolve(base64);
      };
      reader.onerror = reject;
      reader.readAsDataURL(file);
    });
  }

  async deleteProfile(): Promise<boolean> {
    if (!this.profileId) return false;
    try {
      const res = await fetch(`${API_URL}/profile/${this.profileId}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (res.ok) {
        this.clearLocal();
        return true;
      }
      if (res.status === 401 || res.status === 403) this.clearLocal();
    } catch (err) {
      console.error("Delete profile failed", err);
    }
    return false;
  }

  async logout(): Promise<void> {
    try {
      await fetch(`${API_URL}/profile/logout`, {
        method: "POST",
        credentials: "include",
      });
    } catch {
      // ignore errors
    }
    this.clearLocal();
  }

  private clearLocal(): void {
    this.profileId = null;
    this.cachedProfile = null;
    localStorage.removeItem(PROFILE_ID_KEY);
    localStorage.removeItem(PROFILE_PROVIDER_KEY);
    localStorage.removeItem("kappalib_pending_comments");
    localStorage.removeItem(NOTIFICATIONS_KEY);
  }

  private getKappalibCookies(): Record<string, CookieValue> {
    const cookies: Record<string, CookieValue> = {};
    document.cookie.split(";").forEach((c) => {
      const [name, rawValue] = c.trim().split("=");
      if (name && name.startsWith("kappalib_") && rawValue) {
        if (name === "kpl_session") return;
        const value = decodeURIComponent(rawValue);
        const timestampKey = `${name}_updated_at`;
        const storedTimestamp = localStorage.getItem(timestampKey);
        const updatedAt = storedTimestamp
          ? parseInt(storedTimestamp, 10)
          : Date.now();

        cookies[name] = {
          value,
          updated_at: updatedAt,
        };
      }
    });
    return cookies;
  }

  private onLoginCallbacks: Array<() => void> = [];
  private onSyncCallbacks: Array<() => void> = [];

  onLogin(callback: () => void): void {
    if (this.cachedProfile) {
      callback();
    } else {
      this.onLoginCallbacks.push(callback);
    }
  }

  onSync(callback: () => void): void {
    this.onSyncCallbacks.push(callback);
  }

  private notifyLogin(): void {
    this.onLoginCallbacks.forEach((cb) => cb());
    this.onLoginCallbacks = [];
  }

  private notifySync(): void {
    this.onSyncCallbacks.forEach((cb) => cb());
  }

  private applyCookiesIfChanged(cookies: Record<string, CookieValue>): boolean {
    let hasChanges = false;
    const current = document.cookie
      .split(";")
      .reduce<Record<string, string>>((acc, c) => {
        const [name, rawValue] = c.trim().split("=");
        if (name && rawValue) acc[name] = decodeURIComponent(rawValue);
        return acc;
      }, {});

    for (const [name, cv] of Object.entries(cookies)) {
      if (name.startsWith("kappalib_") && name !== "kpl_session") {
        if (current[name] !== cv.value) {
          hasChanges = true;
        }
        document.cookie = `${name}=${encodeURIComponent(cv.value)}; path=/; max-age=31536000; SameSite=Lax`;
        localStorage.setItem(`${name}_updated_at`, cv.updated_at.toString());
      }
    }

    return hasChanges;
  }

  markNotificationsAsRead(): void {
    if (this.cachedProfile) {
      this.cachedProfile.unread_notifications = 0;
    }
    localStorage.setItem(NOTIFICATIONS_KEY, "0");
  }
}

export const profileManager = new ProfileManager();

function refreshUI(): void {
  settingsManager.applyAll();

  const commentsSection = document.getElementById("comments-section");
  if (commentsSection) {
    initComments();
  }

  if (document.querySelector(".cr-wrapper")) {
    refreshLastReadTotalChapters();
  }

  const historyList = document.getElementById("history-list");
  if (historyList) {
    refreshHistory();
  }

  updateProfileBadges();
}

export function initProfile(): void {
  const initialCount = profileManager.getInitialUnreadCount();
  if (initialCount > 0) {
    const headerBtn = document.getElementById("header-profile-btn");
    if (headerBtn) {
      const badge = document.createElement("span");
      badge.className = "pc-notifications-badge";
      badge.textContent = initialCount > 99 ? "99+" : String(initialCount);
      badge.style.display = "flex";
      headerBtn.appendChild(badge);
    }
  }

  profileManager.fetchProfile().then((profile) => {
    if (profile) {
      profileManager.syncCookiesToServer().finally(() => {
        refreshUI();
      });
    } else {
      refreshUI();
    }
  });
}

export function updateKappalibCookieQuietly(name: string, value: string): void {
  if (!name.startsWith("kappalib_")) {
    name = `kappalib_${name}`;
  }
  const encodedValue = encodeURIComponent(value);
  document.cookie = `${name}=${encodedValue}; path=/; max-age=31536000; SameSite=Lax`;
}

export function setKappalibCookie(name: string, value: string): void {
  if (!name.startsWith("kappalib_")) {
    name = `kappalib_${name}`;
  }
  const timestamp = Date.now();
  const encodedValue = encodeURIComponent(value);
  document.cookie = `${name}=${encodedValue}; path=/; max-age=31536000; SameSite=Lax`;
  localStorage.setItem(`${name}_updated_at`, timestamp.toString());

  if (profileManager.isLoggedIn()) {
    profileManager.syncCookiesToServer();
  }
}

function cloneTemplate(id: string): DocumentFragment {
  const template = document.getElementById(id) as HTMLTemplateElement | null;
  if (!template) {
    console.error(`Template #${id} not found`);
    return document.createDocumentFragment();
  }
  return template.content.cloneNode(true) as DocumentFragment;
}

function fillTemplate(
  id: string,
  data: Record<string, string>,
): DocumentFragment {
  const fragment = cloneTemplate(id);

  for (const [key, value] of Object.entries(data)) {
    const el = fragment.querySelector(`[data-field="${key}"]`);
    if (el) {
      if (el.tagName === "IMG") {
        (el as HTMLImageElement).src = value;
      } else {
        el.textContent = value;
      }
    }
  }

  return fragment;
}

export function initProfileModal(): void {
  const profileCard = document.getElementById("profile-card");
  const profileBtn = document.getElementById("header-profile-btn");

  if (!profileCard || !profileBtn) return;

  profileBtn.addEventListener("click", (e) => {
    e.preventDefault();
    e.stopPropagation();

    const searchInput = document.getElementById(
      "search-input",
    ) as HTMLInputElement | null;
    if (searchInput) searchInput.value = "";

    uiManager.toggleProfile(() => {
      renderProfileCard();
    });
  });

  document.addEventListener("click", (e) => {
    if (
      profileCard.style.display === "block" &&
      !profileCard.contains(e.target as Node) &&
      !profileBtn.contains(e.target as Node)
    ) {
      uiManager.closeAll();
    }
  });
}

function renderProfileCard(): void {
  const content = document.getElementById("profile-card");
  if (!content) return;

  const hasSession = content.dataset.hasSession === "true";

  content.innerHTML = "";
  content.appendChild(
    cloneTemplate(hasSession ? "tpl-pc-skeleton" : "tpl-pc-guest-skeleton"),
  );

  profileManager.fetchProfile().then((profile) => {
    if (!profile) {
      renderGuestView();
    } else {
      renderLoggedInView(profile);
    }
    updateProfileBadges();
  });
}

function renderGuestView(): void {
  const content = document.getElementById("profile-card");
  if (!content) return;

  content.innerHTML = "";
  content.appendChild(cloneTemplate("tpl-pc-guest"));

  const currentPath = window.location.pathname;
  const oauthLinks = content.querySelectorAll(".pc-btn-oauth");
  oauthLinks.forEach((link) => {
    const href = link.getAttribute("href");
    if (href) {
      link.setAttribute(
        "href",
        `${href}?from=${encodeURIComponent(currentPath)}`,
      );
      link.addEventListener("click", () => {
        const provider = href.split("/auth/")[1]?.split("/")[0];
        if (provider) {
          profileManager.setProvider(provider);
        }
      });
    }
  });
}

function renderLoggedInView(profile: ProfilePublic): void {
  const content = document.getElementById("profile-card");
  if (!content) return;

  const avatarUrl = profileManager.getAvatarUrl(profile);

  content.innerHTML = "";
  content.appendChild(
    fillTemplate("tpl-pc-profile", {
      avatarUrl,
      displayName: profile.display_name,
      createdAt: formatDate(profile.created_at),
    }),
  );

  const commentsBtn = content.querySelector('a[href="/comments"]');
  const count = profile.unread_notifications || 0;
  if (commentsBtn && count > 0) {
    const badge = document.createElement("span");
    badge.className = "pc-notifications-badge";
    badge.textContent = count > 99 ? "99+" : String(count);
    commentsBtn.appendChild(badge);
  }

  initProfileInteractions(profile);
}

function initProfileInteractions(profile: ProfilePublic): void {
  const avatarWrapper = document.getElementById("pc-avatar-img")?.parentElement;
  const avatarInput = document.getElementById(
    "pc-avatar-input",
  ) as HTMLInputElement;
  const nameText = document.getElementById("pc-name-text");
  const nameInput = document.getElementById(
    "pc-name-input",
  ) as HTMLInputElement;

  let isSavingName = false;
  let currentProfile = profile;

  avatarWrapper?.addEventListener("click", () => {
    avatarInput?.click();
  });

  avatarInput?.addEventListener("change", async () => {
    const file = avatarInput.files?.[0];
    if (!file) return;

    if (file.size > 1024 * 1024) {
      alert("Файл слишком большой (максимум 1 МБ)");
      avatarInput.value = "";
      return;
    }

    const avatarImg = document.getElementById(
      "pc-avatar-img",
    ) as HTMLImageElement;
    const overlay = document.getElementById("pc-avatar-overlay");

    if (avatarImg) avatarImg.style.opacity = "0.5";
    if (overlay) overlay.style.opacity = "0";

    const result = await profileManager.uploadAvatar(file);

    if (avatarImg) avatarImg.style.opacity = "1";
    avatarInput.value = "";

    if (result && avatarImg) {
      avatarImg.src = profileManager.getAvatarUrl(result);
    }
  });

  nameText?.addEventListener("click", () => {
    if (!nameText || !nameInput) return;
    restoreMetaDate();
    nameText.style.display = "none";
    nameInput.style.display = "block";
    nameInput.value = "";
    nameInput.placeholder = currentProfile.display_name;
    nameInput.focus();
  });

  nameInput?.addEventListener("blur", () => {
    saveName();
  });

  nameInput?.addEventListener("keydown", (e) => {
    if (e.key === "Enter") {
      e.preventDefault();
      nameInput.blur();
    }
    if (e.key === "Escape") {
      nameInput.value = "";
      nameInput.blur();
    }
  });

  async function saveName() {
    if (!nameText || !nameInput) return;
    if (isSavingName) return;

    const newName = nameInput.value.trim();

    if (!newName || newName === currentProfile.display_name) {
      cancelNameEdit();
      return;
    }

    isSavingName = true;
    nameInput.disabled = true;

    const result = await profileManager.updateDisplayName(newName);

    nameInput.disabled = false;
    isSavingName = false;

    if (result.error) {
      const metaDateEl = document.querySelector(".pc-meta-date") as HTMLElement;
      if (metaDateEl) {
        metaDateEl.textContent = mapErrorToRussian(result.error);
        metaDateEl.style.color = "var(--color-danger)";
      }
      nameInput.focus();
      return;
    }

    if (result.profile) {
      currentProfile.display_name = result.profile.display_name;
      nameText.textContent = result.profile.display_name;
    }

    cancelNameEdit();
  }

  function mapErrorToRussian(error: string): string {
    const lower = error.toLowerCase();
    if (lower.includes("empty")) return "Пустое имя";
    if (lower.includes("too long") || lower.includes("15")) return "Слишком длинное";
    if (lower.includes("invalid") || lower.includes("character")) return "Недопустимые символы";
    return "Ошибка";
  }

  function cancelNameEdit() {
    if (!nameText || !nameInput) return;
    nameInput.style.display = "none";
    nameInput.value = "";
    nameText.style.display = "inline";
    restoreMetaDate();
  }

  function restoreMetaDate() {
    const metaDateEl = document.querySelector(".pc-meta-date") as HTMLElement;
    if (metaDateEl) {
      metaDateEl.innerHTML = `<svg xmlns="http://w.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 13H3"></path><path d="M16 17H3"></path><path d="m7.2 7.9-3.388 2.5A2 2 0 0 0 3 12.01V20a1 1 0 0 0 1 1h16a1 1 0 0 0 1-1v-8.654c0-2-2.44-6.026-6.44-8.026a1 1 0 0 0-1.082.057L10.4 5.6"></path><circle cx="9" cy="7" r="2"></circle></svg><span data-field="createdAt">${formatDate(currentProfile.created_at)}</span>`;
      metaDateEl.style.color = "";
    }
  }

  document.getElementById("pc-logout")?.addEventListener("click", async () => {
    await profileManager.logout();
    const profileCard = document.getElementById("profile-card");
    if (profileCard) {
      profileCard.dataset.hasSession = "false";
    }
    renderGuestView();
    updateProfileBadges();
  });
}

export function updateProfileBadges(): void {
  const profile = profileManager.getProfileCache();
  const count = profile?.unread_notifications || 0;

  const headerBtn = document.getElementById("header-profile-btn");
  if (!headerBtn) return;

  let badge = headerBtn.querySelector(".pc-notifications-badge") as HTMLElement | null;

  if (count > 0) {
    if (!badge) {
      badge = document.createElement("span");
      badge.className = "pc-notifications-badge";
      headerBtn.appendChild(badge);
    }
    badge.textContent = count > 99 ? "99+" : String(count);
    badge.style.display = "flex";
  } else if (badge) {
    badge.style.display = "none";
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString("ru-RU", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}
