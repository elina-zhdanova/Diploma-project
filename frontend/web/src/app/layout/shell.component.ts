import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { NavigationEnd, Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { filter } from 'rxjs/operators';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { AuthService } from '../core/auth.service';
import { InboxNavRefreshService } from '../core/inbox-nav-refresh.service';
import { ThemeService } from '../core/theme.service';
import { LatechLogoComponent } from '../components/latech-logo/latech-logo.component';
import { AccessCartService } from '../core/access-cart.service';

@Component({
  selector: 'app-shell',
  standalone: true,
  imports: [CommonModule, RouterOutlet, RouterLink, RouterLinkActive, LatechLogoComponent],
  templateUrl: './shell.component.html',
  styleUrl: './shell.component.scss',
})
export class ShellComponent implements OnInit {
  readonly auth = inject(AuthService);
  readonly theme = inject(ThemeService);
  private readonly router = inject(Router);
  private readonly http = inject(HttpClient);
  private readonly inboxNavRefresh = inject(InboxNavRefreshService);
  readonly accessCart = inject(AccessCartService);

  /** Количество заявок, где пользователь — согласующий на текущем шаге (бейдж у пункта «На согласование»). */
  readonly inboxCount = signal(0);
  /** Пункт админ-панели — только rbac admin */
  readonly isAdmin = computed(() => (this.auth.user()?.roles ?? []).includes('admin'));
  /** Делегирование — только если пользователь в цепочке согласования хотя бы одной роли доступа (ИС). */
  /** readonly canDelegate = computed(() => this.auth.user()?.can_delegate === true);
   * readonly showDelegationNav = computed(() => this.inboxCount() > 0);
  /** Кнопка делегирования в меню: показываем только при наличии входящих согласований. */
  readonly showDelegationNav = computed(() => true);
  

  constructor() {
    this.router.events
      .pipe(
        filter((e): e is NavigationEnd => e instanceof NavigationEnd),
        takeUntilDestroyed(),
      )
      .subscribe(() => this.refreshInboxCount());

    this.inboxNavRefresh.refresh$.pipe(takeUntilDestroyed()).subscribe(() => this.refreshInboxCount());
  }

  ngOnInit(): void {
    this.auth.ensureProfile().subscribe((u) => {
      if (!u && this.auth.isLoggedIn()) {
        this.router.navigate(['/login']);
      } else if (u) {
        this.refreshInboxCount();
      }
    });
  }

  private refreshInboxCount(): void {
    if (!this.auth.isLoggedIn()) {
      this.inboxCount.set(0);
      return;
    }
    this.http.get<{ items: unknown[] }>('/api/approvals/inbox').subscribe({
      next: (res) => this.inboxCount.set((res.items ?? []).length),
      error: () => this.inboxCount.set(0),
    });
  }

  logout(): void {
    this.auth.logout();
    this.inboxCount.set(0);
    this.router.navigate(['/login']);
  }

  toggleTheme(): void {
    this.theme.toggle();
  }

  /** Активен пункт «Каталог», но не на странице корзины (/store/cart). */
  catalogNavActive(): boolean {
    const p = this.router.url.split('?')[0];
    if (p === '/store/cart') {
      return false;
    }
    return p === '/store' || p.startsWith('/store/');
  }
}
