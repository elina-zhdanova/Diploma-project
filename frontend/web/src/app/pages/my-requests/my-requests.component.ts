import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { ActivatedRoute, Router, RouterLink, RouterLinkActive } from '@angular/router';
import { apiErrorMessage } from '../../core/api-error';
import { matchesRequestSearch } from '../../core/request-search';

interface RequestSummary {
  id: string;
  status: string;
  risk_score?: number | null;
  created_at?: string | null;
  justification?: string;
  /** Названия ролей доступа из позиций заявки (через запятую). */
  access_role_names?: string;
  /** Всего шагов согласования и сколько уже согласовано. */
  approvals_total?: number;
  approvals_done?: number;
  /** YYYY-MM-DD — опциональный срок необходимости доступа. */
  needed_until?: string | null;
  initiator_name?: string | null;
  initiator_login?: string | null;
}

@Component({
  selector: 'app-my-requests',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, RouterLinkActive],
  templateUrl: './my-requests.component.html',
  styleUrl: './my-requests.component.scss',
})
export class MyRequestsComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  items: RequestSummary[] = [];
  /** Подстрока по логину, ФИО, ролям доступа, обоснованию, id. */
  searchQuery = '';
  loading = true;
  error: string | null = null;
  /** После успешной подачи заявки с детальной страницы или из корзины (несколько заявок). */
  submittedNotice = false;
  /** Сколько заявок только что создано (из query submitted). */
  submittedCount = 0;

  ngOnInit(): void {
    const raw = this.route.snapshot.queryParamMap.get('submitted');
    if (raw) {
      const n = parseInt(raw, 10);
      if (!isNaN(n) && n > 0) {
        this.submittedCount = n;
        this.submittedNotice = true;
      }
      if (this.submittedNotice) {
        const id = this.route.firstChild?.snapshot.paramMap.get('id');
        if (id) {
          this.router.navigate(['/requests', id], { queryParams: {}, replaceUrl: true });
        } else {
          this.router.navigate(['/requests'], { queryParams: {}, replaceUrl: true });
        }
      }
    }
    this.http.get<{ items: RequestSummary[] }>('/api/requests/my').subscribe({
      next: (res) => {
        this.items = res.items ?? [];
        this.loading = false;
      },
      error: (err) => {
        this.loading = false;
        this.error = apiErrorMessage(err);
      },
    });
  }

  get filteredItems(): RequestSummary[] {
    return this.items.filter((r) => matchesRequestSearch(r, this.searchQuery));
  }

  statusPillClass(s: string): string {
    const x = (s || '').toLowerCase();
    if (x === 'pending') {
      return 'gk-status-pill gk-status-pending';
    }
    if (x === 'approved') {
      return 'gk-status-pill gk-status-approved';
    }
    if (x === 'rejected') {
      return 'gk-status-pill gk-status-rejected';
    }
    if (x === 'cancelled') {
      return 'gk-status-pill gk-status-cancelled';
    }
    return 'gk-status-pill';
  }

  requestTitle(r: RequestSummary): string {
    const n = (r.access_role_names ?? '').trim();
    return n || 'Заявка на доступ';
  }

  initiatorLabel(r: RequestSummary): string {
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

  showInitiatorLoginTag(r: RequestSummary): boolean {
    const name = (r.initiator_name ?? '').trim();
    const login = (r.initiator_login ?? '').trim();
    return name.length > 0 && login.length > 0;
  }

  /** Доля завершённых шагов согласования, 0–100 (для полосы прогресса). */
  approvalProgressPercent(r: RequestSummary): number {
    const total = r.approvals_total ?? 0;
    const done = r.approvals_done ?? 0;
    if (total <= 0) {
      return 0;
    }
    return Math.min(100, Math.round((100 * done) / total));
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
