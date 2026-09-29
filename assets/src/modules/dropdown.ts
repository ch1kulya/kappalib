function closeDropdown(root: HTMLElement): void {
  root.classList.remove("active");
  root.querySelector(".dropdown-btn")?.setAttribute("aria-expanded", "false");
}

document.addEventListener("click", (e: Event) => {
  document
    .querySelectorAll<HTMLElement>(".dropdown.active")
    .forEach((root) => {
      if (!root.contains(e.target as Node)) closeDropdown(root);
    });
});

export default class Dropdown {
  root: HTMLElement;
  trigger: HTMLElement | null;
  menu: HTMLElement | null;
  items: NodeListOf<HTMLElement>;
  labelSpan: HTMLElement | null;

  constructor(element: HTMLElement) {
    this.root = element;
    this.trigger = this.root.querySelector<HTMLElement>(".dropdown-btn");
    this.menu = this.root.querySelector<HTMLElement>(".dropdown-menu");
    this.items = this.root.querySelectorAll<HTMLElement>(".dropdown-item");
    this.labelSpan = this.root.querySelector<HTMLElement>(".js-dropdown-label");

    if (!this.trigger || !this.menu) return;

    this.init();
  }

  private init(): void {
    this.trigger?.addEventListener("click", () => {
      this.toggle();
    });

    this.items.forEach((item) => {
      item.addEventListener("click", (e: Event) => {
        if (item.dataset.noSelect === "true") return;
        e.preventDefault();
        this.select(item);
      });
    });

    this.root.addEventListener("keydown", (e: KeyboardEvent) => {
      if (e.key === "Escape") this.close();
    });
  }

  public toggle(): void {
    this.root.classList.contains("active") ? this.close() : this.open();
  }

  public open(): void {
    this.root.classList.add("active");
    this.trigger?.setAttribute("aria-expanded", "true");
  }

  public close(): void {
    closeDropdown(this.root);
  }

  private select(item: HTMLElement): void {
    this.items.forEach((i) => {
      i.classList.remove("selected");
      i.setAttribute("aria-selected", "false");
    });
    item.classList.add("selected");
    item.setAttribute("aria-selected", "true");

    if (this.labelSpan) {
      const span = item.querySelector<HTMLElement>("span");
      const text = span ? span.innerText : item.innerText;
      this.labelSpan.innerText = text;
    }

    const value = item.getAttribute("data-value");

    this.root.dispatchEvent(
      new CustomEvent("change", {
        detail: { value },
        bubbles: true,
      }),
    );

    this.close();
  }
}
