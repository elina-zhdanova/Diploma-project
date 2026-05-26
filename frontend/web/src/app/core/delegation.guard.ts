import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { map, catchError, of } from 'rxjs';
import { AuthService } from './auth.service';

/** Раздел делегирования — только назначенные согласующие в цепочках ролей доступа (см. Me.can_delegate). */
export const delegationGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const u = auth.user();
  if (u?.can_delegate === true) {
    return true;
  }
  if (u?.can_delegate === false) {
    return router.createUrlTree(['/store']);
  }
  return auth.ensureProfile().pipe(
    map((me) => (me?.can_delegate ? true : router.createUrlTree(['/store']))),
    catchError(() => of(router.createUrlTree(['/store']))),
  );
};
