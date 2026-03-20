# Master Plan: Expense Tracker → Production-Ready за 7 дней

## Context

Текущий проект — рабочий CLI + скелет REST API с JSON-хранилищем. Проблемы: сервер имеет свой дублирующий `Expense` struct (без Date, Comment), не использует `internal/model` и `internal/storage`, нет слоёв архитектуры, ID-генерация ломается после удалений, глобальный mutable state без mutex. Цель — превратить это в production-ready веб-приложение с PostgreSQL, авторизацией, красивым UI и деплоем в Docker.

---

## Архитектурные решения

### 1. Роутинг: ТОЛЬКО нативный `http.NewServeMux()`
Никаких chi, gorilla/mux, echo. Используем фичи Go 1.22+: `POST /api/v1/expenses`, `r.PathValue("id")`. Это уже есть в текущем `cmd/server/main.go` — сохраняем подход.

### 2. Разделение API и Web роутов
- `/api/v1/*` — строго JSON (для CLI-клиента и внешних потребителей)
- `/` и далее — HTML рендеринг через `html/template` + HTMX (для браузера)
- Оба набора роутов вызывают одни и те же сервисы, но через разные хендлеры

### 3. CLI = HTTP-клиент (с Дня 2)
CLI перестаёт ходить в хранилище напрямую. Вместо этого стучится в `http://localhost:8080/api/v1/...`. Аналогия: CLI — это как `curl` с красивым TUI, а не отдельное приложение с прямым доступом к данным.

### 4. Тестирование
Unit-тесты для service-слоя (мокаем интерфейс репозитория) + integration-тесты для хендлеров (httptest.NewServer). Добавляются на День 7.

---

## Архитектурное ревью текущего кода

### Что сломано и почему:

1. **Дублирование Expense struct** — `cmd/server/main.go` определяет свой struct (ID, Category, Amount), игнорируя `internal/model.Expense` (ID, Date, Amount, Category, Comment). Аналогия с C: определил один и тот же struct в двух .c файлах вместо общего .h
2. **Нет интерфейса репозитория** — `internal/storage` экспортирует голые функции `SaveExpenses/LoadExpenses` для всего среза. Нельзя подменить JSON на PostgreSQL без переписывания всех вызовов. В Java: конкретный класс без интерфейса
3. **Хрупкая ID-генерация** — сервер использует `len(expenses)+1`, что ломается после удалений
4. **Нет сервисного слоя** — хендлеры напрямую мутируют глобальный слайс. Бизнес-логика размазана по CLI меню
5. **Глобальный mutable state** — `var expenses = []Expense{...}` без mutex = data race при конкурентных HTTP запросах. В C: shared global array без `pthread_mutex_t`
6. **Нет типизированных ошибок** — голые строки вместо domain errors

---

## Целевая архитектура (Hexagonal)

```
go-expense-tracker/
├── cmd/
│   ├── server/main.go              # Wiring: config → DB → repos → services → handlers → router
│   └── cli/
│       ├── main.go                 # CLI entry point (HTTP-клиент с Дня 2)
│       └── client.go              # HTTP-клиент для /api/v1/*
├── internal/
│   ├── config/config.go            # Env-based конфиг (DB_URL, PORT, JWT_SECRET, OAuth)
│   ├── domain/
│   │   ├── expense.go              # Entity: Expense (чистые данные, без DB-тегов)
│   │   ├── user.go                 # Entity: User
│   │   └── errors.go               # ErrNotFound, ErrValidation, ErrUnauthorized
│   ├── repository/
│   │   ├── expense_repository.go   # Интерфейс ExpenseRepository
│   │   ├── user_repository.go      # Интерфейс UserRepository
│   │   ├── postgres/               # pgx-реализации
│   │   │   ├── expense_repo.go
│   │   │   └── user_repo.go
│   │   └── json/
│   │       └── expense_repo.go     # JSON-реализация (только День 1, потом deprecated)
│   ├── service/
│   │   ├── expense_service.go      # Бизнес-логика: CRUD + totals + фильтрация
│   │   └── auth_service.go         # Register, Login, VerifyToken, OAuth
│   ├── handler/
│   │   ├── api_expense.go          # /api/v1/expenses — JSON хендлеры
│   │   ├── api_auth.go             # /api/v1/auth/* — JSON хендлеры
│   │   ├── web_page.go             # /, /login, /dashboard — HTML рендеринг
│   │   ├── web_expense.go          # /expenses/* — HTMX HTML partials
│   │   └── response.go             # writeJSON/writeError хелперы
│   └── middleware/
│       ├── logging.go              # Request logging (log/slog)
│       ├── auth.go                 # JWT/cookie verification
│       └── ratelimit.go            # Token bucket per IP
├── migrations/                     # SQL миграции
├── templates/                      # html/template + HTMX
│   ├── layouts/base.html
│   ├── pages/{login,register,dashboard,expenses}.html
│   └── partials/{expense_row,expense_list}.html
├── static/css/
├── scripts/seed.go                 # Генерация 1000+ фейковых транзакций
├── Dockerfile
└── docker-compose.yml
```

