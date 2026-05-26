import { Component, DestroyRef, HostBinding, OnInit, inject } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { distinctUntilChanged, filter, map } from 'rxjs/operators';
import { HttpClient } from '@angular/common/http';
import { AuthService } from '../../core/auth.service';
import { InboxNavRefreshService } from '../../core/inbox-nav-refresh.service';
import { apiErrorMessage } from '../../core/api-error';

interface RequestItem {
  id: string;
  access_role_id: string;
  access_role_name: string;
  risk_level: string;
  resource_name: string;
  /** Обоснование по этой позиции (если сервер отдал). */
  justification?: string | null;
}

interface ApprovalRow {
  id: string;
  approver_id: string;
  approver_name?: string;
  approver_login?: string;
  step_number: number;
  status: string;
  decision_at?: string | null;
  comment?: string;
}

interface RequestDetail {
  id: string;
  initiator_id: string;
  initiator_name?: string;
  initiator_login?: string;
  status: string;
  risk_score?: number | null;
  created_at?: string | null;
  updated_at?: string | null;
  justification?: string;
  /** YYYY-MM-DD — срок, до которого нужен доступ (включительно). */
  needed_until?: string | null;
  items: RequestItem[];
  approvals: ApprovalRow[];
  /** Сервер: можно ли текущему пользователю решить текущий шаг (в т.ч. по делегированию). */
  viewer_can_act_on_current_step?: boolean;
  viewer_acts_as_delegate?: boolean;
}

