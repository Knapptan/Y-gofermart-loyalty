# Гофермарт — накопительная система лояльности

HTTP API для управления баллами лояльности: регистрация, приём заказов,
начисление и списание баллов. Реализация дипломного проекта «Яндекс Практикум».

## Возможности

- Регистрация и аутентификация по JWT
- Загрузка номеров заказов с проверкой по алгоритму Луна
- Фоновый воркер опрашивает внешний сервис расчёта баллов
- Начисление и списание баллов в транзакциях с блокировкой строки
- Все данные в PostgreSQL, миграции применяются автоматически при старте

## Требования

- Go 1.26+
- PostgreSQL 14+

## Запуск

```bash
export DATABASE_URI="postgres://user:pass@localhost:5432/gophermart?sslmode=disable"
export RUN_ADDRESS="localhost:8080"
export ACCRUAL_SYSTEM_ADDRESS="localhost:8081"
export JWT_SECRET="your-secret-here"
go run ./cmd/gophermart
```

Флаги дублируют переменные окружения: `-a`, `-d`, `-r`, `-jwt-secret`.
Приоритет: **env > flag > default**.

## API

| Метод | Путь                         | Описание         | Auth |
| ----- | ---------------------------- | ---------------- | ---- |
| POST  | `/api/user/register`         | Регистрация      | —    |
| POST  | `/api/user/login`            | Логин            | —    |
| POST  | `/api/user/orders`           | Загрузка заказа  | ✓    |
| GET   | `/api/user/orders`           | Список заказов   | ✓    |
| GET   | `/api/user/balance`          | Баланс           | ✓    |
| POST  | `/api/user/balance/withdraw` | Списание         | ✓    |
| GET   | `/api/user/withdrawals`      | История списаний | ✓    |

Аутентификация: заголовок `Authorization: Bearer <token>`,
токен возвращается в том же заголовке ответа на `/register` и `/login`.

### Примеры

```bash
# Регистрация
curl -i -X POST localhost:8080/api/user/register \
  -H "Content-Type: application/json" \
  -d '{"login":"alice","password":"secret"}'

# Загрузка заказа
curl -i -X POST localhost:8080/api/user/orders \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: text/plain" \
  -d '12345678903'

# Баланс
curl -i localhost:8080/api/user/balance \
  -H "Authorization: Bearer <token>"
```

## Тесты

```bash
go test ./... -v
```

## Архитектура

```
cmd/gophermart      — точка входа
internal/config     — конфигурация
internal/model      — доменные модели
internal/repository — интерфейсы + реализации (postgres)
internal/service    — бизнес-логика
internal/handler    — HTTP-хендлеры
internal/middleware — JWT-аутентификация
internal/accrual    — клиент внешнего сервиса
migrations          — SQL-миграции
```

## Внешний сервис расчёта

Ожидается `GET /api/orders/{number}` со следующими ответами:

- `200` — `{"order":"...","status":"PROCESSED","accrual":500}`
- `204` — заказ не зарегистрирован
- `429` — rate limit (заголовок `Retry-After`)
- `500` — ошибка

Воркер опрашивает `NEW`/`PROCESSING` заказы раз в 5 секунд.