**Маршруты сервера:**
```
# API (JSON) — для CLI и внешних клиентов
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
GET    /api/v1/expenses
POST   /api/v1/expenses
GET    /api/v1/expenses/{id}
PUT    /api/v1/expenses/{id}
DELETE /api/v1/expenses/{id}
GET    /api/v1/stats/monthly
GET    /api/v1/stats/by-category

# Web (HTML) — для браузера
GET    /                          → redirect to /dashboard or /login
GET    /login                     → login page
GET    /register                  → register page
GET    /auth/google               → OAuth redirect
GET    /auth/google/callback      → OAuth callback
GET    /auth/github               → OAuth redirect
GET    /auth/github/callback      → OAuth callback
GET    /dashboard                 → dashboard с графиками
GET    /expenses                  → список расходов (HTMX)
POST   /expenses                  → добавить (HTMX partial response)
PUT    /expenses/{id}             → обновить (HTMX partial response)
DELETE /expenses/{id}             → удалить (HTMX partial response)
```

---

## 7-Day Sprint Plan

### День 1: Clean Architecture Refactor (4-5 ч)

**Цель:** Слоистая архитектура. Сервер компилируется и работает с JSON-бэкендом. API роуты на `/api/v1/*`.

**Файлы для создания:**
- `internal/domain/expense.go` — перенести Expense struct из `internal/model/expense.go`, убрать утилиты
- `internal/domain/errors.go` — `var ErrNotFound`, `var ErrValidation`
- `internal/repository/expense_repository.go` — интерфейс с `context.Context` в каждом методе:
  ```go
  type ExpenseRepository interface {
      GetAll(ctx context.Context) ([]domain.Expense, error)
      GetByID(ctx context.Context, id int) (*domain.Expense, error)
      Create(ctx context.Context, expense *domain.Expense) error
      Update(ctx context.Context, expense *domain.Expense) error
      Delete(ctx context.Context, id int) error
  }
  ```
- `internal/repository/json/expense_repo.go` — struct с `sync.Mutex`, реализует интерфейс, переиспользует логику из `internal/storage/storage.go`
- `internal/service/expense_service.go` — бизнес-логика (валидация amount > 0, category не пуст), сюда переезжает `CalculateTotal`
- `internal/handler/response.go` — `writeJSON`, `writeError`
- `internal/handler/api_expense.go` — struct с dependency на service, метод `RegisterRoutes(mux)`, роуты `/api/v1/expenses*`

**Файлы для переписывания:**
- `cmd/server/main.go` — чистый wiring: repo → service → handler → mux → ListenAndServe
- `cmd/cli/main.go` + `menu.go` — пока обновить импорты на domain/repository (HTTP-клиент на День 2)

**Файлы для удаления:**
- `internal/model/` → заменено на `internal/domain/`
- `internal/storage/` → заменено на `internal/repository/json/`

