const API_URL = process.env.API_URL;
const PROFILE_ID_KEY = "kappalib_profile_id";
const SECRET_TOKEN_KEY = "kappalib_secret_token";
const TURNSTILE_SITE_KEY = process.env.TURNSTILE_SITE_KEY || "";

interface CookieValue {
  value: string;
  updated_at: number;
}

interface ProfilePublic {
  id: string;
  display_name: string;
  avatar_seed: string;
  created_at: string;
}

interface ProfileWithToken extends ProfilePublic {
  secret_token: string;
}

interface LoginResponse {
  profile: ProfilePublic;
  secret_token: string;
  cookies: Record<string, CookieValue>;
}

class ProfileManager {
  private profileId: string | null = null;
  private secretToken: string | null = null;

  constructor() {
    this.profileId = localStorage.getItem(PROFILE_ID_KEY);
    this.secretToken = localStorage.getItem(SECRET_TOKEN_KEY);
  }

  isLoggedIn(): boolean {
    return this.profileId !== null && this.secretToken !== null;
  }

  getProfileId(): string | null {
    return this.profileId;
  }

  async createProfile(turnstileToken: string): Promise<ProfilePublic | null> {
    try {
      const res = await fetch(`${API_URL}/profile`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ turnstile_token: turnstileToken }),
      });
      if (res.ok) {
        const data: ProfileWithToken = await res.json();
        this.profileId = data.id;
        this.secretToken = data.secret_token;
        localStorage.setItem(PROFILE_ID_KEY, data.id);
        localStorage.setItem(SECRET_TOKEN_KEY, data.secret_token);
        this.syncCookiesToServer();
        return data;
      }
    } catch (err) {
      console.error("Create profile failed", err);
    }
    return null;
  }

  async login(syncCode: string): Promise<LoginResponse | null> {
    try {
      const res = await fetch(`${API_URL}/profile/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sync_code: syncCode }),
      });
      if (res.ok) {
        const data: LoginResponse = await res.json();
        this.profileId = data.profile.id;
        this.secretToken = data.secret_token;
        localStorage.setItem(PROFILE_ID_KEY, data.profile.id);
        localStorage.setItem(SECRET_TOKEN_KEY, data.secret_token);
        this.applyCookies(data.cookies);
        return data;
      }
    } catch (err) {
      console.error("Login failed", err);
    }
    return null;
  }

  async fetchProfile(): Promise<ProfilePublic | null> {
    if (!this.profileId) return null;
    try {
      const res = await fetch(`${API_URL}/profile/${this.profileId}`);
      if (res.ok) return await res.json();
      if (res.status === 404) this.logout();
    } catch (err) {
      console.error("Fetch profile failed", err);
    }
    return null;
  }

  async generateSyncCode(): Promise<{
    sync_code: string;
    expires_at: string;
  } | null> {
    if (!this.profileId || !this.secretToken) return null;
    try {
      const res = await fetch(
        `${API_URL}/profile/${this.profileId}/sync-code`,
        {
          method: "POST",
          headers: { "X-Secret-Token": this.secretToken },
        },
      );
      if (res.ok) return await res.json();
      if (res.status === 403) this.logout();
    } catch (err) {
      console.error("Generate sync code failed", err);
    }
    return null;
  }

  async syncCookiesToServer(): Promise<void> {
    if (!this.profileId || !this.secretToken) return;
    const cookies = this.getKappalibCookies();
    try {
      const res = await fetch(`${API_URL}/profile/sync-cookies`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Profile-ID": this.profileId,
          "X-Secret-Token": this.secretToken,
        },
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

  async deleteProfile(): Promise<boolean> {
    if (!this.profileId || !this.secretToken) return false;
    try {
      const res = await fetch(`${API_URL}/profile/${this.profileId}`, {
        method: "DELETE",
        headers: { "X-Secret-Token": this.secretToken },
      });
      if (res.ok) {
        this.logout();
        return true;
      }
      if (res.status === 403) this.logout();
    } catch (err) {
      console.error("Delete profile failed", err);
    }
    return false;
  }

  logout(): void {
    this.profileId = null;
    this.secretToken = null;
    localStorage.removeItem(PROFILE_ID_KEY);
    localStorage.removeItem(SECRET_TOKEN_KEY);
  }

  private getKappalibCookies(): Record<string, CookieValue> {
    const cookies: Record<string, CookieValue> = {};
    document.cookie.split(";").forEach((c) => {
      const [name, value] = c.trim().split("=");
      if (name && name.startsWith("kappalib_") && value) {
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
      if (name.startsWith("kappalib_")) {
        document.cookie = `${name}=${cv.value}; path=/; max-age=31536000; SameSite=Lax`;
        localStorage.setItem(`${name}_updated_at`, cv.updated_at.toString());
      }
    }
  }
}

export const profileManager = new ProfileManager();

export function initProfile(): void {
  if (profileManager.isLoggedIn()) {
    profileManager.syncCookiesToServer();
  }
}

export function setKappalibCookie(name: string, value: string): void {
  if (!name.startsWith("kappalib_")) {
    name = `kappalib_${name}`;
  }
  const timestamp = Date.now();
  document.cookie = `${name}=${value}; path=/; max-age=31536000; SameSite=Lax`;
  localStorage.setItem(`${name}_updated_at`, timestamp.toString());

  if (profileManager.isLoggedIn()) {
    profileManager.syncCookiesToServer();
  }
}

export function initProfileModal(): void {
  const overlay = document.getElementById("profile-modal-overlay");
  const profileBtn = document.getElementById("header-profile-btn");

  if (!overlay || !profileBtn) return;

  profileBtn.addEventListener("click", (e) => {
    e.preventDefault();
    openProfileModal();
  });

  overlay.addEventListener("click", (e) => {
    if (e.target === overlay) closeProfileModal();
  });

  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && overlay.style.display === "flex") {
      closeProfileModal();
    }
  });
}

function openProfileModal(): void {
  const overlay = document.getElementById("profile-modal-overlay");
  if (!overlay) return;

  overlay.style.display = "flex";
  document.body.style.overflow = "hidden";

  if (profileManager.isLoggedIn()) {
    renderLoggedInView();
  } else {
    renderGuestView();
  }
}

function closeProfileModal(): void {
  const overlay = document.getElementById("profile-modal-overlay");
  if (!overlay) return;

  overlay.style.display = "none";
  document.body.style.overflow = "";

  const container = document.getElementById("turnstile-container");
  if (container) container.innerHTML = "";
}

function renderGuestView(): void {
  const content = document.getElementById("profile-modal-content");
  if (!content) return;

  content.innerHTML = `
    <div class="pm-header">
      <h3>Аккаунт</h3>
      <button class="pm-close" id="pm-close">&times;</button>
    </div>
    <div class="pm-body">
      <div class="pm-section">
        <p class="pm-desc">Создайте аккаунт для синхронизации прогресса между устройствами</p>
        <div id="turnstile-container" class="turnstile-container"></div>
        <button class="pm-btn pm-btn-primary" id="pm-create" disabled>Создать аккаунт</button>
      </div>
      <div class="pm-divider"><span>или</span></div>
      <div class="pm-section">
        <p class="pm-desc">Уже есть аккаунт?</p>
        <div class="pm-input-row">
          <input type="text" class="pm-input" id="pm-sync-input" placeholder="КОД" maxlength="8" />
          <button class="pm-btn pm-btn-outline" id="pm-login">Войти</button>
        </div>
        <div id="pm-error" class="pm-error"></div>
      </div>
    </div>
  `;

  document
    .getElementById("pm-close")
    ?.addEventListener("click", closeProfileModal);

  loadTurnstile();

  document.getElementById("pm-create")?.addEventListener("click", async () => {
    const token = (window as any).turnstileToken;
    if (!token) return;

    const btn = document.getElementById("pm-create") as HTMLButtonElement;
    btn.disabled = true;
    btn.textContent = "Создание...";

    const profile = await profileManager.createProfile(token);
    if (profile) {
      renderLoggedInView();
    } else {
      btn.disabled = false;
      btn.textContent = "Создать аккаунт";
      showError("Ошибка создания аккаунта");
    }
  });

  document.getElementById("pm-login")?.addEventListener("click", async () => {
    const input = document.getElementById("pm-sync-input") as HTMLInputElement;
    const code = input.value.trim().toUpperCase();
    if (code.length !== 8) {
      showError("Введите 8-символьный код");
      return;
    }

    const btn = document.getElementById("pm-login") as HTMLButtonElement;
    btn.disabled = true;

    const result = await profileManager.login(code);
    if (result) {
      renderLoggedInView();
    } else {
      btn.disabled = false;
      showError("Неверный или просроченный код");
    }
  });
}

function renderLoggedInView(): void {
  const content = document.getElementById("profile-modal-content");
  if (!content) return;

  content.innerHTML = `
    <div class="pm-header">
      <h3>Профиль</h3>
      <button class="pm-close" id="pm-close">&times;</button>
    </div>
    <div class="pm-body pm-loading">
      <div class="pm-loader"></div>
    </div>
  `;

  document
    .getElementById("pm-close")
    ?.addEventListener("click", closeProfileModal);

  profileManager.fetchProfile().then((profile) => {
    if (!profile) {
      renderGuestView();
      return;
    }

    const body = content.querySelector(".pm-body");
    if (!body) return;

    const avatarUrl = `https://api.dicebear.com/9.x/lorelei-neutral/svg?seed=${profile.avatar_seed}&scale=110&backgroundColor=ffdfbf,d1d4f9,ffd5dc,c0aede,b6e3f4,ffffff&backgroundType=solid,gradientLinear`;

    body.classList.remove("pm-loading");
    body.innerHTML = `
      <div class="pm-profile">
        <img src="${avatarUrl}" alt="" class="pm-avatar" />
        <div class="pm-info">
          <div class="pm-name">${profile.display_name}</div>
          <div class="pm-meta">
            <span class="pm-id">${profile.id}</span>
            <span class="pm-date">${formatDate(profile.created_at)}</span>
          </div>
        </div>
      </div>
      <div class="pm-section">
        <p class="pm-desc">Код для входа на другом устройстве</p>
        <div id="pm-code-area"></div>
        <button class="pm-btn pm-btn-primary" id="pm-get-code">Получить код</button>
      </div>
      <div class="pm-footer">
        <button class="pm-btn pm-btn-text" id="pm-logout">Выйти</button>
        <button class="pm-btn pm-btn-danger-text" id="pm-delete">Удалить аккаунт</button>
      </div>
    `;

    document
      .getElementById("pm-get-code")
      ?.addEventListener("click", async () => {
        const btn = document.getElementById("pm-get-code") as HTMLButtonElement;
        btn.disabled = true;
        btn.textContent = "Генерация...";

        const result = await profileManager.generateSyncCode();
        if (result) {
          const area = document.getElementById("pm-code-area");
          if (area) {
            area.innerHTML = `<div class="pm-code">${result.sync_code}</div>`;
          }
          btn.style.display = "none";
        } else {
          btn.disabled = false;
          btn.textContent = "Получить код";
        }
      });

    document.getElementById("pm-logout")?.addEventListener("click", () => {
      profileManager.logout();
      renderGuestView();
    });

    document
      .getElementById("pm-delete")
      ?.addEventListener("click", async () => {
        if (!confirm("Удалить аккаунт? Это действие необратимо.")) return;
        const deleted = await profileManager.deleteProfile();
        if (deleted) {
          renderGuestView();
        }
      });
  });
}

