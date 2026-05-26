import { Component, DestroyRef, OnInit, inject } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { apiErrorMessage } from '../../core/api-error';
import { InboxNavRefreshService } from '../../core/inbox-nav-refresh.service';
import { matchesRequestSearch } from '../../core/request-search';

interface RequestSummary {
  id: string;
  status: string;
  risk_score?: number | null;
  created_at?: string | null;
  justification?: string;
  /** Текущий шаг маршрута согласования */
  current_step?: number;
  needed_until?: string | null;
  /** Названия ролей доступа из позиций заявки (через запятую). */
  access_role_names?: string;
  /** Инициатор заявки (с сервера). */
  initiator_name?: string | null;
  initiator_login?: string | null;
}

interface BulkApproveResponse {
  approved: string[];
  failed: { id: string; error: string }[];
}

interface BulkRejectResponse {
  rejected: string[];
  failed: { id: string; error: string }[];
}

@Component({
  selector: 'app-inbox',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, RouterLinkActive],
  templateUrl: './inbox.component.html',
  styleUrl: './inbox.component.scss',
})
export class InboxComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly inboxNavRefresh = inject(InboxNavRefreshService);
  private readonly destroyRef = inject(DestroyRef);

  items: RequestSummary[] = [];
  /** Подстрока по логину, ФИО, ролям доступа, обоснованию, id. */
  searchQuery = '';
  loading = true;
  error: string | null = null;

  /** Выбранные id заявок для массового согласования. */
  selectedIds: string[] = [];
  bulkComment = '';
  bulkBusy = false;
  /** Результат последнего массового действия (согласование или отказ). */
  lastBulkResult: { type: 'approve' | 'reject'; text: string } | null = null;
  bulkError: string | null = null;

  ngOnInit(): void {
    this.fetchInbox();
    this.inboxNavRefresh.refresh$.pipe(takeUntilDestroyed(this.destroyRef)).subscribe(() => this.fetchInbox({ silent: true }));
  }

  private fetchInbox(opts?: { silent?: boolean }): void {
    if (!opts?.silent) {
      this.loading = true;
    }
    this.error = null;
    this.http.get<{ items: RequestSummary[] }>('/api/approvals/inbox').subscribe({
      next: (res) => {
        this.items = res.items ?? [];
        const idSet = new Set(this.items.map((i) => i.id));
        this.selectedIds = this.selectedIds.filter((id) => idSet.has(id));
        this.loading = false;
      },
      error: (err) => {
        this.loading = false;
        this.error = apiErrorMessage(err);
      },
    });
  }

  formatDateShort(ymd: string | null | undefined): string {
    if (!ymd || !/^\d{4}-\d{2}-\d{2}$/.test(ymd)) {
      return '';
    }
    const [y, m, d] = ymd.split('-').map(Number);
    const dt = new Date(y, m - 1, d);
    if (Number.isNaN(dt.getTime())) {
      return ymd;
    }
    return dt.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short', year: 'numeric' });
  }

  /** Роли из API — строка «роль1, роль2» → список для отображения как в каталоге. */
  roleNamesList(r: RequestSummary): string[] {
    const raw = (r.access_role_names ?? '').trim();
    if (!raw) {
      return [];
    }
    return raw
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);
  }

  requestTitle(r: RequestSummary): string {
    const names = this.roleNamesList(r);
    if (names.length === 0) {
      return 'Заявка на доступ';
    }
    return names.join(', ');
  }

  /** Имя инициатора для строки в списке. */
  initiatorDisplayName(r: RequestSummary): string {
    const name = (r.initiator_name ?? '').trim();
    const login = (r.initiator_login ?? '').trim();
    if (name) {
      return name;
    }
    if (login) {
      return login;
    }
    return '—';
  }

  /** Логин отдельным тегом — только если есть и ФИО, и логин (иначе логин уже в имени). */
  showInitiatorLoginTag(r: RequestSummary): boolean {
    const name = (r.initiator_name ?? '').trim();
    const login = (r.initiator_login ?? '').trim();
    return name.length > 0 && login.length > 0;
  }

  get filteredItems(): RequestSummary[] {
    return this.items.filter((r) => matchesRequestSearch(r, this.searchQuery));
  }

  isSelected(id: string): boolean {
    return this.selectedIds.includes(id);
  }

  toggle(id: string): void {
    if (this.selectedIds.includes(id)) {
      this.selectedIds = this.selectedIds.filter((x) => x !== id);
    } else {
      this.selectedIds = [...this.selectedIds, id];
    }
  }

  get allSelected(): boolean {
    const vis = this.filteredItems;
    return vis.length > 0 && vis.every((r) => this.selectedIds.includes(r.id));
  }

  toggleAll(): void {
    const ids = this.filteredItems.map((r) => r.id);
    if (ids.length === 0) {
      return;
    }
    const allOn = ids.every((id) => this.selectedIds.includes(id));
    if (allOn) {
      const idSet = new Set(ids);
      this.selectedIds = this.selectedIds.filter((x) => !idSet.has(x));
    } else {
      this.selectedIds = [...new Set([...this.selectedIds, ...ids])];
    }
  }

  bulkApprove(): void {
    if (this.selectedIds.length === 0) {
      return;
    }
    this.bulkBusy = true;
    this.bulkError = null;
    this.lastBulkResult = null;
    this.http
      .post<BulkApproveResponse>('/api/approvals/bulk-approve', {
        request_ids: this.selectedIds,
        comment: this.bulkComment.trim(),
      })
      .subscribe({
        next: (res) => {
          this.bulkBusy = false;
          const nOk = res.approved?.length ?? 0;
          const nFail = res.failed?.length ?? 0;
          let text: string;
          if (nFail === 0) {
            text = nOk === 1 ? 'Согласована 1 заявка.' : `Согласовано заявок: ${nOk}.`;
          } else {
            text = `Согласовано: ${nOk}, не удалось: ${nFail}.`;
          }
          this.lastBulkResult = { type: 'approve', text };
          this.selectedIds = [];
          this.bulkComment = '';
          this.inboxNavRefresh.ping();
          this.fetchInbox({ silent: true });
        },
        error: (err) => {
          this.bulkBusy = false;
          this.bulkError = apiErrorMessage(err);
        },
      });
  }

  bulkReject(): void {
    if (this.selectedIds.length === 0) {
      return;
    }
    this.bulkBusy = true;
    this.bulkError = null;
    this.lastBulkResult = null;
    this.http
      .post<BulkRejectResponse>('/api/approvals/bulk-reject', {
        request_ids: this.selectedIds,
        comment: this.bulkComment.trim(),
      })
      .subscribe({
        next: (res) => {
          this.bulkBusy = false;
          const nOk = res.rejected?.length ?? 0;
          const nFail = res.failed?.length ?? 0;
          let text: string;
          if (nFail === 0) {
            text = nOk === 1 ? 'В доступе отказано по 1 заявке.' : `В доступе отказано по заявкам: ${nOk}.`;
          } else {
            text = `Отклонено: ${nOk}, не удалось: ${nFail}.`;
          }
          this.lastBulkResult = { type: 'reject', text };
          this.selectedIds = [];
          this.bulkComment = '';
          this.inboxNavRefresh.ping();
          this.fetchInbox({ silent: true });
        },
        error: (err) => {
          this.bulkBusy = false;
          this.bulkError = apiErrorMessage(err);
        },
      });
  }

  formatWhen(iso: string | null | undefined): string {
    if (!iso) {
      return '';
    }
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) {
      return '';
    }
    return d.toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' });
  }
}
