import { Injectable, signal } from '@angular/core';

const STORAGE_KEY = 'itshop-theme';

export type ThemeMode = 'dark' | 'light';

@Injectable({ providedIn: 'root' })
export class ThemeService {
  /** Текущая тема (синхронизирована с `html[data-theme]`). */
  readonly mode = signal<ThemeMode>('dark');

  constructor() {
    if (typeof document === 'undefined') {
      return;
    }
    const stored = localStorage.getItem(STORAGE_KEY) as ThemeMode | null;
    let initial: ThemeMode = 'dark';
    if (stored === 'light' || stored === 'dark') {
      initial = stored;
    }
    this.apply(initial, false);
  }

  /** Переключить светлая ↔ тёмная. */
  toggle(): void {
    this.apply(this.mode() === 'dark' ? 'light' : 'dark', true);
  }

  apply(mode: ThemeMode, persist = true): void {
    this.mode.set(mode);
    document.documentElement.setAttribute('data-theme', mode);
    if (persist) {
      localStorage.setItem(STORAGE_KEY, mode);
    }
    const meta = document.querySelector('meta[name="theme-color"]');
    if (meta) {
      meta.setAttribute('content', mode === 'light' ? '#f8fafc' : '#0a0a0a');
    }
  }

  isLight(): boolean {
    return this.mode() === 'light';
  }
}
