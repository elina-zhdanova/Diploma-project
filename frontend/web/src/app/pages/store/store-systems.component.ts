import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { apiErrorMessage } from '../../core/api-error';
import { CatalogTreeService, CatalogSystem } from '../../core/catalog-tree.service';
import { CatalogUiStateService } from '../../core/catalog-ui-state.service';

@Component({
  selector: 'app-store-systems',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './store-systems.component.html',
  styleUrl: './store-systems.component.scss',
})
export class StoreSystemsComponent implements OnInit {
  private readonly catalog = inject(CatalogTreeService);
  readonly ui = inject(CatalogUiStateService);

  tree: CatalogSystem[] = [];
  loading = true;
  error: string | null = null;

  ngOnInit(): void {
    this.catalog.getTree().subscribe({
      next: (items) => {
        this.tree = items;
        this.loading = false;
      },
      error: (err) => {
        this.loading = false;
        this.error = apiErrorMessage(err);
      },
    });
  }

  /** Системы с учётом сохранённого поиска: система, ресурс и группы (роли) доступа. */
  get filteredSystems(): CatalogSystem[] {
    const q = this.ui.searchQuery().trim().toLowerCase();
    if (!q) {
      return [...this.tree].sort((a, b) => a.name.localeCompare(b.name, 'ru', { sensitivity: 'base' }));
    }
    const tokens = q.split(/\s+/).filter(Boolean);
    return this.tree
      .filter((s) => {
        const roleNames = s.resources.flatMap((r) => r.roles.map((x) => x.name)).join(' ');
        const resourceNames = s.resources.map((r) => r.name).join(' ');
        const bucket = [s.name, s.description ?? '', resourceNames, roleNames].join(' ').toLowerCase();
        return tokens.every((t) => bucket.includes(t));
      })
      .sort((a, b) => a.name.localeCompare(b.name, 'ru', { sensitivity: 'base' }));
  }

  roleCount(sys: CatalogSystem): number {
    let n = 0;
    for (const res of sys.resources) {
      n += res.roles.length;
    }
    return n;
  }
}
