import { Routes } from '@angular/router';
import { authGuard } from './core/auth.guard';
import { adminGuard } from './core/admin.guard';
import { delegationGuard } from './core/delegation.guard';

export const routes: Routes = [
  { path: 'login', loadComponent: () => import('./pages/login/login.component').then((m) => m.LoginComponent) },
  {
    path: '',
    loadComponent: () => import('./layout/shell.component').then((m) => m.ShellComponent),
    canActivate: [authGuard],
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'store' },
      { path: 'cart', redirectTo: '/store/cart', pathMatch: 'full' },
      {
        path: 'store',
        loadComponent: () => import('./pages/store/store-shell.component').then((m) => m.StoreShellComponent),
        children: [
          {
            path: '',
            loadComponent: () => import('./pages/store/store-systems.component').then((m) => m.StoreSystemsComponent),
          },
          {
            path: 'cart',
            loadComponent: () => import('./pages/store/access-cart.component').then((m) => m.AccessCartComponent),
          },
          {
            path: 'system/:systemId',
            loadComponent: () =>
              import('./pages/store/store-system-roles.component').then((m) => m.StoreSystemRolesComponent),
          },
          {
            path: ':id',
            loadComponent: () => import('./pages/access-detail/access-detail.component').then((m) => m.AccessDetailComponent),
            data: { panel: true },
          },
        ],
      },
      {
        path: 'access/:id',
        loadComponent: () => import('./pages/access-detail/access-detail.component').then((m) => m.AccessDetailComponent),
      },
      { path: 'profile', loadComponent: () => import('./pages/profile/profile.component').then((m) => m.ProfileComponent) },
      {
        path: 'requests',
        loadComponent: () => import('./pages/requests-workspace/requests-workspace.component').then((m) => m.RequestsWorkspaceComponent),
        children: [
          {
            path: ':id',
            loadComponent: () => import('./pages/request-detail/request-detail.component').then((m) => m.RequestDetailComponent),
            data: { panel: true },
          },
        ],
      },
      {
        path: 'inbox',
        loadComponent: () => import('./pages/inbox-workspace/inbox-workspace.component').then((m) => m.InboxWorkspaceComponent),
        children: [
          {
            path: ':id',
            loadComponent: () => import('./pages/request-detail/request-detail.component').then((m) => m.RequestDetailComponent),
            data: { panel: true },
          },
        ],
      },
      {
        path: 'delegation',
        canActivate: [delegationGuard],
        loadComponent: () => import('./pages/delegation/delegation.component').then((m) => m.DelegationComponent),
      },
      {
        path: 'tz',
        canActivate: [adminGuard],
        loadComponent: () => import('./pages/tz-modules/tz-modules.component').then((m) => m.TzModulesComponent),
      },
      {
        path: 'audit',
        canActivate: [adminGuard],
        loadComponent: () => import('./pages/audit/audit.component').then((m) => m.AuditComponent),
      },
      {
        path: 'admin',
        canActivate: [adminGuard],
        loadComponent: () => import('./pages/admin/admin.component').then((m) => m.AdminComponent),
      },
    ],
  },
  { path: '**', redirectTo: '/store' },
];
