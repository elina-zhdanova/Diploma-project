import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthService } from '../../core/auth.service';
import { apiErrorMessage } from '../../core/api-error';
import { LatechLogoComponent } from '../../components/latech-logo/latech-logo.component';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, FormsModule, LatechLogoComponent],
  templateUrl: './login.component.html',
  styleUrl: './login.component.scss',
})
export class LoginComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  login = 'initiator';
  password = 'password';
  loading = false;
  error: string | null = null;

  submit(): void {
    this.error = null;
    this.loading = true;
    this.auth.loginAndLoadMe(this.login, this.password).subscribe({
      next: () => {
        this.loading = false;
        const ret = this.route.snapshot.queryParamMap.get('returnUrl') || '/store';
        this.router.navigateByUrl(ret);
      },
      error: (err) => {
        this.loading = false;
        this.error = apiErrorMessage(err);
      },
    });
  }
}
