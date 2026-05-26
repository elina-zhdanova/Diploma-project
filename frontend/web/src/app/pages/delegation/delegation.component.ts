import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { AuthService } from '../../core/auth.service';
import { apiErrorMessage } from '../../core/api-error';

interface AccessRoleRef {
  id: string;
  name: string;
}

interface DelegationRow {
  id: string;
  from_user_id: string;
  to_user_id: string;
  from_login: string;
  from_full_name: string;
  to_login: string;
  to_full_name: string;
  permanent: boolean;
  start_date: string | null;
  end_date: string | null;
  access_roles: AccessRoleRef[];
  is_active: boolean;
}

interface DelegationUserHit {
  id: string;
  login: string;
  full_name: string;
}

@Component({
  selector: 'app-delegation',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './delegation.component.html',
  styleUrl: './delegation.component.scss',
})
export class DelegationComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly auth = inject(AuthService);

  items: DelegationRow[] = [];
  myAccessRoles: AccessRoleRef[] = [];
  loading = true;
  listError: string | null = null;
  rolesError: string | null = null;

  /** Поиск получателя (любой пользователь, кроме себя) */
  recipientSearchQ = '';
  recipientSearchLoading = false;
  recipientSearchError: string | null = null;
  recipientHits: DelegationUserHit[] = [];
  selectedRecipient: DelegationUserHit | null = null;

  formToUserId = '';
  formPermanent = false;
  formStart = '';
  formEnd = '';
  /**
   * `all` — на бэкенд уходит пустой список ролей = все полномочия согласования.
   * `picked` — только отмеченные роли (можно несколько за одно делегирование).
   */
  scopeMode: 'all' | 'picked' = 'all';
  /** Роли для режима `picked`. */
  selectedRoleIds: string[] = [];
  /** Фильтр списка ролей по названию. */
  roleFilterQ = '';
  formBusy = false;
  formError: string | null = null;
  /** Краткое подтверждение после успешного создания. */
  formSuccess: string | null = null;
  /** Фильтр списка ниже: все / только исходящие / только входящие. */
  listFilter: 'all' | 'out' | 'in' = 'all';

  ngOnInit(): void {
    this.auth.ensureProfile().subscribe(() => {
      this.loadMyAccessRoles();
      this.loadList();
    });
  }

  meId(): string | null {
    return this.auth.user()?.id ?? null;
  }

  isOutgoing(d: DelegationRow): boolean {
    const u = this.meId();
    return !!u && d.from_user_id === u;
  }

  canDelete(d: DelegationRow): boolean {
    return this.isOutgoing(d);
  }

  get filteredAccessRoles(): AccessRoleRef[] {
    const q = this.roleFilterQ.trim().toLowerCase();
    if (!q) {
      return this.myAccessRoles;
    }
    return this.myAccessRoles.filter((r) => r.name.toLowerCase().includes(q));
  }

  pickedRolesCount(): number {
    return this.selectedRoleIds.length;
  }

  toggleRole(id: string): void {
    if (this.selectedRoleIds.includes(id)) {
      this.selectedRoleIds = this.selectedRoleIds.filter((x) => x !== id);
    } else {
      this.selectedRoleIds = [...this.selectedRoleIds, id];
    }
  }

  isRoleChecked(id: string): boolean {
    return this.selectedRoleIds.includes(id);
  }

  selectAllFilteredRoles(): void {
    const ids = new Set(this.selectedRoleIds);
    for (const r of this.filteredAccessRoles) {
      ids.add(r.id);
    }
    this.selectedRoleIds = [...ids];
  }

  clearPickedRoles(): void {
    this.selectedRoleIds = [];
  }

  outgoingCount(): number {
    return this.items.filter((d) => this.isOutgoing(d)).length;
  }

  incomingCount(): number {
    return this.items.filter((d) => !this.isOutgoing(d)).length;
  }

  get filteredDelegations(): DelegationRow[] {
    switch (this.listFilter) {
      case 'out':
        return this.items.filter((d) => this.isOutgoing(d));
      case 'in':
        return this.items.filter((d) => !this.isOutgoing(d));
      default:
        return this.items;
    }
  }

  setScopeMode(mode: 'all' | 'picked'): void {
    this.scopeMode = mode;
  }

  setPermanent(value: boolean): void {
    this.formPermanent = value;
  }

  dismissSuccess(): void {
    this.formSuccess = null;
  }

  private loadList(): void {
    this.loading = true;
    this.listError = null;
    this.http.get<{ items: DelegationRow[] }>('/api/delegations').subscribe({
      next: (res) => {
        this.items = res.items ?? [];
        this.loading = false;
      },
      error: (err) => {
        this.loading = false;
        this.listError = apiErrorMessage(err);
      },
    });
  }

  searchRecipientUsers(): void {
    const q = this.recipientSearchQ.trim();
    if (q.length < 2) {
      this.recipientSearchError = 'Введите минимум 2 символа (логин или ФИО).';
      this.recipientHits = [];
      return;
    }
    this.recipientSearchLoading = true;
    this.recipientSearchError = null;
    this.http
      .get<{ items: DelegationUserHit[] }>('/api/delegation/users/search', { params: { q } })
      .subscribe({
        next: (res) => {
          this.recipientSearchLoading = false;
          this.recipientHits = res.items ?? [];
          if (this.recipientHits.length === 0) {
            this.recipientSearchError = 'Никого не найдено.';
          }
        },
        error: (err) => {
          this.recipientSearchLoading = false;
          this.recipientHits = [];
          this.recipientSearchError = apiErrorMessage(err);
        },
      });
  }

  selectRecipient(u: DelegationUserHit): void {
    this.formToUserId = u.id;
    this.selectedRecipient = u;
    this.recipientHits = [];
    this.recipientSearchError = null;
  }

  clearRecipient(): void {
    this.formToUserId = '';
    this.selectedRecipient = null;
  }

  private loadMyAccessRoles(): void {
    this.rolesError = null;
    this.http.get<{ items: AccessRoleRef[] }>('/api/delegation/my-access-roles').subscribe({
      next: (res) => {
        this.myAccessRoles = res.items ?? [];
      },
      error: (err) => {
        this.rolesError = apiErrorMessage(err);
      },
    });
  }

  submit(): void {
    this.formSuccess = null;
    if (!this.formToUserId) {
      this.formError = 'Укажите получателя.';
      return;
    }
    if (!this.formPermanent && (!this.formStart || !this.formEnd)) {
      this.formError = 'Укажите период (даты) или отметьте «навсегда».';
      return;
    }
    if (this.scopeMode === 'picked' && this.selectedRoleIds.length === 0) {
      this.formError = 'Выберите хотя бы одну роль или переключитесь на «Все роли».';
      return;
    }
    this.formBusy = true;
    this.formError = null;
    const accessRoleIds = this.scopeMode === 'all' ? [] : [...this.selectedRoleIds];
    this.http
      .post('/api/delegations', {
        to_user_id: this.formToUserId,
        permanent: this.formPermanent,
        start_date: this.formPermanent ? '' : this.formStart,
        end_date: this.formPermanent ? '' : this.formEnd,
        access_role_ids: accessRoleIds,
      })
      .subscribe({
        next: () => {
          this.formBusy = false;
          this.formSuccess = 'Делегирование создано — запись появится в списке ниже.';
          this.formToUserId = '';
          this.selectedRecipient = null;
          this.formPermanent = false;
          this.formStart = '';
          this.formEnd = '';
          this.scopeMode = 'all';
          this.selectedRoleIds = [];
          this.roleFilterQ = '';
          this.recipientSearchQ = '';
          this.listFilter = 'all';
          this.loadList();
        },
        error: (err) => {
          this.formBusy = false;
          this.formError = apiErrorMessage(err);
        },
      });
  }

  remove(d: DelegationRow): void {
    if (!this.canDelete(d)) {
      return;
    }
    this.formSuccess = null;
    this.formBusy = true;
    this.formError = null;
    this.http.delete(`/api/delegations/${d.id}`).subscribe({
      next: () => {
        this.formBusy = false;
        this.loadList();
      },
      error: (err) => {
        this.formBusy = false;
        this.formError = apiErrorMessage(err);
      },
    });
  }

  personLabel(login: string, fullName: string): string {
    const fn = (fullName ?? '').trim();
    const lg = (login ?? '').trim();
    if (fn && !fn.includes('?')) {
      return fn;
    }
    return lg || '—';
  }

  formatRange(d: DelegationRow): string {
    if (d.permanent) {
      return 'бессрочно';
    }
    const a = d.start_date ?? '';
    const b = d.end_date ?? '';
    if (a && b) {
      return `${a} — ${b}`;
    }
    return '—';
  }

  scopeLabel(d: DelegationRow): string {
    const ar = d.access_roles ?? [];
    if (ar.length === 0) {
      return 'Все роли, где вы согласующий';
    }
    return ar.map((x) => x.name).join(', ');
  }
}
