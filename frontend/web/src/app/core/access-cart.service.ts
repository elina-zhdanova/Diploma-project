import { Injectable, computed, signal } from '@angular/core';

const STORAGE_KEY = 'itshop.accessCart.v2';
const LEGACY_KEY = 'itshop.accessCart.v1';

export interface AccessCartLine {
  accessRoleId: string;
  name: string;
  systemName: string;
  resourceName: string;
  /** Обоснование запроса именно этой роли. */
  justification: string;
}

@Injectable({ providedIn: 'root' })
export class AccessCartService {
  /** Позиции корзины (роли доступа). */
  readonly items = signal<AccessCartLine[]>([]);

  readonly count = computed(() => this.items().length);

  constructor() {
    try {
      let raw = sessionStorage.getItem(STORAGE_KEY);
      if (!raw) {
        const legacy = sessionStorage.getItem(LEGACY_KEY);
        if (legacy) {
          raw = this.migrateV1ToV2(legacy);
          sessionStorage.removeItem(LEGACY_KEY);
          if (raw) {
            sessionStorage.setItem(STORAGE_KEY, raw);
          }
        }
      }
      if (raw) {
        const parsed = JSON.parse(raw) as unknown;
        if (Array.isArray(parsed)) {
          const lines: AccessCartLine[] = [];
          for (const x of parsed) {
            if (x && typeof x === 'object' && 'accessRoleId' in x && 'name' in x) {
              const o = x as Record<string, unknown>;
              lines.push({
                accessRoleId: String(o['accessRoleId']),
                name: String(o['name'] ?? ''),
                systemName: String(o['systemName'] ?? ''),
                resourceName: String(o['resourceName'] ?? ''),
                justification: typeof o['justification'] === 'string' ? o['justification'] : '',
              });
            }
          }
          this.items.set(lines);
        }
      }
    } catch {
      /* ignore */
    }
  }

  private migrateV1ToV2(legacyJson: string): string | null {
    try {
      const parsed = JSON.parse(legacyJson) as unknown;
      if (!Array.isArray(parsed)) {
        return null;
      }
      const out: AccessCartLine[] = [];
      for (const x of parsed) {
        if (x && typeof x === 'object' && 'accessRoleId' in x && 'name' in x) {
          const o = x as Record<string, unknown>;
          out.push({
            accessRoleId: String(o['accessRoleId']),
            name: String(o['name'] ?? ''),
            systemName: String(o['systemName'] ?? ''),
            resourceName: String(o['resourceName'] ?? ''),
            justification: '',
          });
        }
      }
      return JSON.stringify(out);
    } catch {
      return null;
    }
  }

  private persist(): void {
    try {
      sessionStorage.setItem(STORAGE_KEY, JSON.stringify(this.items()));
    } catch {
      /* ignore */
    }
  }

  has(id: string): boolean {
    return this.items().some((x) => x.accessRoleId === id);
  }

  add(line: AccessCartLine): void {
    if (this.has(line.accessRoleId)) {
      return;
    }
    const j = (line.justification ?? '').trim();
    this.items.update((arr) => [...arr, { ...line, justification: j }]);
    this.persist();
  }

  /** Обновить обоснование по id роли. */
  setJustification(accessRoleId: string, justification: string): void {
    const j = justification.trim();
    this.items.update((arr) =>
      arr.map((x) => (x.accessRoleId === accessRoleId ? { ...x, justification: j } : x)),
    );
    this.persist();
  }

  remove(accessRoleId: string): void {
    this.items.update((arr) => arr.filter((x) => x.accessRoleId !== accessRoleId));
    this.persist();
  }

  clear(): void {
    this.items.set([]);
    this.persist();
  }

  /** Заменить список целиком (например, после частичной отправки корзины). */
  setItems(lines: AccessCartLine[]): void {
    this.items.set(lines.map((l) => ({ ...l, justification: (l.justification ?? '').trim() })));
    this.persist();
  }
}
