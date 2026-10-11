interface PointerInfo {
  x: number;
  y: number;
}

interface PinchState {
  startDist: number;
  startScale: number;
  startTx: number;
  startTy: number;
  startMidX: number;
  startMidY: number;
}

interface ImageViewerElements {
  viewer: HTMLElement;
  img: HTMLImageElement;
  scaleLabel: HTMLElement;
  zoomInBtn: HTMLButtonElement;
  zoomOutBtn: HTMLButtonElement;
  resetBtn: HTMLButtonElement;
  closeBtn: HTMLButtonElement;
}

const VIEWABLE_IMAGES_SELECTOR = [
  ".novel-cover-container img",
  ".chapter-content img",
  ".comment-content img",
].join(", ");

class ImageViewer {
  private static readonly MIN_SCALE = 0.05;
  private static readonly MAX_SCALE = 10;
  private static readonly EPS = 1e-6;
  private static readonly ZOOM_STEP = 1.25;
  private static readonly DRAG_THRESHOLD = 3;
  private static readonly WHEEL_LINE_HEIGHT = 16;

  private readonly viewer: HTMLElement;
  private readonly img: HTMLImageElement;
  private readonly scaleLabel: HTMLElement;
  private readonly zoomInBtn: HTMLButtonElement;
  private readonly zoomOutBtn: HTMLButtonElement;
  private readonly resetBtn: HTMLButtonElement;
  private readonly closeBtn: HTMLButtonElement;

  private scale = 1;
  private tx = 0;
  private ty = 0;
  private initialScale = 1;

  private readonly pointers = new Map<number, PointerInfo>();
  private dragging = false;
  private moved = false;
  private startX = 0;
  private startY = 0;
  private startTx = 0;
  private startTy = 0;
  private pressOnImage = false;
  private pinch: PinchState | null = null;
  private returnFocus: HTMLElement | null = null;
  private readonly inertElements: HTMLElement[] = [];

  constructor(elements: ImageViewerElements) {
    this.viewer = elements.viewer;
    this.img = elements.img;
    this.scaleLabel = elements.scaleLabel;
    this.zoomInBtn = elements.zoomInBtn;
    this.zoomOutBtn = elements.zoomOutBtn;
    this.resetBtn = elements.resetBtn;
    this.closeBtn = elements.closeBtn;

    this.bindEvents();
  }

  public isOpen(): boolean {
    return this.viewer.classList.contains("open");
  }

  public open(src: string, alt: string): void {
    this.returnFocus = document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null;
    this.resetPointers();
    this.initialScale = 1;
    this.scale = 1;
    this.tx = 0;
    this.ty = 0;
    this.render();

    document.body.style.overflow = "hidden";
    this.setPageInert(true);
    this.viewer.classList.add("open");
    this.viewer.setAttribute("aria-hidden", "false");

    this.img.classList.add("is-loading");
    this.img.alt = alt;
    this.img.onload = () => this.handleLoad();
    this.img.onerror = () => this.close();
    this.img.src = src;

    if (this.img.complete && this.img.naturalWidth) {
      this.handleLoad();
    }

    this.viewer.focus();
  }

  public close(): void {
    if (!this.isOpen()) return;

    this.resetPointers();
    document.body.style.overflow = "";
    this.setPageInert(false);
    this.viewer.classList.remove("open");
    this.viewer.setAttribute("aria-hidden", "true");
    this.returnFocus?.focus();
    this.returnFocus = null;
  }

  public handleResize(): void {
    if (this.isOpen() && this.isAtInitialScale()) this.fitToViewport();
  }

  private bindEvents(): void {
    this.closeBtn.addEventListener("click", () => this.close());
    this.zoomInBtn.addEventListener("click", () => this.zoomAtCenter(ImageViewer.ZOOM_STEP));
    this.zoomOutBtn.addEventListener("click", () => this.zoomAtCenter(1 / ImageViewer.ZOOM_STEP));
    this.resetBtn.addEventListener("click", () => this.fitToViewport());

    this.viewer.addEventListener("click", (e) => this.handleClick(e));
    this.viewer.addEventListener("pointerdown", (e) => this.handlePointerDown(e));
    this.viewer.addEventListener("pointermove", (e) => this.handlePointerMove(e));
    this.viewer.addEventListener("pointerup", (e) => this.handlePointerEnd(e));
    this.viewer.addEventListener("pointercancel", (e) => this.handlePointerEnd(e));
    this.viewer.addEventListener("lostpointercapture", (e) => this.handlePointerEnd(e));
    this.viewer.addEventListener("touchstart", (e) => e.stopPropagation(), { passive: true });
    this.viewer.addEventListener("wheel", (e) => this.handleWheel(e), { passive: false });
    this.img.addEventListener("dragstart", (e) => e.preventDefault());
  }

