import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { apiErrorMessage } from '../../core/api-error';
import { CatalogTreeService, CatalogSystem } from '../../core/catalog-tree.service';
import { CatalogUiStateService } from '../../core/catalog-ui-state.service';
import { AccessCartService } from '../../core/access-cart.service';

/** Плоская строка для списка ролей. */
interface StoreRoleRow {
  id: string;
  name: string;
  description?: string | null;
  risk_level: string;
  systemName: string;
  resourceName: string;
  sensitive: boolean;
}

@Component({
  selector: 'app-store-system-roles',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './store-system-roles.component.html',
  styleUrl: './store-system-roles.component.scss',
})
export class StoreSystemRolesComponent implements OnInit {
  private readonly catalog = inject(CatalogTreeService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  readonly ui = inject(CatalogUiStateService);
  readonly cart = inject(AccessCartService);

  systemId = '';
  system: CatalogSystem | null = null;
  loading = true;
  error: string | null = null;

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('systemId');
    if (!id) {
      this.router.navigate(['/store']);
      return;
    }
    this.systemId = id;

    this.catalog.getTree().subscribe({
      next: (tree) => {
        const sys = tree.find((s) => s.id === this.systemId) ?? null;
        if (!sys) {
          this.router.navigate(['/store']);
          return;
        }
        this.system = sys;
        this.loading = false;
      },
      error: (err) => {
        this.loading = false;
        this.error = apiErrorMessage(err);
      },
    });
  }

  /** Диалог: для какой роли вводим обоснование перед добавлением в корзину. */
  cartDialogRow: StoreRoleRow | null = null;
  cartJustificationDraft = '';

  get filteredRoles(): StoreRoleRow[] {
    if (!this.system) {
      return [];
    }
    const q = this.ui.searchQuery().trim().toLowerCase();
    const rows: StoreRoleRow[] = [];

    for (const res of this.system.resources) {
      for (const role of res.roles) {
        if (q) {
          const bucket = [this.system.name, res.name, role.name, role.description ?? '', role.risk_level]
            .join(' ')
            .toLowerCase();
          if (!bucket.includes(q)) {
            continue;
          }
        }
        rows.push({
          id: role.id,
          name: role.name,
          description: role.description,
          risk_level: role.risk_level,
          systemName: this.system.name,
          resourceName: res.name,
          sensitive: res.sensitive,
        });
      }
    }

    rows.sort((a, b) => a.name.localeCompare(b.name, 'ru', { sensitivity: 'base' }));
    return rows;
  }

  openCartDialog(row: StoreRoleRow, event: Event): void {
    event.preventDefault();
    event.stopPropagation();
    this.cartDialogRow = row;
    this.cartJustificationDraft = '';
  }

  closeCartDialog(): void {
    this.cartDialogRow = null;
    this.cartJustificationDraft = '';
  }

  confirmCartDialog(): void {
    const row = this.cartDialogRow;
    if (!row) {
      return;
    }
    const j = this.cartJustificationDraft.trim();
    if (!j) {
      return;
    }
    this.cart.add({
      accessRoleId: row.id,
      name: row.name,
      systemName: row.systemName,
      resourceName: row.resourceName,
      justification: j,
    });
    this.closeCartDialog();
  }

  inCart(id: string): boolean {
    return this.cart.has(id);
  }

  riskTagClass(level: string): string {
    const l = (level || '').toLowerCase();
    if (l === 'high') {
      return 'gk-tag gk-tag-risk-high';
    }
    if (l === 'medium') {
      return 'gk-tag gk-tag-risk-medium';
    }
    return 'gk-tag gk-tag-risk-low';
  }
}