**Роутинг:** строго `http.NewServeMux()` + `POST /api/v1/expenses`, `GET /api/v1/expenses/{id}` и т.д. (Go 1.22+ паттерны)

**Проверка:** `go build ./...` компилируется, curl на все 5 CRUD эндпоинтов `/api/v1/*` работает, данные сохраняются в expenses.json

---

### День 2: PostgreSQL + Docker + CLI→HTTP-клиент (5-6 ч)

**Цель:** Приложение в Docker, данные в PostgreSQL. CLI стучится в REST API вместо прямого доступа к файлу.

**Что изучить:** Основы SQL (CREATE TABLE, SELECT, INSERT, UPDATE, DELETE, JOIN), что такое Docker image/container/volume, docker-compose сервисы.

**Файлы:**
- `internal/config/config.go` — чтение из env: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, SERVER_PORT
- `migrations/000001_create_expenses.up.sql` — `CREATE TABLE expenses (id SERIAL PRIMARY KEY, date TIMESTAMPTZ NOT NULL DEFAULT NOW(), amount DECIMAL(12,2) NOT NULL, category VARCHAR(100) NOT NULL, comment TEXT DEFAULT '')`
- `migrations/000001_create_expenses.down.sql` — `DROP TABLE IF EXISTS expenses`
- `internal/repository/postgres/expense_repo.go` — pgx-реализация ExpenseRepository
- `Dockerfile` — multi-stage build (golang:1.25-alpine → alpine:latest, ~15MB итоговый образ)
- `docker-compose.yml` — сервисы: db (postgres:16-alpine) + app (build: .)

**Рефакторинг CLI:**
- `cmd/cli/client.go` — HTTP-клиент: struct `APIClient` с базовым URL, методы `GetAll()`, `Create()`, `GetByID()`, `Update()`, `Delete()` — каждый делает HTTP-запрос к `/api/v1/expenses*` и парсит JSON-ответ
- `cmd/cli/menu.go` — заменить все вызовы repository на вызовы APIClient
- `cmd/cli/main.go` — создать APIClient с `http://localhost:8080`, передать в RunMenu
- Удалить импорт `repository/json` из CLI — он больше не нужен

**Зависимости:** `github.com/jackc/pgx/v5`, `github.com/golang-migrate/migrate/v4`

**Проверка:** `docker-compose up --build`, CLI подключается к серверу через HTTP и работает как раньше

---

### День 3: Авторизация (5-6 ч)

**Цель:** Регистрация/логин по email+password, JWT в cookie, расходы привязаны к пользователю.

**Файлы:**
- `internal/domain/user.go` — User entity
- `internal/repository/user_repository.go` — интерфейс
- `internal/repository/postgres/user_repo.go` — pgx-реализация
- `migrations/000002_create_users.up.sql`
- `migrations/000003_add_user_id_to_expenses.up.sql`
- `internal/service/auth_service.go` — Register (bcrypt hash), Login (compare + JWT), VerifyToken
- `internal/handler/api_auth.go` — POST /api/v1/auth/register, /api/v1/auth/login, /api/v1/auth/logout
- `internal/middleware/auth.go` — RequireAuth middleware, извлекает userID из JWT cookie → context

**Обновить CLI:** добавить команды login/register в меню, сохранять JWT cookie для последующих запросов

**Зависимости:** `golang.org/x/crypto` (bcrypt), `github.com/golang-jwt/jwt/v5`

**Проверка:** регистрация → логин → получение cookie → CRUD расходов только своих

---

### День 4: OAuth2 (Google + GitHub) (4-5 ч)

**Цель:** Вход через Google и GitHub. Привязка по email.

**Файлы:**
- `internal/service/oauth_service.go` — конфиг провайдеров, GetAuthURL, HandleCallback
- `internal/handler/oauth_handler.go` — GET /auth/google, /auth/google/callback, аналогично для GitHub
- `migrations/000004_add_oauth_fields.up.sql` — oauth_provider, oauth_id, password_hash NULLABLE

**Зависимость:** `golang.org/x/oauth2`

