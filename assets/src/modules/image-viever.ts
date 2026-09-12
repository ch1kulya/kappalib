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
  viewer: HTMLDivElement;
  img: HTMLImageElement;
  scaleLabel: HTMLSpanElement;
  zoomInBtn: HTMLButtonElement;
  zoomOutBtn: HTMLButtonElement;
  resetBtn: HTMLButtonElement;
  closeBtn: HTMLButtonElement;
}

class ImageViewer {
  private static readonly MIN_SCALE = 0.05;
  private static readonly MAX_SCALE = 10;
  private static readonly EPS = 1e-6;
  private static readonly ZOOM_STEP = 1.25;

  private readonly viewer: HTMLDivElement;
  private readonly img: HTMLImageElement;
  private readonly scaleLabel: HTMLSpanElement;
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

  public attach(element: HTMLElement): void {
    element.style.cursor = 'pointer'
    element.addEventListener('click', () => {
      const src = element instanceof HTMLImageElement
        ? element.src
        : element.getAttribute('data-src') ?? '';

      if (src) this.open(src);
    });
  }

  private bindEvents(): void {
    this.viewer.addEventListener('click', (e: MouseEvent) => {
      if (e.target === this.viewer) this.close();
    });

    this.closeBtn.addEventListener('click', () => {
      this.close();
    });

    this.zoomInBtn.addEventListener('click', () => {
      this.zoomAtCenter(ImageViewer.ZOOM_STEP);
    });

    this.zoomOutBtn.addEventListener('click', () => {
      this.zoomAtCenter(1 / ImageViewer.ZOOM_STEP);
    });

    this.resetBtn.addEventListener('click', () => {
      this.reset();
    });

    this.img.addEventListener('click', () => {
      this.handleClick();
    });

    this.img.addEventListener('pointerdown', (e: PointerEvent) => {
      this.handlePointerDown(e);
    });

    this.img.addEventListener('pointermove', (e: PointerEvent) => {
      this.handlePointerMove(e);
    });

    this.img.addEventListener('pointerup', (e: PointerEvent) => {
      this.handlePointerEnd(e);
    });

    this.img.addEventListener('pointercancel', (e: PointerEvent) => {
      this.handlePointerEnd(e);
    });

    this.img.addEventListener('wheel', (e: WheelEvent) => {
      this.handleWheel(e);
    }, { passive: false });

    this.img.addEventListener('dragstart', (e: DragEvent) => {
      e.preventDefault();
    });
  }

  private open(src: string): void {
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
      this.fitToViewport();
    }