@Component({
  selector: 'app-request-detail',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './request-detail.component.html',
  styleUrl: './request-detail.component.scss',
})
export class RequestDetailComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly authService = inject(AuthService);
  private readonly inboxNavRefresh = inject(InboxNavRefreshService);
  private readonly destroyRef = inject(DestroyRef);

  /** Правая колонка на экране «Мои заявки» (~25% ширины). */
  panelMode = false;

  @HostBinding('class.request-detail-host--panel')
  get hostPanelClass(): boolean {
    return this.panelMode;
  }

  req: RequestDetail | null = null;
  loading = true;
  error: string | null = null;

  actionError: string | null = null;
  actionBusy = false;

  approveComment = '';
  rejectComment = '';

  ngOnInit(): void {
    this.panelMode = !!this.route.snapshot.data['panel'];
    this.route.data.pipe(takeUntilDestroyed(this.destroyRef)).subscribe((d) => {
      this.panelMode = !!d['panel'];
    });
    this.route.paramMap
      .pipe(
        map((pm) => pm.get('id')),
        filter((id): id is string => !!id),
        distinctUntilChanged(),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe((id) => this.load(id));
  }

  private load(id: string): void {
    this.approveComment = '';
    this.rejectComment = '';
    this.loading = true;
    this.error = null;
    this.http.get<RequestDetail>(`/api/requests/${id}`).subscribe({
      next: (r) => {
        this.req = r;
        this.loading = false;
      },
      error: (err) => {
        this.loading = false;
        this.error = apiErrorMessage(err);
      },
    });
  }

  meId(): string | null {
    return this.authService.user()?.id ?? null;
  }

  isInitiator(): boolean {
    const u = this.meId();
    return !!u && !!this.req && this.req.initiator_id === u;
  }

  /** ФИО или логин — чтобы не показывать «не указан», если в БД только login. */
  initiatorDisplayName(): string {
    if (!this.req) {
      return '';
    }
    const name = (this.req.initiator_name ?? '').trim();
    const login = (this.req.initiator_login ?? '').trim();
    if (name) {
      return name;
    }
    if (login) {
      return login;
    }
    return '';
  }

  showInitiatorLoginTag(): boolean {
    if (!this.req) {
      return false;
    }
    const name = (this.req.initiator_name ?? '').trim();
    const login = (this.req.initiator_login ?? '').trim();
    return name.length > 0 && login.length > 0;
  }

  canCancel(): boolean {
    if (!this.req || !this.isInitiator()) {
      return false;
    }
    const s = this.req.status.toLowerCase();
    return s === 'pending' || s === 'draft';
  }

  canDecide(): boolean {
    return !!this.req?.viewer_can_act_on_current_step;
  }

  actsAsDelegate(): boolean {
    return !!this.req?.viewer_acts_as_delegate;
  }

  cancel(): void {
    if (!this.req) {
      return;
    }
    this.actionBusy = true;
    this.actionError = null;
    this.http.post(`/api/requests/${this.req.id}/cancel`, {}).subscribe({
      next: () => {
        this.actionBusy = false;
        this.load(this.req!.id);
      },
      error: (err) => {
        this.actionBusy = false;
        this.actionError = apiErrorMessage(err);
      },
    });
  }

  approve(): void {
    if (!this.req) {
      return;
    }
    this.actionBusy = true;
    this.actionError = null;
    this.http.post(`/api/approvals/${this.req.id}/approve`, { comment: this.approveComment || '' }).subscribe({
      next: () => {
        this.actionBusy = false;
        this.approveComment = '';
        this.inboxNavRefresh.ping();
        this.load(this.req!.id);
      },
      error: (err) => {
        this.actionBusy = false;
        this.actionError = apiErrorMessage(err);
      },
    });
  }

  reject(): void {
    if (!this.req) {
      return;
    }
    this.actionBusy = true;
    this.actionError = null;
    this.http.post(`/api/approvals/${this.req.id}/reject`, { comment: this.rejectComment || '' }).subscribe({
      next: () => {
        this.actionBusy = false;
        this.rejectComment = '';
        this.inboxNavRefresh.ping();
        this.load(this.req!.id);
      },
      error: (err) => {
        this.actionBusy = false;
        this.actionError = apiErrorMessage(err);
      },
    });
  }

  /** Для согласований / legacy badge */
  statusClass(s: string): string {
    const x = (s || '').toLowerCase();
    if (x === 'pending') {
      return 'pending';
    }
    if (x === 'approved') {
      return 'approved';
    }
    if (x === 'rejected') {
      return 'rejected';
    }
    if (x === 'cancelled') {
      return 'cancelled';
    }
    return '';
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

  /** Есть обоснования по позициям — общий блок в шапке не показываем, чтобы не дублировать. */
  hasPerItemJustification(): boolean {
    const items = this.req?.items ?? [];
    return items.some((i) => (i.justification ?? '').trim().length > 0);
  }

  /** Имя из БД иногда приходит с «????» из‑за сбоя кодировки — тогда показываем логин. */
  private isBogusApproverLabel(s: string): boolean {
    if (!s.trim()) {
      return true;
    }
    if (s.includes('?')) {
      return true;
    }
    if (s.includes('\uFFFD')) {
      return true;
    }
    return false;
  }

  approverDisplayName(a: ApprovalRow): string {
    const name = (a.approver_name ?? '').trim();
    const login = (a.approver_login ?? '').trim();
    if (!login && !name) {
      return 'Согласующий';
    }
    if (this.isBogusApproverLabel(name)) {
      return login ? `Учётная запись: ${login}` : 'Согласующий';
    }
    return name;
  }

  /** Не дублируем логин, если он уже выведен в строке имени. */
  showApproverLoginTag(a: ApprovalRow): boolean {
    const name = (a.approver_name ?? '').trim();
    const login = (a.approver_login ?? '').trim();
    if (!login) {
      return false;
    }
    if (this.isBogusApproverLabel(name)) {
      return false;
    }
    return true;
  }

  /** Строка YYYY-MM-DD → краткая дата для РФ. */
  formatDateRU(ymd: string | null | undefined): string {
    if (!ymd || !/^\d{4}-\d{2}-\d{2}$/.test(ymd)) {
      return ymd ?? '';
    }
    const [y, m, d] = ymd.split('-').map(Number);
    const dt = new Date(y, m - 1, d);
    if (Number.isNaN(dt.getTime())) {
      return ymd;
    }
    return dt.toLocaleDateString('ru-RU', { dateStyle: 'medium' });
  }

  formatWhen(iso: string | null | undefined): string {
    if (!iso) {
      return 'не указано';
    }
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) {
      return iso;
    }
    return d.toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' });
  }

  /** Заголовок: названия ролей из позиций (одна или несколько через запятую). */
  requestRoleHeadline(): string {
    if (!this.req?.items?.length) {
      return 'Заявка';
    }
    const names = this.req.items.map((i) => (i.access_role_name ?? '').trim()).filter(Boolean);
    if (names.length === 0) {
      return 'Заявка';
    }
    if (names.length === 1) {
      return names[0];
    }
    return names.join(', ');
  }

  /** Список-родитель: «На согласование» или «Мои заявки». */
  backToListPath(): string[] {
    const path = this.router.url.split('?')[0];
    if (path.startsWith('/inbox')) {
      return ['/inbox'];
    }
    return ['/requests'];
  }

  formatBytes(n: number): string {
    if (n < 1024) {
      return `${n} Б`;
    }
    if (n < 1024 * 1024) {
      return `${(n / 1024).toFixed(1)} КБ`;
    }
    return `${(n / (1024 * 1024)).toFixed(1)} МБ`;
  }

}
