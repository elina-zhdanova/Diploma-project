import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map, shareReplay } from 'rxjs';

export interface CatalogRole {
  id: string;
  name: string;
  description?: string | null;
  risk_level: string;
}

export interface CatalogResource {
  id: string;
  name: string;
  sensitive: boolean;
  roles: CatalogRole[];
}

export interface CatalogSystem {
  id: string;
  name: string;
  description?: string | null;
  resources: CatalogResource[];
}

@Injectable({ providedIn: 'root' })
export class CatalogTreeService {
  private readonly http = inject(HttpClient);

  private tree$?: Observable<CatalogSystem[]>;

  /** Дерево каталога; один запрос на сессию приложения. */
  getTree(): Observable<CatalogSystem[]> {
    if (!this.tree$) {
      this.tree$ = this.http.get<{ items: CatalogSystem[] }>('/api/catalog/tree').pipe(
        map((res) => res.items ?? []),
        shareReplay(1),
      );
    }
    return this.tree$;
  }
}