function loadTurnstile(): void {
  const container = document.getElementById("turnstile-container");
  const createBtn = document.getElementById("pm-create") as HTMLButtonElement;
  if (!container || !createBtn) return;

  (window as any).turnstileToken = null;

  if ((window as any).turnstile) {
    renderTurnstile();
  } else {
    const script = document.createElement("script");
    script.src =
      "https://challenges.cloudflare.com/turnstile/v0/api.js?onload=onTurnstileLoad";
    script.async = true;
    (window as any).onTurnstileLoad = renderTurnstile;
    document.head.appendChild(script);
  }
}

function renderTurnstile(): void {
  const container = document.getElementById("turnstile-container");
  const createBtn = document.getElementById("pm-create") as HTMLButtonElement;
  if (!container || !createBtn || !TURNSTILE_SITE_KEY) return;

  (window as any).turnstile.render(container, {
    sitekey: TURNSTILE_SITE_KEY,
    callback: (token: string) => {
      (window as any).turnstileToken = token;
      createBtn.disabled = false;
    },
    "expired-callback": () => {
      (window as any).turnstileToken = null;
      createBtn.disabled = true;
    },
  });
}

function showError(msg: string): void {
  const el = document.getElementById("pm-error");
  if (el) el.textContent = msg;
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString("ru-RU", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}