  private setPageInert(inert: boolean): void {
    if (!inert) {
      this.inertElements.forEach((el) => {
        el.inert = false;
      });
      this.inertElements.length = 0;
      return;
    }

    for (const el of this.viewer.parentElement?.children ?? []) {
      if (el instanceof HTMLElement && el !== this.viewer && !el.inert) {
        el.inert = true;
        this.inertElements.push(el);
      }
    }
  }

  private isPanelTarget(target: EventTarget | null): boolean {
    return target instanceof Element && target.closest(".image-viewer-panel") !== null;
  }

  private handleLoad(): void {
    this.img.onload = null;
    this.img.onerror = null;
    this.img.classList.remove("is-loading");
    this.fitToViewport();
  }

  private resetPointers(): void {
    this.pointers.clear();
    this.pinch = null;
    this.dragging = false;
    this.moved = false;
    this.img.classList.remove("is-dragging");
  }

  private fitToViewport(): void {
    const nw = this.img.naturalWidth;
    const nh = this.img.naturalHeight;

    if (nw && nh) {
      this.initialScale = Math.min(1, this.viewer.clientWidth / nw, this.viewer.clientHeight / nh);
    }

    this.scale = this.initialScale;
    this.tx = 0;
    this.ty = 0;
    this.render();
  }

  private render(): void {
    this.img.style.transform = `translate(-50%, -50%) translate(${this.tx}px, ${this.ty}px) scale(${this.scale})`;
    this.scaleLabel.textContent = `${Math.round(this.scale * 100)}%`;
    this.img.classList.toggle("is-zoomed", !this.isAtInitialScale());
  }

  private isAtInitialScale(): boolean {
    return Math.abs(this.scale - this.initialScale) < ImageViewer.EPS;
  }

  private zoomAtCenter(factor: number): void {
    const next = this.clamp(this.scale * factor);
    if (next === this.scale) return;

    const k = next / this.scale;
    this.tx *= k;
    this.ty *= k;
    this.scale = next;
    this.render();
  }

  private handleClick(e: MouseEvent): void {
    if (this.isPanelTarget(e.target) || this.moved) return;

    if (!this.pressOnImage) {
      this.close();
    } else if (this.isAtInitialScale()) {
      this.scale = this.clamp(this.initialScale * 2);
      this.render();
    } else {
      this.fitToViewport();
    }
  }

