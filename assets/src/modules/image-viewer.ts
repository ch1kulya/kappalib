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
  private pinch: PinchState | null = null;
  private returnFocus: HTMLElement | null = null;

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

  public open(src: string): void {
    this.returnFocus = document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null;
    this.resetPointers();
    this.initialScale = 1;
    this.scale = 1;
    this.tx = 0;
    this.ty = 0;
    this.render();

    this.img.onload = () => {
      this.img.onload = null;
      this.fitToViewport();
    };
    this.img.src = src;

    if (this.img.complete && this.img.naturalWidth) {
      this.img.onload = null;
      this.fitToViewport();
    }

    document.body.style.overflow = "hidden";
    this.viewer.classList.add("open");
    this.viewer.setAttribute("aria-hidden", "false");
    this.closeBtn.focus();
  }

  public close(): void {
    if (!this.isOpen()) return;

    this.resetPointers();
    document.body.style.overflow = "";
    this.viewer.classList.remove("open");
    this.viewer.setAttribute("aria-hidden", "true");
    this.returnFocus?.focus();
    this.returnFocus = null;
  }

  private bindEvents(): void {
    this.viewer.addEventListener("click", (e) => {
      if (e.target === this.viewer) this.close();
    });

    this.closeBtn.addEventListener("click", () => this.close());
    this.zoomInBtn.addEventListener("click", () => this.zoomAtCenter(ImageViewer.ZOOM_STEP));
    this.zoomOutBtn.addEventListener("click", () => this.zoomAtCenter(1 / ImageViewer.ZOOM_STEP));
    this.resetBtn.addEventListener("click", () => this.reset());

    this.img.addEventListener("click", () => this.handleClick());
    this.img.addEventListener("pointerdown", (e) => this.handlePointerDown(e));
    this.img.addEventListener("pointermove", (e) => this.handlePointerMove(e));
    this.img.addEventListener("pointerup", (e) => this.handlePointerEnd(e));
    this.img.addEventListener("pointercancel", (e) => this.handlePointerEnd(e));
    this.img.addEventListener("dragstart", (e) => e.preventDefault());

    this.viewer.addEventListener("wheel", (e) => this.handleWheel(e), { passive: false });
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

    if (!nw || !nh) return;

    this.initialScale = Math.min(1, window.innerWidth / nw, window.innerHeight / nh);
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

  private reset(): void {
    this.scale = this.initialScale;
    this.tx = 0;
    this.ty = 0;
    this.render();
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

  private handleClick(): void {
    if (this.moved) return;

    if (this.isAtInitialScale()) {
      this.scale = this.clamp(this.initialScale * 2);
      this.render();
    } else {
      this.reset();
    }
  }

  private handlePointerDown(e: PointerEvent): void {
    if (e.pointerType === "mouse" && e.button !== 0) return;

    this.img.setPointerCapture(e.pointerId);
    this.pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });

    if (this.pointers.size === 1) {
      this.moved = false;
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

    const next = this.clamp(this.scale * Math.exp(-e.deltaY * 0.0015));
    if (next === this.scale) return;

    const dx = e.clientX - window.innerWidth / 2;
    const dy = e.clientY - window.innerHeight / 2;
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
    const cx = window.innerWidth / 2;
    const cy = window.innerHeight / 2;

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
    return Math.min(Math.max(v, ImageViewer.MIN_SCALE), ImageViewer.MAX_SCALE);
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

    e.preventDefault();
    e.stopPropagation();
    viewer.open(src);
  }, true);

  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && viewer.isOpen()) {
      viewer.close();
    }
  });
}
