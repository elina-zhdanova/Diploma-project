# IT Shop — каркас по ТЗ (Angular + Go + PostgreSQL + MinIO)

Отдельный контур от текущего Gatekeeper (`gatekeeper-frontend/` + Python backend). Сценарии переносятся сюда поэтапно.

## Структура

```text
itshop/
  backend/           # Go API + sqlc + миграции
  deploy/            # docker-compose: postgres, minio, api, web
  frontend/web/      # Angular 19 MVP
```

## Быстрый старт (Docker)

```bash
cd itshop/deploy
docker compose up -d --build
```

| Сервис | URL |
|---|---|
| API | http://localhost:8081 |
| Health | http://localhost:8081/health |
| Angular web | http://localhost:4200 |
| PostgreSQL | localhost:5433 (`itshop` / `itshop` / `itshop`) |
| MinIO API | http://localhost:9000 |
| MinIO Console | http://localhost:9001 (`minioadmin` / `minioadmin`) |

## Frontend (Angular)

```bash
cd itshop/frontend/web
npm install
npm start
```

`npm start` проксирует `/api` и `/internal` на API по умолчанию `http://localhost:8080` (как у `go run ./cmd/api`). Если backend в Docker с пробросом **8081→8080**, задайте `ITSHOP_API_PROXY=http://localhost:8081` перед `npm start`.

## Backend (локально без Docker)

Требуется Go 1.22+:

```bash
cd itshop/backend
go run ./cmd/api
```

## Что уже перенесено

- JWT auth: `POST /api/auth/login`, `GET /api/auth/me`
- Catalog: `GET /api/systems`, `GET /api/resources`, `GET /api/access-roles`
- Requests: создание, мои заявки, детали, отзыв
- Approvals: inbox, approve/reject/delegate
- Audit: `GET /api/audit`
- Attachments: upload + presigned URL через MinIO
- Internal mock: IAM (`/internal/iam/user`) и rule-based AI (`/internal/ai/analyze`)

Подробнее о целевой архитектуре: [`../docs/architecture.md`](../docs/architecture.md).
