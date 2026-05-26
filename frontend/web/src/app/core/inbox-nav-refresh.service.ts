import { Injectable } from '@angular/core';
import { Subject } from 'rxjs';

/** Сообщает шапке пересчитать «входящие» (например после approve/reject без смены URL). */
@Injectable({ providedIn: 'root' })
export class InboxNavRefreshService {
  private readonly bus = new Subject<void>();
  readonly refresh$ = this.bus.asObservable();

  ping(): void {
    this.bus.next();
  }
}
