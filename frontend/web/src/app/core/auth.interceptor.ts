import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { AuthService } from './auth.service';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const t = auth.token();
  if (t) {
    req = req.clone({ setHeaders: { Authorization: `Bearer ${t}` } });
  }
  return next(req);
};
