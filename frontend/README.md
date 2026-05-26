# Angular frontend (`web`)

Приложение уже сгенерировано в `itshop/frontend/web`.

## Локальный запуск

```bash
cd itshop/frontend/web
npm install
npm start
```

Прокси (`proxy.conf.js`) по умолчанию направляет на `http://localhost:8080`. Для API в Docker на хосте **8081** используйте `ITSHOP_API_PROXY=http://localhost:8081`.

## Что уже есть

- форма логина (`/api/auth/login`)
- вызов `auth/me`
- загрузка каталогов (`systems/resources/access-roles`)
- создание заявки и действия согласования
- загрузка вложения
- просмотр аудита

Это MVP-экран для поэтапного переноса сценариев Gatekeeper.

