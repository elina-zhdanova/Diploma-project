import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

export type TzStatus = 'done' | 'partial' | 'planned';

export interface TzModuleLink {
  label: string;
  route: string;
}

export interface TzModuleBlock {
  code: string;
  title: string;
  requirements: string[];
  status: TzStatus;
  links: TzModuleLink[];
  note?: string;
}

@Component({
  selector: 'app-tz-modules',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './tz-modules.component.html',
  styleUrl: './tz-modules.component.scss',
})
export class TzModulesComponent {
  readonly blocks: TzModuleBlock[] = [
    {
      code: '5.1',
      title: 'Управление пользователями',
      requirements: [
        'аутентификация пользователей',
        'разграничение прав доступа по ролям',
        'роли: инициатор, согласующий, специалист ИБ, администратор',
        'получение данных о пользователях из корпоративных источников (или эмуляция)',
      ],
      status: 'done',
      links: [
        { label: 'Вход (аутентификация)', route: '/login' },
        { label: 'Профиль: роли и режим каталога IAM', route: '/profile' },
      ],
      note: 'В профиле отображается, включена ли эмуляция корпоративного каталога (IAM_MOCK) и правил риска (AI_MOCK).',
    },
    {
      code: '5.2',
      title: 'Каталог доступов',
      requirements: [
        'информационные системы, ресурсы, роли доступа',
        'поиск и фильтрация',
        'описания ролей и уровни доступа (риск)',
      ],
      status: 'done',
      links: [{ label: 'Каталог ролей', route: '/store' }],
    },
    {
      code: '5.3',
      title: 'Управление заявками',
      requirements: [
        'создание заявок',
        'несколько доступов в одной заявке',
        'вложения к заявке',
        'список и детали заявки',
        'отзыв заявки до завершения',
      ],
      status: 'done',
      links: [
        { label: 'Мои заявки', route: '/requests' },
        { label: 'Каталог для заказа доступа', route: '/store' },
      ],
      note: 'В карточке заявки: позиции, обоснование, вложения (загрузка и скачивание для инициатора и согласующих), отзыв инициатором.',
    },
    {
      code: '5.4',
      title: 'Согласование заявок',
      requirements: [
        'маршрут и многоэтапное согласование',
        'утверждение, отклонение, делегирование',
        'фиксация решений и комментариев',
        'последовательность этапов',
      ],
      status: 'done',
      links: [
        { label: 'На согласование (шаг маршрута)', route: '/inbox' },
        { label: 'Детали заявки: цепочка и решения', route: '/requests' },
      ],
      note: 'Во входящих показан номер текущего шага; в деталях — полная цепочка с комментариями и датами решений.',
    },
    {
      code: '5.5',
      title: 'Делегирование',
      requirements: [
        'передача полномочий согласования',
        'ограничение по времени',
        'учёт при маршрутизации',
      ],
      status: 'done',
      links: [{ label: 'Панель делегирования', route: '/delegation' }],
      note: 'Период по датам; доступ у согласующих из цепочек ролей; заместитель видит пометку при решении шага.',
    },
    {
      code: '5.6',
      title: 'Интеграция с IAM/IDM',
      requirements: [
        'передача утверждённых заявок во внешний контур',
        'статус выполнения, обработка ошибок, повторы',
      ],
      status: 'planned',
      links: [],
      note: 'Очередь исполнения и контракт с IAM — по roadmap backend.',
    },
    {
      code: '5.7',
      title: 'Интеллектуальный модуль',
      requirements: [
        'анализ параметров заявки',
        'уровень риска и рекомендации',
        'автообработка низкорисковых заявок',
        'сохранение результатов анализа',
      ],
      status: 'done',
      links: [{ label: 'Заявка: блок анализа и риска', route: '/requests' }],
      note: 'В деталях заявки — рекомендация, уверенность, обоснование и балл риска (запись в ai_decisions при AI_MOCK).',
    },
    {
      code: '5.8',
      title: 'Журнал аудита',
      requirements: [
        'регистрация действий',
        'история изменений заявок',
        'фильтрация и поиск',
        'анализ событий ИБ',
      ],
      status: 'done',
      links: [{ label: 'Журнал аудита', route: '/audit' }],
      note: 'Фильтры, таблица, сводка по типам; доступ только для admin.',
    },
    {
      code: '5.9',
      title: 'Администрирование',
      requirements: [
        'каталог доступов',
        'маршруты согласования',
        'роли пользователей',
        'метрики и логи',
      ],
      status: 'done',
      links: [{ label: 'Панель администрирования (роль admin)', route: '/admin' }],
      note: 'Сводка каталога, создание роли с цепочкой согласующих, пользователи и метрики; ссылки в каталог и аудит.',
    },
  ];

  statusClass(s: TzStatus): string {
    switch (s) {
      case 'done':
        return 'tz-mod-badge tz-mod-badge--done';
      case 'partial':
        return 'tz-mod-badge tz-mod-badge--partial';
      default:
        return 'tz-mod-badge tz-mod-badge--planned';
    }
  }

  statusText(s: TzStatus): string {
    switch (s) {
      case 'done':
        return 'В интерфейсе';
      case 'partial':
        return 'Частично';
      default:
        return 'Планируется';
    }
  }
}