  private handlePointerDown(e: PointerEvent): void {
    if (this.isPanelTarget(e.target)) {
      this.moved = true;
      return;
    }
    if (e.pointerType === "mouse" && e.button !== 0) return;

    this.viewer.setPointerCapture(e.pointerId);
    this.pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });

    if (this.pointers.size === 1) {
      this.moved = false;
      this.pressOnImage = e.target === this.img;
      this.startDrag(e.clientX, e.clientY);
    } else if (this.pointers.size === 2) {
      this.startPinch();
    } else {
      this.pinch = null;
      this.moved = true;
      this.dragging = false;
      this.img.classList.remove("is-dragging");
    }

    e.preventDefault();
  }

  private handlePointerMove(e: PointerEvent): void {
    if (!this.pointers.has(e.pointerId)) return;
    this.pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });

    if (this.pointers.size === 2 && this.pinch) {
      this.updatePinch();
      return;
    }

    if (this.dragging && this.pointers.size === 1) {
      const dx = e.clientX - this.startX;
      const dy = e.clientY - this.startY;

      if (Math.abs(dx) > ImageViewer.DRAG_THRESHOLD || Math.abs(dy) > ImageViewer.DRAG_THRESHOLD) {
        this.moved = true;
      }

      this.tx = this.startTx + dx;
      this.ty = this.startTy + dy;
      this.render();
    }
  }

  private handlePointerEnd(e: PointerEvent): void {
    if (!this.pointers.has(e.pointerId)) return;

    this.pointers.delete(e.pointerId);

    if (this.pointers.size === 2) {
      this.startPinch();
    } else if (this.pointers.size === 1) {
      const [p] = this.pointers.values();
      this.pinch = null;
      this.moved = true;
      this.startDrag(p.x, p.y);
    } else {
      this.pinch = null;
      this.dragging = false;
      this.img.classList.remove("is-dragging");
    }
  }

  private startDrag(x: number, y: number): void {
    this.dragging = true;
    this.startX = x;
    this.startY = y;
    this.startTx = this.tx;
    this.startTy = this.ty;
    this.pinch = null;
    this.img.classList.add("is-dragging");
  }

  private handleWheel(e: WheelEvent): void {
    e.preventDefault();

    const lineHeight = e.deltaMode === WheelEvent.DOM_DELTA_LINE ? ImageViewer.WHEEL_LINE_HEIGHT : 1;
    const pageHeight = e.deltaMode === WheelEvent.DOM_DELTA_PAGE ? this.viewer.clientHeight : 1;
    const next = this.clamp(this.scale * Math.exp(-e.deltaY * lineHeight * pageHeight * 0.0015));
    if (next === this.scale) return;

    const dx = e.clientX - this.viewer.clientWidth / 2;
    const dy = e.clientY - this.viewer.clientHeight / 2;
    const k = next / this.scale;

    this.tx = dx * (1 - k) + this.tx * k;
    this.ty = dy * (1 - k) + this.ty * k;
    this.scale = next;
    this.render();
  }

  private startPinch(): void {
    const mid = this.getMidpoint();
    this.pinch = {
      startDist: this.getDistance(),
      startScale: this.scale,
      startTx: this.tx,
      startTy: this.ty,
      startMidX: mid.x,
      startMidY: mid.y,
    };
    this.dragging = false;
    this.moved = true;
    this.img.classList.remove("is-dragging");
  }

  private updatePinch(): void {
    if (!this.pinch || !this.pinch.startDist) return;

    const dist = this.getDistance();
    const mid = this.getMidpoint();
    const newScale = this.clamp(this.pinch.startScale * (dist / this.pinch.startDist));
    const k = newScale / this.pinch.startScale;
    const cx = this.viewer.clientWidth / 2;
    const cy = this.viewer.clientHeight / 2;

    this.tx = (mid.x - cx) - (this.pinch.startMidX - cx) * k + this.pinch.startTx * k;
    this.ty = (mid.y - cy) - (this.pinch.startMidY - cy) * k + this.pinch.startTy * k;
    this.scale = newScale;
    this.render();
  }

  private getDistance(): number {
    const [a, b] = this.pointers.values();
    return Math.hypot(a.x - b.x, a.y - b.y);
  }

  private getMidpoint(): PointerInfo {
    const [a, b] = this.pointers.values();
    return { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 };
  }

  private clamp(v: number): number {
    return Math.min(Math.max(v, Math.min(ImageViewer.MIN_SCALE, this.initialScale)), ImageViewer.MAX_SCALE);
  }
}

function findViewableImage(target: EventTarget | null): HTMLImageElement | null {
  if (!(target instanceof HTMLImageElement)) return null;
  if (!target.matches(VIEWABLE_IMAGES_SELECTOR) || target.closest("a")) return null;
  return target;
}

export function initImageViewer(): void {
  const viewerEl = document.getElementById("image-viewer");
  const img = document.getElementById("image-viewer-img") as HTMLImageElement | null;
  const scaleLabel = document.getElementById("image-viewer-scale");
  const zoomInBtn = document.getElementById("image-viewer-zoom-in") as HTMLButtonElement | null;
  const zoomOutBtn = document.getElementById("image-viewer-zoom-out") as HTMLButtonElement | null;
  const resetBtn = document.getElementById("image-viewer-reset") as HTMLButtonElement | null;
  const closeBtn = document.getElementById("image-viewer-close") as HTMLButtonElement | null;

  if (!viewerEl || !img || !scaleLabel || !zoomInBtn || !zoomOutBtn || !resetBtn || !closeBtn) {
    return;
  }

  const viewer = new ImageViewer({
    viewer: viewerEl,
    img,
    scaleLabel,
    zoomInBtn,
    zoomOutBtn,
    resetBtn,
    closeBtn,
  });

  document.addEventListener("click", (e) => {
    const image = findViewableImage(e.target);
    if (!image) return;

    const src = image.currentSrc || image.src;
    if (!src) return;

    if (image.closest(".spoiler")) e.stopPropagation();
    viewer.open(src, image.alt);
  }, true);

  window.addEventListener("keydown", (e) => {
    if (!viewer.isOpen()) return;

    e.stopPropagation();
    if (e.key === "Escape") viewer.close();
  }, true);

  window.addEventListener("resize", () => viewer.handleResize());
}
