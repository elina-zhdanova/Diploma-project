import { Injectable, signal } from '@angular/core';

const SEARCH_KEY = 'itshop.catalog.search';

/**
 * Сохраняет строку поиска по каталогу между визитами и экранами «системы» / «роли в системе».
 */
@Injectable({ providedIn: 'root' })
export class CatalogUiStateService {
  readonly searchQuery = signal(this.loadSearch());

  private loadSearch(): string {
    try {
      return localStorage.getItem(SEARCH_KEY) ?? '';
    } catch {
      return '';
    }
  }

  setSearch(query: string): void {
    this.searchQuery.set(query);
    try {
      localStorage.setItem(SEARCH_KEY, query);
    } catch {
      /* ignore quota */
    }
  }
}
