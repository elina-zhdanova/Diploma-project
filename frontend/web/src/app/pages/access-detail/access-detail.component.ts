import { Component, DestroyRef, OnInit, inject } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { distinctUntilChanged, map } from 'rxjs/operators';
import { apiErrorMessage } from '../../core/api-error';
import { AccessCartService } from '../../core/access-cart.service';

interface AccessRoleDetail {
  id: string;
  name: string;
  code?: string;
  description?: string;
  risk_level: string;
  resource_name: string;
  sensitive?: boolean;
  system_name: string;
  is_active?: boolean;
}

@Component({
  selector: 'app-access-detail',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './access-detail.component.html',
  styleUrl: './access-detail.component.scss',
})
export class AccessDetailComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly destroyRef = inject(DestroyRef);
  readonly cart = inject(AccessCartService);

  role: AccessRoleDetail | null = null;
  justification = '';
  /** Опционально: доступ нужен не позже этой даты (включительно). */
  limitAccessByDate = false;
  neededUntil = '';
  /** Минимальная дата для поля (сегодня, локальный календарь). */
  minNeededUntil = '';
  /** Как в Gatekeeper: сначала кнопка «Запросить доступ», затем форма */
  showRequestForm = false;
  /** Ввод обоснования перед добавлением текущей роли в корзину. */
  addCartPanelOpen = false;
  cartLineJustification = '';
  loading = true;
  submitting = false;
  error: string | null = null;

  ngOnInit(): void {
    const d = new Date();
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    this.minNeededUntil = `${y}-${m}-${day}`;

    this.route.paramMap
      .pipe(
        map((pm) => pm.get('id')),
        distinctUntilChanged(),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe((id) => {
        if (!id?.trim()) {
          void this.router.navigate(['/store']);
          return;
        }
        this.load(id);
      });
  }

  private load(id: string): void {
    this.justification = '';
    this.limitAccessByDate = false;
    this.neededUntil = '';
    this.showRequestForm = false;
    this.addCartPanelOpen = false;
    this.cartLineJustification = '';
    this.loading = true;
    this.error = null;
    this.role = null;

    this.http.get<AccessRoleDetail>(`/api/access-roles/${id}`).subscribe({
      next: (r) => {
        this.role = r;
        this.loading = false;
      },
      error: (err) => {
        this.loading = false;
        this.error = apiErrorMessage(err);
      },
    });
  }

  submit(): void {
    if (!this.role || !this.justification.trim()) {
      return;
    }
    if (this.limitAccessByDate && !this.neededUntil.trim()) {
      this.error = 'Укажите дату «нужен до» или снимите галочку.';
      return;
    }
    this.submitting = true;
    this.error = null;
    const body: Record<string, unknown> = {
      items: [{ access_role_id: this.role.id, justification: this.justification.trim() }],
    };
    if (this.limitAccessByDate && this.neededUntil.trim()) {
      body['needed_until'] = this.neededUntil.trim();
    }
    this.http.post<{ id: string }>('/api/requests', body).subscribe({
      next: () => {
        this.submitting = false;
        this.router.navigate(['/requests'], { queryParams: { submitted: '1' } });
      },
      error: (err) => {
        this.submitting = false;
        this.error = apiErrorMessage(err);
      },
    });
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

  confirmAddToCart(): void {
    const r = this.role;
    if (!r) {
      return;
    }
    const j = this.cartLineJustification.trim();
    if (!j) {
      return;
    }
    this.cart.add({
      accessRoleId: r.id,
      name: r.name,
      systemName: r.system_name,
      resourceName: r.resource_name,
      justification: j,
    });
    this.addCartPanelOpen = false;
    this.cartLineJustification = '';
  }

  cancelAddToCart(): void {
    this.addCartPanelOpen = false;
    this.cartLineJustification = '';
  }

  inCart(): boolean {
    const id = this.role?.id;
    return !!id && this.cart.has(id);
  }

  letterAvatar(): string {
    const n = this.role?.name?.trim();
    if (!n) {
      return '?';
    }
    return n.charAt(0).toUpperCase();
  }
}
