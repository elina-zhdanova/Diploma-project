import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { AccessCartLine, AccessCartService } from '../../core/access-cart.service';
import { apiErrorMessage } from '../../core/api-error';

@Component({
  selector: 'app-access-cart',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './access-cart.component.html',
  styleUrl: './access-cart.component.scss',
})
export class AccessCartComponent {
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);
  readonly cart = inject(AccessCartService);

  limitAccessByDate = false;
  neededUntil = '';
  minNeededUntil = '';
  submitting = false;
  error: string | null = null;

  constructor() {
    const d = new Date();
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    this.minNeededUntil = `${y}-${m}-${day}`;
  }

  remove(id: string): void {
    this.cart.remove(id);
  }

  allLinesHaveJustification(): boolean {
    const lines = this.cart.items();
    if (lines.length === 0) {
      return false;
    }
    return lines.every((l) => (l.justification ?? '').trim().length > 0);
  }

  async submit(): Promise<void> {
    const lines = this.cart.items();
    if (lines.length === 0) {
      this.error = 'Добавьте в корзину хотя бы одну роль в каталоге.';
      return;
    }
    if (!this.allLinesHaveJustification()) {
      this.error = 'Укажите обоснование для каждой выбранной роли.';
      return;
    }
    if (this.limitAccessByDate && !this.neededUntil.trim()) {
      this.error = 'Укажите дату «нужен до» или снимите галочку.';
      return;
    }
    this.submitting = true;
    this.error = null;

    const needed = this.limitAccessByDate && this.neededUntil.trim() ? this.neededUntil.trim() : undefined;
    const failed: AccessCartLine[] = [];
    let lastErr: string | null = null;
    let ok = 0;

    for (const line of lines) {
      const body: Record<string, unknown> = {
        items: [{ access_role_id: line.accessRoleId, justification: line.justification.trim() }],
      };
      if (needed) {
        body['needed_until'] = needed;
      }
      try {
        await firstValueFrom(this.http.post<{ id: string }>('/api/requests', body));
        ok++;
      } catch (err) {
        failed.push(line);
        lastErr = apiErrorMessage(err);
      }
    }

    this.submitting = false;

    if (ok === lines.length) {
      this.cart.clear();
      void this.router.navigate(['/requests'], { queryParams: { submitted: String(ok) } });
      return;
    }

    if (ok > 0) {
      this.cart.setItems(failed);
      this.error = `Создано отдельных заявок: ${ok}. В корзине остались позиции (${failed.length}), которые не удалось отправить.${lastErr ? ` Последняя ошибка: ${lastErr}` : ''}`;
      return;
    }

    this.error = lastErr ?? 'Не удалось создать заявки.';
  }
}
