# Security

NetQuest — учебное приложение, но базовые security controls уже включены.

## Auth

- Пользователь может создать local account или войти через demo login.
- Access token используется frontend для API-запросов.
- Refresh token хранится в HttpOnly cookie `netquest_refresh_token`.
- Backend не возвращает refresh token в JSON response после login/register/demo/refresh.
- Logout отзывает refresh token.

## Cookies

Refresh cookie:

- `HttpOnly`;
- `SameSite=Lax`;
- `Secure` в production;
- path `/api/v1/auth`.

Frontend отправляет cookie через `credentials: "include"`.

## Headers

Backend и frontend добавляют базовые headers:

- `X-Content-Type-Options: nosniff`;
- `Referrer-Policy`;
- `Permissions-Policy`;
- `Content-Security-Policy`.

## Сетевая безопасность

NetQuest не выполняет настоящие DNS-запросы, ICMP Ping или HTTPS-запросы к пользовательским доменам. Все сценарии рассчитываются внутри backend simulation engine.

## Production notes

Для production-запуска обязательно задайте сильные значения:

- `JWT_SECRET`;
- `REFRESH_SECRET`;
- `POSTGRES_PASSWORD`;
- `DOMAIN`;
- `ACME_EMAIL`;
- `PUBLIC_APP_URL`.

Production-compose использует Caddy как HTTPS reverse proxy.
