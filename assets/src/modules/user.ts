import { bindProfileEditor, profileManager } from "./profile";
import { hideTooltip, initTooltips } from "./tooltip";

function initHeatmapFade(page: HTMLElement): void {
  const scroll = page.querySelector<HTMLElement>(".up-heatmap-scroll");
  const content = scroll?.firstElementChild;
  if (!scroll || !content) return;

  const update = (): void => {
    const outer = scroll.getBoundingClientRect();
    const inner = content.getBoundingClientRect();
    scroll.classList.toggle("fade-start", inner.left < outer.left - 1);
    scroll.classList.toggle("fade-end", inner.right > outer.right + 1);
  };

  update();
  scroll.addEventListener("scroll", update, { passive: true });
  scroll.addEventListener("scroll", hideTooltip, { passive: true });

  if (typeof ResizeObserver !== "undefined") {
    new ResizeObserver(() => requestAnimationFrame(update)).observe(scroll);
  }
}

function initOwnerEditor(page: HTMLElement): void {
  if (page.dataset.owner !== "true") return;

  const nameText = document.getElementById("up-name-text");

  bindProfileEditor(
    {
      avatarWrapper: document.getElementById("up-avatar-wrapper"),
      avatarImg: document.getElementById("up-avatar-img") as HTMLImageElement | null,
      avatarOverlay: document.getElementById("up-avatar-overlay"),
      avatarInput: document.getElementById("up-avatar-input") as HTMLInputElement | null,
      nameText,
      nameInput: document.getElementById("up-name-input") as HTMLInputElement | null,
      meta: document.getElementById("up-meta"),
    },
    nameText?.textContent?.trim() || "",
    (profile) => {
      const avatarUrl = profileManager.getAvatarUrl(profile);
      page
        .querySelectorAll<HTMLImageElement>(`.comment-avatar-link[href="/${profile.id}"] img`)
        .forEach((img) => {
          img.src = avatarUrl;
        });
      page
        .querySelectorAll(`.comment-author-link[href="/${profile.id}"]`)
        .forEach((el) => {
          el.textContent = profile.display_name;
        });
    },
  );
}

export function initUserProfilePage(): void {
  const page = document.querySelector<HTMLElement>(".user-profile-page");
  if (!page) return;

  initTooltips(page.querySelectorAll<HTMLElement>(".up-heatmap-grid, .up-achievements"));
  initHeatmapFade(page);
  initOwnerEditor(page);
}
