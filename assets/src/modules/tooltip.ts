const TOOLTIP_ID = "tooltip";
const TOOLTIP_MARGIN = 8;
const TOOLTIP_OFFSET = 6;
const TOOLTIP_HIDE_DELAY = 80;

const scopes = new Set<HTMLElement>();
let tooltip: HTMLDivElement | null = null;
let activeTarget: HTMLElement | null = null;
let hideTimer = 0;

export function hideTooltip(): void {
  window.clearTimeout(hideTimer);
  activeTarget?.classList.remove("active");
  activeTarget?.removeAttribute("aria-describedby");
  activeTarget = null;
  tooltip?.classList.remove("visible");
}

function scheduleHide(): void {
  window.clearTimeout(hideTimer);
  hideTimer = window.setTimeout(hideTooltip, TOOLTIP_HIDE_DELAY);
}

function ensureTooltip(): HTMLDivElement {
  if (tooltip) return tooltip;

  tooltip = document.createElement("div");
  tooltip.className = "tooltip";
  tooltip.id = TOOLTIP_ID;
  tooltip.setAttribute("role", "tooltip");
  document.body.appendChild(tooltip);

  window.addEventListener("scroll", hideTooltip, { passive: true });
  window.addEventListener("resize", hideTooltip);
  document.addEventListener("pointerdown", (e) => {
    const inside = Array.from(scopes).some((scope) => scope.contains(e.target as Node));
    if (!inside) hideTooltip();
  });

  return tooltip;
}

function showTooltip(target: HTMLElement): void {
  const el = ensureTooltip();
  window.clearTimeout(hideTimer);
  if (activeTarget === target) return;
  activeTarget?.classList.remove("active");
  activeTarget?.removeAttribute("aria-describedby");
  activeTarget = target;
  target.classList.add("active");
  target.setAttribute("aria-describedby", TOOLTIP_ID);

  el.replaceChildren();
  const title = target.dataset.tipTitle;
  if (title) {
    const titleEl = document.createElement("span");
    titleEl.className = "tooltip-title";
    titleEl.textContent = title;
    el.appendChild(titleEl);
  }
  el.appendChild(document.createTextNode(target.dataset.tip || ""));
  el.classList.add("visible");

  const targetRect = target.getBoundingClientRect();
  const tipRect = el.getBoundingClientRect();
  const maxLeft = window.innerWidth - tipRect.width - TOOLTIP_MARGIN;
  const left = Math.max(
    TOOLTIP_MARGIN,
    Math.min(targetRect.left + targetRect.width / 2 - tipRect.width / 2, maxLeft),
  );
  let top = targetRect.top - tipRect.height - TOOLTIP_OFFSET;
  if (top < TOOLTIP_MARGIN) {
    top = targetRect.bottom + TOOLTIP_OFFSET;
  }

  el.style.left = `${left}px`;
  el.style.top = `${top}px`;
}

function findTarget(scope: HTMLElement, node: EventTarget | null): HTMLElement | null {
  if (!(node instanceof Element)) return null;
  const target = node.closest<HTMLElement>("[data-tip]");
  return target && scope.contains(target) ? target : null;
}

export function initTooltips(elements: Iterable<HTMLElement>): void {
  for (const scope of elements) {
    if (scopes.has(scope)) continue;
    scopes.add(scope);
    ensureTooltip();

    scope.addEventListener("pointerover", (e) => {
      const target = findTarget(scope, e.target);
      if (target) showTooltip(target);
    });
    scope.addEventListener("pointerout", (e) => {
      if (e.pointerType === "mouse" && !findTarget(scope, e.relatedTarget)) {
        scheduleHide();
      }
    });
    scope.addEventListener("focusin", (e) => {
      const target = findTarget(scope, e.target);
      if (target) showTooltip(target);
    });
    scope.addEventListener("focusout", hideTooltip);
  }
}

export function initHelpTooltips(): void {
  initTooltips(document.querySelectorAll<HTMLElement>(".btn-help[data-tip]"));
}
