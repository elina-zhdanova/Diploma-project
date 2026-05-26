import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from './auth.service';

/** Доступ к маршрутам администрирования — только rbac-роль `admin` в профиле. */
export const adminGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  if (auth.user()?.roles?.includes('admin')) {
    return true;
  }
  router.navigate(['/store']);
  return false;
};