**Проверка:** OAuth flow работает с localhost (или ngrok для callback)

---

### День 5: Frontend — html/template + HTMX + TailwindCSS (5-6 ч)

**Цель:** Полноценный UI. HTMX для динамики, минимум JS.

**Файлы:**
- `templates/layouts/base.html` — скелет с Tailwind CDN + HTMX script
- `templates/pages/login.html` — форма + кнопки Google/GitHub
- `templates/pages/register.html`
- `templates/pages/expenses.html` — таблица расходов + форма добавления
- `templates/partials/expense_row.html` — `<tr>` для HTMX swap
- `internal/handler/web_page.go` — рендеринг страниц (/, /login, /register, /dashboard)
- `internal/handler/web_expense.go` — HTMX-хендлеры (POST/PUT/DELETE /expenses → возвращают HTML partials)
- Обновить `cmd/server/main.go` — `http.FileServer` для static/, регистрация web-роутов

**Web-роуты** отдают HTML (для браузера). **API-роуты** (`/api/v1/*`) остаются строго JSON (для CLI).

**HTMX-атрибуты** вместо JS: `hx-delete="/expenses/5" hx-target="closest tr" hx-swap="outerHTML"`

**Проверка:** браузер → логин → список расходов → добавить/удалить без перезагрузки страницы

---

### День 6: Dashboard + Графики + Seed Script (4-5 ч)

**Цель:** Красивый дашборд с Chart.js. Демо-данные для работодателя.

**Файлы:**
- `scripts/seed.go` — создаёт пользователя test@demo.com / 123456, генерирует 1000+ транзакций за 12 месяцев с категориями: Такси, Кофе, Продукты, Подписки, Транспорт, Одежда, Развлечения, Образование
- `templates/pages/dashboard.html` — карточки (итого за месяц, среднее в день, топ категория) + `<canvas>` для Chart.js
- `internal/handler/api_expense.go` — добавить `GET /api/v1/stats/monthly`, `GET /api/v1/stats/by-category`
- SQL: `SELECT date_trunc('month', date), SUM(amount) FROM expenses WHERE user_id = $1 GROUP BY 1`

**Проверка:** `go run scripts/seed.go` → логин test@demo.com → дашборд с графиками

---

### День 7: Тесты, Graceful Shutdown, Rate Limiting, Deploy (5-6 ч)

**Цель:** Production-ready. Тесты. Один `docker-compose up` запускает всё.

**Тесты:**
- `internal/service/expense_service_test.go` — unit-тесты с мок-репозиторием:
  - Создать `mockExpenseRepo` struct, реализующий `ExpenseRepository` интерфейс
  - Тестировать: Create валидация (amount ≤ 0, пустая категория), GetAll, GetByID с ErrNotFound, GetTotal
  - Аналогия: в Java это Mockito mock для DAO, в Go — просто struct с нужными методами
- `internal/handler/api_expense_test.go` — integration-тесты:
  - `httptest.NewServer` + реальный сервис с мок-репозиторием
  - Тестировать: POST 201, GET 200, GET 404, DELETE 204, невалидный JSON → 400

**Инфраструктура:**
- Обновить `cmd/server/main.go` — graceful shutdown через `signal.Notify` + `srv.Shutdown(ctx)`
- `internal/middleware/logging.go` — `log/slog`, логирование method/path/status/duration
- `internal/middleware/ratelimit.go` — `golang.org/x/time/rate`, token bucket per IP
- Обновить `docker-compose.yml` — healthcheck для postgres, restart policy, `.env` файл
- Обновить `Dockerfile` — COPY migrations, templates, static

**Зависимость:** `golang.org/x/time`

**Финальная проверка:**
1. `go test ./...` — все тесты зелёные
2. `docker-compose up --build` стартует чисто
3. `go run scripts/seed.go` заполняет данные
4. Логин test@demo.com / 123456 → дашборд с графиками
5. CLI подключается к API, все команды работают
6. OAuth Google/GitHub работает
7. Rate limiting срабатывает при спаме запросов
8. `Ctrl+C` → graceful shutdown в логах
9. Рестарт → данные на месте (PostgreSQL volume)

