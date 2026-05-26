import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient, HttpParams } from '@angular/common/http';
import { RouterLink } from '@angular/router';
import { apiErrorMessage } from '../../core/api-error';

export interface AuditItem {
  id: string;
  user_login: string;
  request_id: string;
  action: string;
  details: Record<string, unknown> | null;
  created_at: string | null;
}

@Component({
  selector: 'app-audit',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './audit.component.html',
  styleUrl: './audit.component.scss',
})
export class AuditComponent implements OnInit {
  private readonly http = inject(HttpClient);

  items: AuditItem[] = [];
  loading = true;
  error: string | null = null;

  searchQ = '';
  filterUserId = '';
  filterRequestId = '';

  /** Сводка по типам событий (для блока ИБ) — по текущей выборке */
  categoryCounts: { key: string; label: string; count: number }[] = [];

  ngOnInit(): void {
    this.load();
  }

  load(): void {
    this.loading = true;
    this.error = null;
    let p = new HttpParams().set('limit', '200').set('offset', '0');
    const q = this.searchQ.trim();
    if (q) {
      p = p.set('q', q);
    }
    const uid = this.filterUserId.trim();
    if (uid) {
      p = p.set('user_id', uid);
    }
    const rid = this.filterRequestId.trim();
    if (rid) {
      p = p.set('request_id', rid);
    }
    this.http.get<{ items: AuditItem[] }>('/api/audit', { params: p }).subscribe({
      next: (res) => {
        this.items = res.items ?? [];
        this.rebuildCategoryCounts();
        this.loading = false;
      },
      error: (err) => {
        this.loading = false;
        this.error = apiErrorMessage(err);
      },
    });
  }

  applyFilters(): void {
    this.load();
  }

  clearFilters(): void {
    this.searchQ = '';
    this.filterUserId = '';
    this.filterRequestId = '';
    this.load();
  }

  private rebuildCategoryCounts(): void {
    const map = new Map<string, number>();
    for (const it of this.items) {
      const prefix = (it.action || '').split('.')[0] || 'other';
      map.set(prefix, (map.get(prefix) ?? 0) + 1);
    }
    const labels: Record<string, string> = {
      auth: 'Аутентификация',
      request: 'Заявки',
      approval: 'Согласование',
      attachment: 'Вложения',
      other: 'Прочее',
    };
    this.categoryCounts = [...map.entries()]
      .map(([key, count]) => ({
        key,
        label: labels[key] ?? key,
        count,
      }))
      .sort((a, b) => b.count - a.count);
  }

  formatWhen(iso: string | null | undefined): string {
    if (!iso) {
      return '—';
    }
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) {
      return iso;
    }
    return d.toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'medium' });
  }

  shortId(s: string): string {
    if (!s) {
      return '—';
    }
    return s.length > 12 ? `${s.slice(0, 8)}…` : s;
  }

  detailsJson(d: Record<string, unknown> | null): string {
    if (!d || typeof d !== 'object') {
      return '';
    }
    try {
      return JSON.stringify(d, null, 2);
    } catch {
      return String(d);
    }
  }

  actionClass(action: string): string {
    const p = (action || '').split('.')[0];
    if (p === 'auth') {
      return 'audit-act audit-act--auth';
    }
    if (p === 'approval') {
      return 'audit-act audit-act--approval';
    }
    if (p === 'request') {
      return 'audit-act audit-act--request';
    }
    if (p === 'attachment') {
      return 'audit-act audit-act--attach';
    }
    return 'audit-act';
  }
}
