import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { RouterLink } from '@angular/router';
import { forkJoin } from 'rxjs';
import { apiErrorMessage } from '../../core/api-error';

interface AdminOverview {
  catalog: {
    systems_count: number;
    resources_count: number;
    access_roles_count: number;
  };
  metrics: {
    users_count: number;
    requests_count: number;
    audit_entries_count: number;
  };
  workflow: {
    description: string;
    steps: { step: number; user_id: string; login?: string | null; full_name?: string | null }[];
  };
  users: { id: string; login: string; full_name: string; email: string; roles: string }[];
}

interface AdminAccessRoleForm {
  resources: { id: string; system_name: string; resource_name: string; label: string }[];
  risk_levels: string[];
}

@Component({
  selector: 'app-admin',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './admin.component.html',
  styleUrl: './admin.component.scss',
})
export class AdminComponent implements OnInit {
  private readonly http = inject(HttpClient);

  data: AdminOverview | null = null;
  formResources: AdminAccessRoleForm['resources'] = [];
  riskLevels: string[] = ['low', 'medium', 'high'];

  /** Поиск согласующих по логину */
  userSearchQ = '';
  userSearchLoading = false;
  userSearchError: string | null = null;
  userSearchHits: { id: string; login: string; full_name: string }[] = [];

  loading = true;
  error: string | null = null;

  createBusy = false;
  createError: string | null = null;
  createSuccess: string | null = null;

  newResourceId = '';
  newName = '';
  newDescription = '';
  newRisk: string = 'medium';
  approverChain: { id: string; login: string; full_name: string }[] = [];

  ngOnInit(): void {
    forkJoin({
      overview: this.http.get<AdminOverview>('/api/admin/overview'),
      form: this.http.get<AdminAccessRoleForm>('/api/admin/access-role/form'),
    }).subscribe({
      next: ({ overview, form }) => {
        this.data = overview;
        this.formResources = form.resources ?? [];
        if (form.risk_levels?.length) {
          this.riskLevels = form.risk_levels;
        }
        this.loading = false;
      },
      error: (err) => {
        this.loading = false;
        this.error = apiErrorMessage(err);
      },
    });
  }

  searchUsersForChain(): void {
    const q = this.userSearchQ.trim();
    if (q.length < 2) {
      this.userSearchError = 'Введите минимум 2 символа логина.';
      this.userSearchHits = [];
      return;
    }
    this.userSearchLoading = true;
    this.userSearchError = null;
    this.http.get<{ items: { id: string; login: string; full_name: string }[] }>('/api/admin/users/search', { params: { q } }).subscribe({
      next: (res) => {
        this.userSearchLoading = false;
        this.userSearchHits = res.items ?? [];
        if (this.userSearchHits.length === 0) {
          this.userSearchError = 'Никого не найдено.';
        } else {
          this.userSearchError = null;
        }
      },
      error: (err) => {
        this.userSearchLoading = false;
        this.userSearchHits = [];
        this.userSearchError = apiErrorMessage(err);
      },
    });
  }

  addUserToChain(u: { id: string; login: string; full_name: string }): void {
    if (this.isInApproverChain(u.id)) {
      return;
    }
    this.approverChain = [...this.approverChain, u];
  }

  isInApproverChain(userId: string): boolean {
    return this.approverChain.some((x) => x.id === userId);
  }

  removeApprover(i: number): void {
    this.approverChain = this.approverChain.filter((_, idx) => idx !== i);
  }

  moveApprover(i: number, dir: -1 | 1): void {
    const j = i + dir;
    if (j < 0 || j >= this.approverChain.length) {
      return;
    }
    const next = [...this.approverChain];
    [next[i], next[j]] = [next[j], next[i]];
    this.approverChain = next;
  }

  submitNewAccessRole(): void {
    this.createError = null;
    this.createSuccess = null;
    const name = this.newName.trim();
    if (!this.newResourceId || !name) {
      this.createError = 'Укажите ресурс и название роли доступа.';
      return;
    }
    if (this.approverChain.length === 0) {
      this.createError = 'Добавьте в цепочку хотя бы одного согласующего.';
      return;
    }
    this.createBusy = true;
    this.http
      .post<{ id: string }>('/api/admin/access-roles', {
        resource_id: this.newResourceId,
        name,
        description: this.newDescription.trim(),
        risk_level: this.newRisk,
        approver_user_ids: this.approverChain.map((a) => a.id),
      })
      .subscribe({
        next: (res) => {
          this.createBusy = false;
          this.createSuccess = `Роль создана (id: ${res.id}). Она появится в каталоге.`;
          this.newName = '';
          this.newDescription = '';
          this.approverChain = [];
          if (this.data) {
            this.data.catalog.access_roles_count += 1;
          }
        },
        error: (err) => {
          this.createBusy = false;
          this.createError = apiErrorMessage(err);
        },
      });
  }
}