    document.body.style.overflow = 'hidden'
    this.viewer.classList.add('open');
  }

  private close(): void {
    document.body.style.overflow = ''
    this.viewer.classList.remove('open');
  }

  private fitToViewport(): void {
    const vw = window.innerWidth;
    const vh = window.innerHeight;
    const nw = this.img.naturalWidth;
    const nh = this.img.naturalHeight;

    if (!nw || !nh) return;

    const fit = Math.min(1, vw / nw, vh / nh);
    this.initialScale = fit < 1 ? fit : 1;
    this.scale = this.initialScale;
    this.tx = 0;
    this.ty = 0;
    this.render();
  }

  private render(): void {
    this.img.style.transform =
      'translate(-50%, -50%) translate(' + this.tx + 'px, ' + this.ty + 'px) scale(' + this.scale + ')';
    this.scaleLabel.textContent = Math.round(this.scale * 100) + '%';

    const atInitial = Math.abs(this.scale - this.initialScale) < ImageViewer.EPS;
    this.img.classList.toggle('is-zoomed', !atInitial);
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

    if (Math.abs(this.scale - this.initialScale) < ImageViewer.EPS) {
      this.scale = this.initialScale * 2;
    } else {
      this.scale = this.initialScale;
      this.tx = 0;
      this.ty = 0;
    }
    this.render();
  }

  private handlePointerDown(e: PointerEvent): void {
    if (e.pointerType === 'mouse' && e.button !== 0) return;

    this.img.setPointerCapture(e.pointerId);
    this.pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });

    if (this.pointers.size === 1) {
      this.dragging = true;
      this.moved = false;
      this.startX = e.clientX;
      this.startY = e.clientY;
      this.startTx = this.tx;
      this.startTy = this.ty;
      this.img.classList.add('is-dragging');
      this.pinch = null;
    } else if (this.pointers.size === 2) {
      this.startPinch();
    } else {
      this.pinch = null;
      this.moved = true;
      this.dragging = false;
      this.img.classList.remove('is-dragging');
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

      if (Math.abs(dx) > 3 || Math.abs(dy) > 3) this.moved = true;

      this.tx = this.startTx + dx;
      this.ty = this.startTy + dy;
      this.render();
    }
  }

  private handlePointerEnd(e: PointerEvent): void {
    if (!this.pointers.has(e.pointerId)) return;

    this.pointers.delete(e.pointerId);
    try { this.img.releasePointerCapture(e.pointerId); } catch { /* noop */ }

    if (this.pointers.size === 2) {
      this.startPinch();
    } else if (this.pointers.size === 1) {
      this.pinch = null;
      const p = [...this.pointers.values()][0];
      this.dragging = true;
      this.moved = true;
      this.startX = p.x;
      this.startY = p.y;
      this.startTx = this.tx;
      this.startTy = this.ty;
      this.img.classList.add('is-dragging');
    } else {
      this.pinch = null;
      this.dragging = false;
      this.img.classList.remove('is-dragging');
    }
  }

  private handleWheel(e: WheelEvent): void {
    e.preventDefault();

    const next = this.clamp(this.scale * Math.exp(-e.deltaY * 0.0015));
    if (next === this.scale) return;

    const vx = window.innerWidth / 2;
    const vy = window.innerHeight / 2;
    const dx = e.clientX - vx;
    const dy = e.clientY - vy;
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
      startMidY: mid.y
    };
    this.dragging = false;
    this.moved = true;
    this.img.classList.remove('is-dragging');
  }

  private updatePinch(): void {
    if (!this.pinch) return;

    const dist = this.getDistance();
    const mid = this.getMidpoint();

    const rawScale = this.pinch.startScale * (dist / this.pinch.startDist);
    const newScale = this.clamp(rawScale);
    const k = newScale / this.pinch.startScale;

    const vw = window.innerWidth;
    const vh = window.innerHeight;

    this.tx = (mid.x - vw / 2) - (this.pinch.startMidX - vw / 2) * k + this.pinch.startTx * k;
    this.ty = (mid.y - vh / 2) - (this.pinch.startMidY - vh / 2) * k + this.pinch.startTy * k;
    this.scale = newScale;

    this.render();
    this.moved = true;
  }

  private getDistance(): number {
    const pts = [...this.pointers.values()];
    const dx = pts[0].x - pts[1].x;
    const dy = pts[0].y - pts[1].y;
    return Math.hypot(dx, dy);
  }

  private getMidpoint(): { x: number; y: number } {
    const pts = [...this.pointers.values()];
    return {
      x: (pts[0].x + pts[1].x) / 2,
      y: (pts[0].y + pts[1].y) / 2
    };
  }

  private clamp(v: number): number {
    return Math.min(Math.max(v, ImageViewer.MIN_SCALE), ImageViewer.MAX_SCALE);
  }
}

const viewer = new ImageViewer({
  viewer: document.getElementById('viewer-id') as HTMLDivElement,
  img: document.getElementById('viewer-img-id') as HTMLImageElement,
  scaleLabel: document.getElementById('viewer-scale-id') as HTMLSpanElement,
  zoomInBtn: document.getElementById('viewer-zoom-in') as HTMLButtonElement,
  zoomOutBtn: document.getElementById('viewer-zoom-out') as HTMLButtonElement,
  resetBtn: document.getElementById('viewer-reset') as HTMLButtonElement,
  closeBtn: document.getElementById('viewer-close') as HTMLButtonElement
});

export function initImageViewer() {
  const poster: HTMLElement | null = document.querySelector('.poster-wrapper img')
  const commentsList: HTMLElement | null = document.querySelector('.comments-list')
  const chapterContent: HTMLElement | null = document.querySelector('.chapter-content')
  
  const alreadyLoadedImages: HTMLElement[] = []
  
  if (poster) {
    alreadyLoadedImages.push(poster)
  }
  if (chapterContent) {
    alreadyLoadedImages.push(...findImagesIn(chapterContent))
  }
  if (commentsList) {
    alreadyLoadedImages.push(...findImagesIn(commentsList))
    observeDomMutations(commentsList, handleMutations)
  }
  alreadyLoadedImages.forEach((image) =>
    viewer.attach(image)
  )
}

function observeDomMutations(
  target: Element,
  onMutate: (mutations: MutationRecord[]) => void
) {
  const observer = new MutationObserver(onMutate)
  
  observer.observe(target, {
    childList: true,
    subtree: true,
  });
}

function findImagesIn(node: Node | Element): HTMLImageElement[] {
  if (node instanceof HTMLImageElement) return [node]
  return node instanceof Element
    ? Array.from(node.querySelectorAll('img'))
    : []
}

function handleMutations(mutations: MutationRecord[]) {
  mutations.forEach(mutation => {
    mutation.addedNodes.forEach((node) => {
      const images = findImagesIn(node)
      images.forEach((image) =>
        viewer.attach(image)
      )
    })
  })
}