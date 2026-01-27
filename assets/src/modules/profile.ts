const API_URL = process.env.API_URL;
const PROFILE_ID_KEY = "kappalib_profile_id";
const PROFILE_PROVIDER_KEY = "kappalib_oauth_provider";
const S3_URL = `${process.env.S3_USE_SSL !== "false" ? "https" : "http"}://${process.env.S3_ENDPOINT}/${process.env.S3_BUCKET}`;

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

  async fetchProfile(): Promise<ProfilePublic | null> {
    try {
      const url = `${API_URL}/profile/me`;
      const res = await fetch(url, { credentials: "include" });
      if (res.ok) {
        const profile = await res.json();
        this.profileId = profile.id;
        localStorage.setItem(PROFILE_ID_KEY, profile.id);
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
        this.applyCookies(merged);
      }
    } catch (err) {
      console.error("Sync cookies failed", err);
    }
  }

  async updateDisplayName(newName: string): Promise<ProfilePublic | null> {
    if (!this.profileId) return null;
    try {
      const res = await fetch(`${API_URL}/profile/${this.profileId}/name`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ display_name: newName }),
      });
      if (res.ok) return await res.json();
    } catch (err) {
      console.error("Update name failed", err);
    }
    return null;
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
    localStorage.removeItem(PROFILE_ID_KEY);
    localStorage.removeItem(PROFILE_PROVIDER_KEY);
    localStorage.removeItem("kappalib_pending_comments");
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

  private applyCookies(cookies: Record<string, CookieValue>): void {
    for (const [name, cv] of Object.entries(cookies)) {
      if (name.startsWith("kappalib_") && name !== "kpl_session") {
        document.cookie = `${name}=${encodeURIComponent(cv.value)}; path=/; max-age=31536000; SameSite=Lax`;
        localStorage.setItem(`${name}_updated_at`, cv.updated_at.toString());
      }
    }
  }
}

export const profileManager = new ProfileManager();

export function initProfile(): void {
  profileManager.fetchProfile().then((profile) => {
    if (profile) {
      profileManager.syncCookiesToServer();
    }
  });
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
  const backdrop = document.getElementById("header-backdrop");
  const settingsCard = document.getElementById("settings-card");

  if (!profileCard || !profileBtn) return;

  profileBtn.addEventListener("click", (e) => {
    e.preventDefault();
    e.stopPropagation();

    const settingsBtn = document.getElementById("header-settings-btn");
    if (settingsCard?.style.display === "block") {
      settingsCard.style.display = "none";
      settingsBtn?.classList.remove("active");
    }

    if (profileCard.style.display === "block") {
      closeProfileCard();
    } else {
      openProfileCard();
    }
  });

  backdrop?.addEventListener("click", () => {
    closeProfileCard();
  });

  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && profileCard.style.display === "block") {
      closeProfileCard();
    }
  });

  document.addEventListener("click", (e) => {
    if (
      profileCard.style.display === "block" &&
      !profileCard.contains(e.target as Node) &&
      !profileBtn.contains(e.target as Node)
    ) {
      closeProfileCard();
    }
  });
}

function openProfileCard(): void {
  const profileCard = document.getElementById("profile-card");
  const profileBtn = document.getElementById("header-profile-btn");
  const backdrop = document.getElementById("header-backdrop");
  if (!profileCard) return;

  profileCard.style.display = "block";
  profileBtn?.classList.add("active");
  backdrop?.classList.add("active");
  document.body.style.overflow = "hidden";

  renderProfileCard();
}

function closeProfileCard(): void {
  const profileCard = document.getElementById("profile-card");
  const profileBtn = document.getElementById("header-profile-btn");
  const backdrop = document.getElementById("header-backdrop");
  const settingsCard = document.getElementById("settings-card");
  if (!profileCard) return;

  profileCard.style.display = "none";
  profileBtn?.classList.remove("active");

  if (settingsCard?.style.display !== "block") {
    backdrop?.classList.remove("active");
    document.body.style.overflow = "";
  }
}

export function setBackdropActive(active: boolean): void {
  const backdrop = document.getElementById("header-backdrop");
  if (active) {
    backdrop?.classList.add("active");
    document.body.style.overflow = "hidden";
  } else {
    backdrop?.classList.remove("active");
    document.body.style.overflow = "";
  }
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
      return;
    }

    renderLoggedInView(profile);
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

