import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, of, tap, switchMap, catchError } from 'rxjs';

export interface LoginResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
}

export interface Me {
  id: string;
  login: string;
  full_name: string;
  email: string;
  roles: string[];
  /** Согласующий в цепочке хотя бы одной роли доступа (роль → ресурс → ИС) — может пользоваться делегированием. */
  can_delegate?: boolean;
  /** IAM_MOCK: данные пользователей из локальной БД вместо корпоративного каталога. */
  iam_catalog_emulated?: boolean;
  /** AI_MOCK: правила оценки риска на стороне приложения. */
  ai_rules_emulated?: boolean;
}

const TOKEN_KEY = 'itshop_token';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly http = inject(HttpClient);

  readonly token = signal<string | null>(null);
  readonly user = signal<Me | null>(null);

  constructor() {
    const t = sessionStorage.getItem(TOKEN_KEY);
    if (t) {
      this.token.set(t);
    }
  }

  isLoggedIn(): boolean {
    return !!this.token();
  }

  loadMe(): Observable<Me> {
    return this.http.get<Me>('/api/auth/me').pipe(
      tap((u) => this.user.set(u))
    );
  }

  loginAndLoadMe(login: string, password: string): Observable<Me> {
    return this.http.post<LoginResponse>('/api/auth/login', { login, password }).pipe(
      tap((res) => {
        sessionStorage.setItem(TOKEN_KEY, res.access_token);
        this.token.set(res.access_token);
      }),
      switchMap(() => this.loadMe())
    );
  }

  logout(): void {
    sessionStorage.removeItem(TOKEN_KEY);
    this.token.set(null);
    this.user.set(null);
  }

  /** Если есть токен, но нет профиля — подгружаем */
  ensureProfile(): Observable<Me | null> {
    if (!this.token()) {
      return of(null);
    }
    if (this.user()) {
      return of(this.user()!);
    }
    return this.loadMe().pipe(
      catchError(() => {
        this.logout();
        return of(null);
      })
    );
  }
}