---

## Внешние зависимости (все pure Go, без CGO)

| Пакет | Зачем | День |
|-------|-------|------|
| `github.com/jackc/pgx/v5` | PostgreSQL драйвер | 2 |
| `github.com/golang-migrate/migrate/v4` | Миграции БД | 2 |
| `golang.org/x/crypto` | bcrypt хеширование | 3 |
| `github.com/golang-jwt/jwt/v5` | JWT токены | 3 |
| `golang.org/x/oauth2` | OAuth2 клиент | 4 |
| `golang.org/x/time` | Rate limiter | 7 |

---

## День 1: Пошаговые инструкции

### Шаг 1: Создай domain layer (30 мин)

Создай `internal/domain/expense.go`:
- Перенеси `Expense` struct из `internal/model/expense.go`, пакет `domain`
- Оставь только struct + конструктор `NewExpense(category, amount, comment) Expense`
- **НЕ** переноси `CalculateTotal`, `NextID`, `FindExpenseByID`, `DeleteExpenseFromSlice` — они уйдут в service/repository

Создай `internal/domain/errors.go`:
```go
var (
    ErrNotFound   = errors.New("not found")
    ErrValidation = errors.New("validation error")
)
```

### Шаг 2: Определи интерфейс репозитория (20 мин)

Создай `internal/repository/expense_repository.go` с интерфейсом `ExpenseRepository` (5 методов, все с `context.Context`).

**Зачем ctx:** PostgreSQL на День 2 потребует ctx для таймаутов запросов. Добавив сейчас, не придётся менять сигнатуры потом.

### Шаг 3: Реализуй JSON-репозиторий (45 мин)

Создай `internal/repository/json/expense_repo.go`:
- Struct: `filePath string` + `sync.Mutex` + `expenses []domain.Expense`
- Конструктор: загружает файл при старте (как текущий `LoadExpenses`)
- Все методы: lock → операция → save to file → unlock
- Mutex критичен: это `pthread_mutex_t` из C, без него — data race

### Шаг 4: Создай сервисный слой (45 мин)

`internal/service/expense_service.go`:
- Struct принимает `repository.ExpenseRepository` через конструктор (DI как передача vtable в C)
- Валидация: amount > 0, category не пуст
- `GetTotal(ctx)` — сюда переезжает логика `CalculateTotal`

### Шаг 5: HTTP хендлеры (60 мин)

`internal/handler/response.go` — writeJSON, writeError
`internal/handler/api_expense.go`:
- Struct с зависимостью на service
- HandleCreate, HandleGetAll, HandleGetByID, HandleUpdate, HandleDelete
- Маппинг ошибок: `errors.Is(err, domain.ErrNotFound)` → 404, ErrValidation → 400, else → 500
- `RegisterRoutes(mux)` — регистрирует роуты на `/api/v1/expenses*`

### Шаг 6: Перепиши cmd/server/main.go (30 мин)

Чистый wiring:
```go
repo → service → handler → mux → ListenAndServe
```
Это DI без фреймворка. В Spring это @Autowired, в C — передача struct с function pointers в init().

### Шаг 7: Обнови CLI (20 мин)

Обнови импорты в `cmd/cli/main.go` и `menu.go` на `domain` и `repository/json`. (HTTP-клиент — на День 2)

### Шаг 8: Удали старое (5 мин)

Удали `internal/model/`, `internal/storage/`. Запусти `go build ./...`.

### Шаг 9: Тестируй (15 мин)

```bash
go run cmd/server/main.go
curl localhost:8080/api/v1/expenses
curl -X POST localhost:8080/api/v1/expenses -d '{"category":"Test","amount":500,"comment":"test"}'
curl localhost:8080/api/v1/expenses/1
curl -X PUT localhost:8080/api/v1/expenses/1 -d '{"category":"Updated","amount":999,"comment":"changed"}'
curl -X DELETE localhost:8080/api/v1/expenses/1
```
