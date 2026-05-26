import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-latech-logo',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './latech-logo.component.html',
  styleUrl: './latech-logo.component.scss',
})
export class LatechLogoComponent {
  @Input() compact = false;
  /** Узкая шапка: компактная марка, подпись скрывается уже чем 1100px по ширине. */
  @Input() compactNav = false;
  @Input() variant: 'dark' | 'light' = 'dark';
  /** Подзаголовок вместо «Gatekeeper» */
  @Input() productLine = 'IT Shop';
  /** Основная строка вместо «Access Store» */
  @Input() titleLine = 'Access Store';
}
