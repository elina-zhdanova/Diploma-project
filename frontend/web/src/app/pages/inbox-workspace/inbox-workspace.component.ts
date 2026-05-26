import { Component, ElementRef, HostListener, inject, viewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NavigationEnd, Router, RouterOutlet } from '@angular/router';
import { filter } from 'rxjs/operators';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { InboxComponent } from '../inbox/inbox.component';

const STORAGE_KEY = 'itshop.inbox.detailPaneWidthPx';
const DEFAULT_DETAIL_W = 380;
const MIN_DETAIL_W = 260;
const MIN_LIST_W = 200;

@Component({
  selector: 'app-inbox-workspace',
  standalone: true,
  imports: [CommonModule, RouterOutlet, InboxComponent],
  templateUrl: './inbox-workspace.component.html',
  styleUrls: ['./inbox-workspace.component.scss', '../requests-workspace/requests-workspace.component.scss'],
})
export class InboxWorkspaceComponent {
  private readonly router = inject(Router);

  readonly gridRef = viewChild<ElementRef<HTMLElement>>('gridRef');

  /** В URL выбрана заявка (`/inbox/:id`) — показываем колонку деталей. */
  hasDetail = false;

  detailWidthPx = DEFAULT_DETAIL_W;

  private dragStartX = 0;
  private dragStartWidth = 0;

  constructor() {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      const n = parseInt(saved, 10);
      if (!Number.isNaN(n) && n >= MIN_DETAIL_W) {
        this.detailWidthPx = n;
      }
    }

    this.router.events
      .pipe(
        filter((e): e is NavigationEnd => e instanceof NavigationEnd),
        takeUntilDestroyed(),
      )
      .subscribe(() => this.applyDetailFlag());
    this.applyDetailFlag();
  }

  @HostListener('window:resize')
  onWindowResize(): void {
    if (!this.hasDetail) {
      return;
    }
    this.detailWidthPx = this.clampDetailWidth(this.detailWidthPx);
  }

  onResizerPointerDown(event: PointerEvent): void {
    if (event.pointerType === 'mouse' && event.button !== 0) {
      return;
    }
    event.preventDefault();
    const el = event.currentTarget as HTMLElement;
    el.setPointerCapture(event.pointerId);

    this.dragStartX = event.clientX;
    this.dragStartWidth = this.detailWidthPx;

    const onMove = (e: PointerEvent) => {
      const delta = e.clientX - this.dragStartX;
      this.detailWidthPx = this.clampDetailWidth(this.dragStartWidth + delta);
    };
    const onUp = (e: PointerEvent) => {
      try {
        el.releasePointerCapture(e.pointerId);
      } catch {
        /* ignore */
      }
      el.removeEventListener('pointermove', onMove);
      el.removeEventListener('pointerup', onUp);
      el.removeEventListener('pointercancel', onUp);
      document.body.style.removeProperty('cursor');
      document.body.style.removeProperty('user-select');
      localStorage.setItem(STORAGE_KEY, String(this.detailWidthPx));
    };

    el.addEventListener('pointermove', onMove);
    el.addEventListener('pointerup', onUp);
    el.addEventListener('pointercancel', onUp);
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
  }

  onResizerKeydown(event: KeyboardEvent): void {
    const step = 24;
    if (event.key === 'ArrowLeft') {
      event.preventDefault();
      this.detailWidthPx = this.clampDetailWidth(this.detailWidthPx - step);
      localStorage.setItem(STORAGE_KEY, String(this.detailWidthPx));
    } else if (event.key === 'ArrowRight') {
      event.preventDefault();
      this.detailWidthPx = this.clampDetailWidth(this.detailWidthPx + step);
      localStorage.setItem(STORAGE_KEY, String(this.detailWidthPx));
    }
  }

  private applyDetailFlag(): void {
    const path = this.router.url.split('?')[0];
    const m = path.match(/^\/inbox\/([^/?#]+)/);
    this.hasDetail = !!(m && m[1]);
    if (this.hasDetail) {
      queueMicrotask(() => {
        this.detailWidthPx = this.clampDetailWidth(this.detailWidthPx);
      });
    }
  }

  private clampDetailWidth(w: number): number {
    const grid = this.gridRef()?.nativeElement;
    const total = grid?.getBoundingClientRect().width ?? window.innerWidth;
    const resizer = 6;
    const max = Math.max(MIN_DETAIL_W, total - MIN_LIST_W - resizer);
    return Math.round(Math.min(max, Math.max(MIN_DETAIL_W, w)));
  }
}