function getProviderSvg(provider: string | null): string {
  const svgs: Record<string, string> = {
    google:
      '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><path fill="currentColor" d="M16.318 13.714v5.484h9.078c-.37 2.354-2.745 6.901-9.078 6.901c-5.458 0-9.917-4.521-9.917-10.099s4.458-10.099 9.917-10.099c3.109 0 5.193 1.318 6.38 2.464l4.339-4.182C24.251 1.584 20.641.001 16.318.001c-8.844 0-16 7.151-16 16s7.156 16 16 16c9.234 0 15.365-6.49 15.365-15.635c0-1.052-.115-1.854-.255-2.651z"></path></svg>',
    github:
      '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><path fill="currentColor" d="M16 .396c-8.839 0-16 7.167-16 16c0 7.073 4.584 13.068 10.937 15.183c.803.151 1.093-.344 1.093-.772c0-.38-.009-1.385-.015-2.719c-4.453.964-5.391-2.151-5.391-2.151c-.729-1.844-1.781-2.339-1.781-2.339c-1.448-.989.115-.968.115-.968c1.604.109 2.448 1.645 2.448 1.645c1.427 2.448 3.744 1.74 4.661 1.328c.14-1.031.557-1.74 1.011-2.135c-3.552-.401-7.287-1.776-7.287-7.907c0-1.751.62-3.177 1.645-4.297c-.177-.401-.719-2.031.141-4.235c0 0 1.339-.427 4.4 1.641a15.4 15.4 0 0 1 4-.541c1.36.009 2.719.187 4 .541c3.043-2.068 4.381-1.641 4.381-1.641c.859 2.204.317 3.833.161 4.235c1.015 1.12 1.635 2.547 1.635 4.297c0 6.145-3.74 7.5-7.296 7.891c.556.479 1.077 1.464 1.077 2.959c0 2.14-.02 3.864-.02 4.385c0 .416.28.916 1.104.755c6.4-2.093 10.979-8.093 10.979-15.156c0-8.833-7.161-16-16-16z"></path></svg>',
    yandex:
      '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path fill="currentColor" fill-rule="evenodd" d="M13.5 8a5.5 5.5 0 1 1-11 0a5.5 5.5 0 0 1 11 0M15 8A7 7 0 1 1 1 8a7 7 0 0 1 14 0M6.136 5.103a.75.75 0 0 0-1.272.795l2.044 3.27c.223.357.342.77.342 1.192v1.14a.75.75 0 0 0 1.5 0v-1.14a3.75 3.75 0 0 0-.57-1.987zm5 .795a.75.75 0 1 0-1.272-.795L8.77 6.853a.75.75 0 0 0 1.272.795z" clip-rule="evenodd"></path></svg>',
  };
  return provider ? svgs[provider] || "" : "";
}

function renderLoggedInView(profile: ProfilePublic): void {
  const content = document.getElementById("profile-card");
  if (!content) return;

  const avatarUrl = profileManager.getAvatarUrl(profile);
  const provider = profileManager.getProvider();

  content.innerHTML = "";
  content.appendChild(
    fillTemplate("tpl-pc-profile", {
      avatarUrl,
      displayName: profile.display_name,
      profileId: profile.id,
      createdAt: formatDate(profile.created_at),
    }),
  );

  if (provider) {
    const providerBg = content.querySelector(".pc-provider-bg");
    if (providerBg) {
      providerBg.setAttribute("data-provider", provider);
      providerBg.innerHTML = getProviderSvg(provider);
    }
  }

  initProfileInteractions(profile);
}

function initProfileInteractions(profile: ProfilePublic): void {
  const avatarWrapper = document.getElementById("pc-avatar-img")?.parentElement;
  const avatarInput = document.getElementById(
    "pc-avatar-input",
  ) as HTMLInputElement;
  const nameText = document.getElementById("pc-name-text");
  const nameEdit = document.getElementById("pc-name-edit");
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

  nameEdit?.addEventListener("click", () => {
    if (!nameText || !nameInput) return;
    nameText.style.display = "none";
    nameEdit.style.display = "none";
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
    if (!nameText || !nameInput || !nameEdit) return;
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

    if (result) {
      currentProfile.display_name = result.display_name;
      nameText.textContent = result.display_name;
    }

    cancelNameEdit();
  }

  function cancelNameEdit() {
    if (!nameText || !nameInput || !nameEdit) return;
    nameInput.style.display = "none";
    nameInput.value = "";
    nameText.style.display = "inline";
    nameEdit.style.display = "inline-flex";
  }

  document.getElementById("pc-logout")?.addEventListener("click", async () => {
    await profileManager.logout();
    const profileCard = document.getElementById("profile-card");
    if (profileCard) {
      profileCard.dataset.hasSession = "false";
    }
    renderGuestView();
  });

  document.getElementById("pc-delete")?.addEventListener("click", async () => {
    if (!confirm("Удалить аккаунт? Это действие необратимо.")) return;
    const deleted = await profileManager.deleteProfile();
    if (deleted) {
      const profileCard = document.getElementById("profile-card");
      if (profileCard) {
        profileCard.dataset.hasSession = "false";
      }
      renderGuestView();
    }
  });
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString("ru-RU", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}
