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

### День 3: Авторизация — JWT + bcrypt (5-6 ч)

**Цель:** Регистрация/логин по email+password. JWT в HttpOnly cookie. Каждый расход привязан к user_id. Без авторизации — 401.

**Что изучить:** Как работает bcrypt (hash = salt + cost + digest, сравнение через `bcrypt.CompareHashAndPassword`), JWT (header.payload.signature, claim-ы sub/exp/iat), middleware chain в Go (функция принимает `http.Handler` и возвращает `http.Handler` — как decorator в Python или обёртка в C через function pointer).

**Зависимости:** `go get golang.org/x/crypto` (bcrypt), `go get github.com/golang-jwt/jwt/v5`

#### Шаг 1: Миграция — таблица users (15 мин)

Создай `migrations/000002_create_users.up.sql`:
```sql
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```
Создай `migrations/000002_create_users.down.sql`:
```sql
DROP TABLE IF EXISTS users;
```

#### Шаг 2: Миграция — user_id в expenses (15 мин)

Создай `migrations/000003_add_user_id_to_expenses.up.sql`:
```sql
ALTER TABLE expenses ADD COLUMN user_id INTEGER REFERENCES users(id);
CREATE INDEX idx_expenses_user_id ON expenses(user_id);
```
Создай `migrations/000003_add_user_id_to_expenses.down.sql`:
```sql
ALTER TABLE expenses DROP COLUMN IF EXISTS user_id;
```

**Важно:** `user_id` пока NULLABLE — старые записи без пользователя не сломаются. На День 7 можно сделать NOT NULL после миграции данных.

Обнови `RunMigrations()` в `internal/repository/postgres/expense_repo.go` чтобы выполнял все .up.sql файлы по порядку (или создай отдельный `migrations.go`).

#### Шаг 3: User entity (10 мин)

Создай `internal/domain/user.go`:
```go
package domain

import "time"

type User struct {
    ID           int       `json:"id"`
    Email        string    `json:"email"`
    PasswordHash string    `json:"-"`          // "-" = не попадёт в JSON ответ (как @JsonIgnore в Java)
    Name         string    `json:"name"`
    CreatedAt    time.Time `json:"created_at"`
}
```

Добавь ошибки в `internal/domain/expense.go` (или отдельный errors.go):
```go
var (
    ErrEmailExists   = errors.New("email already exists")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrInvalidCreds  = errors.New("invalid email or password")
)
```

#### Шаг 4: UserRepository (20 мин)

Создай `internal/repository/user_repository.go`:
```go
type UserRepository interface {
    Create(ctx context.Context, user *domain.User) error
    GetByEmail(ctx context.Context, email string) (*domain.User, error)
    GetByID(ctx context.Context, id int) (*domain.User, error)
}
```

Создай `internal/repository/postgres/user_repo.go`:
- `Create` — INSERT с RETURNING id. Обрабатывай UNIQUE violation → `domain.ErrEmailExists`
- `GetByEmail` — SELECT WHERE email = $1. `sql.ErrNoRows` → `domain.ErrNotFound`
- `GetByID` — SELECT WHERE id = $1

**Аналогия C:** `UserRepository` интерфейс — это struct с function pointers. `user_repo.go` — конкретная реализация этих function pointers для PostgreSQL.

#### Шаг 5: AuthService (45 мин)

Создай `internal/service/auth_service.go`:
```go
type AuthService struct {
    userRepo  repository.UserRepository
    jwtSecret []byte
}

func NewAuthService(userRepo repository.UserRepository, jwtSecret string) *AuthService
```

Методы:
- `Register(ctx, email, password, name) (*domain.User, error)`:
  1. Валидация: email не пуст, password >= 6 символов
  2. `bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)` — хеширование
  3. `userRepo.Create(ctx, &user)` — сохранение
  4. Аналогия: bcrypt = `crypt()` из POSIX, но с автоматическим salt

- `Login(ctx, email, password) (string, error)`:
  1. `userRepo.GetByEmail(ctx, email)` — найти пользователя
  2. `bcrypt.CompareHashAndPassword(user.PasswordHash, password)` — сравнение
  3. Если ок — генерируем JWT token:
     ```go
     claims := jwt.MapClaims{
         "sub": user.ID,
         "exp": time.Now().Add(24 * time.Hour).Unix(),
         "iat": time.Now().Unix(),
     }
     token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
     return token.SignedString(s.jwtSecret)
     ```
  4. Аналогия: JWT = подписанный JSON. Как HMAC в C: `HMAC_SHA256(secret, header+payload)`. Сервер может проверить подпись без базы данных.

- `VerifyToken(tokenString) (int, error)`:
  1. `jwt.Parse(tokenString, keyFunc)` — парсит и проверяет подпись
  2. Извлекает `sub` claim → возвращает userID
  3. Если токен expired или подпись невалидна → `domain.ErrUnauthorized`

#### Шаг 6: Auth middleware (30 мин)

Создай `internal/middleware/auth.go`:
```go
type contextKey string
const UserIDKey contextKey = "userID"

func RequireAuth(authService *service.AuthService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            cookie, err := r.Cookie("token")
            if err != nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }
            userID, err := authService.VerifyToken(cookie.Value)
            if err != nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }
            ctx := context.WithValue(r.Context(), UserIDKey, userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func GetUserID(ctx context.Context) int {
    return ctx.Value(UserIDKey).(int)
}
```

#### Шаг 7: Auth handler (30 мин)

Создай `internal/handler/api_auth.go`:
- `HandleRegister` — POST /api/v1/auth/register: парсит JSON, вызывает Register, возвращает 201
- `HandleLogin` — POST /api/v1/auth/login: парсит JSON, получает JWT, ставит HttpOnly cookie
- `HandleLogout` — POST /api/v1/auth/logout: удаляет cookie (MaxAge: -1)

#### Шаг 8: Обнови expense CRUD — привязка к user_id (30 мин)

1. Добавь `UserID int` в `domain.Expense`
2. Обнови SQL: WHERE user_id = $X во всех запросах
3. Обнови интерфейс `ExpenseRepository` — добавь `userID int`
4. Обнови `ExpenseService` — прокидывай userID из ctx
5. Обнови хендлеры — `middleware.GetUserID(r.Context())`

#### Шаг 9: Wiring в main.go (20 мин)

1. Добавь `JWT_SECRET` в config.go
2. Создай `userRepo` и `authService`
3. Auth роуты БЕЗ middleware
4. Expense роуты ЧЕРЕЗ middleware

#### Шаг 10: Обнови CLI (30 мин)

1. Добавь пункты "Регистрация", "Войти"
2. Используй `http.Client` с `http.CookieJar`:
```go
jar, _ := cookiejar.New(nil)
client := &http.Client{Jar: jar}
```

#### Проверка:
```bash
curl -v -X POST localhost:8080/api/v1/auth/register -d '{"email":"test@test.com","password":"123456","name":"Test"}'
curl -v -c cookies.txt -X POST localhost:8080/api/v1/auth/login -d '{"email":"test@test.com","password":"123456"}'
curl -b cookies.txt -X POST localhost:8080/api/v1/expenses -d '{"category":"Coffee","amount":500,"comment":"latte"}'
curl -b cookies.txt localhost:8080/api/v1/expenses
curl localhost:8080/api/v1/expenses  # → 401
```

---

### День 4: OAuth2 — Google + GitHub (4-5 ч)

**Цель:** Вход через Google и GitHub. Привязка по email — если пользователь зарегистрирован через email+password, а потом логинится через Google с тем же email, он попадает в тот же аккаунт.

**Что изучить:** OAuth2 Authorization Code Flow (redirect → consent → callback с code → обмен code на access_token → запрос userinfo).

**Зависимость:** `go get golang.org/x/oauth2`

#### Шаг 1: Миграция — OAuth поля (10 мин)

Создай `migrations/000004_add_oauth_fields.up.sql`:
```sql
ALTER TABLE users
    ADD COLUMN oauth_provider VARCHAR(20) DEFAULT '',
    ADD COLUMN oauth_id VARCHAR(255) DEFAULT '',
    ALTER COLUMN password_hash DROP NOT NULL;
CREATE UNIQUE INDEX idx_users_oauth ON users(oauth_provider, oauth_id) WHERE oauth_provider != '';
```

#### Шаг 2: Config — OAuth credentials (15 мин)

Добавь `GoogleClientID`, `GoogleClientSecret`, `GoogleRedirectURL`, аналогично для GitHub в `config.go`.

#### Шаг 3: Обнови UserRepository (15 мин)

Добавь: `GetByOAuth`, `CreateOAuth`, `LinkOAuth`

#### Шаг 4: OAuthService (45 мин)

Создай `internal/service/oauth_service.go`:
- `GetGoogleAuthURL(state)`, `GetGitHubAuthURL(state)`
- `HandleGoogleCallback(ctx, code)` — обмен code → token → userinfo → find/create user → JWT
- `HandleGitHubCallback(ctx, code)` — аналогично

#### Шаг 5: OAuth handler (30 мин)

Создай `internal/handler/oauth_handler.go`:
- GET /auth/google → redirect на Google consent
- GET /auth/google/callback → обмен code, ставит cookie, redirect на /dashboard
- Аналогично для GitHub

#### Шаг 6: Wiring + тест (30 мин)

**Проверка:** браузер → `http://localhost:8080/auth/google` → Google consent → callback → cookie → redirect

---

### День 5: Frontend — html/template + HTMX + TailwindCSS (5-6 ч)

**Цель:** Полноценный UI в браузере. Login/register, список расходов с добавлением/удалением без перезагрузки.

**Что изучить:** `html/template` (как Jinja2 — `{{.FieldName}}`, `{{range}}`), HTMX (`hx-get`, `hx-post`, `hx-swap` заменяют AJAX), TailwindCSS (utility CSS через CDN).

#### Шаг 1: Структура templates (10 мин)

```
templates/
├── layouts/base.html
├── pages/{login,register,expenses}.html
└── partials/{expense_row,expense_list,expense_form}.html
```

#### Шаг 2: Base layout (20 мин)

`templates/layouts/base.html` — `<html>`, Tailwind CDN, HTMX script, navbar, `{{template "content" .}}`

#### Шаг 3: Login + Register (30 мин)

`hx-post="/auth/login"` — HTMX делает POST, ответ вставляется в DOM. При успехе — `HX-Redirect: /expenses`.

#### Шаг 4: Expenses страница (45 мин)

Форма добавления (`hx-post="/expenses" hx-target="#expense-list" hx-swap="afterbegin"`) + таблица с `expense_row.html` partial. Удаление: `hx-delete="/expenses/{{.ID}}" hx-target="#expense-{{.ID}}" hx-swap="outerHTML"`.

#### Шаг 5: WebPageHandler (30 мин)

`internal/handler/web_page.go` — HandleIndex, HandleLogin, HandleRegister, HandleExpenses. Загрузка шаблонов через `template.ParseGlob`.

#### Шаг 6: WebExpenseHandler (30 мин)

`internal/handler/web_expense.go` — HandleCreate (form data → partial HTML), HandleDelete (пустой ответ), HandleWebLogin/Logout.

**API vs Web:** API = JSON (CLI, bot). Web = form data → HTML (браузер + HTMX). Оба вызывают одни сервисы.

#### Шаг 7: Wiring + static files (20 мин)

`http.FileServer` для static/, регистрация web-роутов в main.go.

**Проверка:** браузер → login → добавить/удалить расход без перезагрузки

---

### День 6: Dashboard + Графики + Seed Script (4-5 ч)

**Цель:** Дашборд с Chart.js (расходы по месяцам, по категориям). Seed-скрипт с 1200 демо-транзакциями.

#### Шаг 1: Stats в repository и service (45 мин)

`GetMonthlyStats(ctx, userID)`, `GetCategoryStats(ctx, userID)` — SQL с GROUP BY.

#### Шаг 2: Stats API endpoints (20 мин)

GET /api/v1/stats/monthly, GET /api/v1/stats/by-category → JSON.

#### Шаг 3: Dashboard страница (45 мин)

`templates/pages/dashboard.html` — карточки (итого за месяц, среднее в день, топ категория) + `<canvas>` для Chart.js. Графики через `fetch('/api/v1/stats/...')` → Chart.js (единственное место с JS).

#### Шаг 4: Seed script (45 мин)

`scripts/seed.go` — создаёт test@demo.com / 123456, генерирует 1200 расходов за 12 месяцев по 8 категориям.

#### Шаг 5: Wiring (15 мин)

GET /dashboard + API stats роуты (с auth middleware).

**Проверка:** `go run scripts/seed.go` → login → dashboard с графиками

---

### День 7: Тесты, Graceful Shutdown, Rate Limiting, Polish (5-6 ч)

**Цель:** Production-ready. Тесты. Graceful shutdown. Логирование. Rate limiting.

**Зависимость:** `go get golang.org/x/time`

#### Шаг 1: Unit-тесты service (60 мин)

`internal/service/expense_service_test.go` — mock repo + table-driven тесты: Create (валидация), GetAll, GetByID (found + not found), GetTotal, Delete.

#### Шаг 2: Integration-тесты handler (45 мин)

`internal/handler/api_expense_test.go` — `httptest.NewServer` + mock repo. POST 201, GET 200/404, POST 400, DELETE 200/404.

#### Шаг 3: Graceful shutdown (30 мин)

`signal.Notify(quit, SIGINT, SIGTERM)` → `srv.Shutdown(ctx)` с 5 сек таймаутом.

#### Шаг 4: Logging middleware (30 мин)

`internal/middleware/logging.go` — `log/slog`, method/path/status/duration.

#### Шаг 5: Rate limiting (30 мин)

`internal/middleware/ratelimit.go` — `golang.org/x/time/rate`, token bucket per IP. 10 req/sec, burst 20.

#### Шаг 6: Docker финализация (30 мин)

Обнови Dockerfile (COPY templates, static), docker-compose (healthcheck, restart, env_file), `.env.example`.

**Проверка:** `go test ./...` + `docker-compose up --build` + graceful shutdown + rate limiting

---

### День 8: Telegram Bot — `gopkg.in/telebot.v3` (5-6 ч)

**Цель:** Telegram бот с теми же функциями что на сайте (кроме аналитических графиков). Привязка аккаунта через короткий код (безопасно — пароль не вводится в Telegram).

**Что изучить:** Telegram Bot API (long polling), state machine для step-by-step input, `gopkg.in/telebot.v3`.

**Зависимость:** `go get gopkg.in/telebot.v3`

**Где взять токен:** @BotFather → /newbot

#### Шаг 1: Миграция — telegram_chat_id (15 мин)

`migrations/000005_add_telegram_chat_id.up.sql`:
```sql
ALTER TABLE users ADD COLUMN telegram_chat_id BIGINT UNIQUE;
```
Обнови `domain.User`: `TelegramChatID *int64`

#### Шаг 2: Config (10 мин)

`TelegramBotToken` в `config.go` из env `TELEGRAM_BOT_TOKEN`.

#### Шаг 3: Механизм привязки — Link Code (30 мин)

**Flow:** /start → бот генерирует 6-значный код → пользователь вводит код на Dashboard → аккаунты связаны.

`internal/service/link_service.go`:
- `GenerateCode(chatID int64) string` — 6-значный код, хранится in-memory, TTL 10 мин
- `ConfirmLink(ctx, userID, code) error` — находит pending code, привязывает telegram_chat_id к user

Добавь в `UserRepository`: `GetByTelegramChatID(ctx, chatID)`, `LinkTelegram(ctx, userID, chatID)`.

#### Шаг 4: Web endpoint для привязки (20 мин)

POST /telegram/link (с auth middleware) — форма на Dashboard, парсит code, вызывает `linkService.ConfirmLink`.

#### Шаг 5: Bot handler — Clean Architecture (45 мин)

Создай `internal/handler/telegram/bot.go`:
```go
type BotHandler struct {
    bot            *tele.Bot
    expenseService *service.ExpenseService
    userService    *service.UserService
    linkService    *service.LinkService
    sessions       map[int64]*UserSession
    mu             sync.RWMutex
}
```

Бот — это handler-слой (как HTTP handlers). Принимает сервисы через DI. **Не лезет в БД напрямую.**

Команды через `b.Handle("/command", handler)` — telebot.v3 API.

#### Шаг 6: Команды и auth check (45 мин)

`internal/handler/telegram/handlers.go`:
- `/start` → проверяет привязку. Если нет — генерирует код. Если да — "Аккаунт привязан!"
- `/help` → список команд
- `/add` → запускает state machine (категория → сумма → комментарий)
- `/list` → показывает последние 20 расходов
- `/total` → сумма
- `/delete` → запрашивает ID

Auth: `userService.GetByTelegramChatID(chatID)` → если nil, "Привяжите аккаунт через /start".

#### Шаг 7: State machine (30 мин)

`handleText` — fallback для всех текстовых сообщений. Если session.State != "" — обрабатывает как часть flow (awaiting_category → awaiting_amount → awaiting_comment). `sync.RWMutex` для thread safety.

#### Шаг 8: UserService (10 мин)

`internal/service/user_service.go`: `GetByTelegramChatID(ctx, chatID)` — прокидка в repo.

#### Шаг 9: Wiring в main.go (20 мин)

```go
if cfg.TelegramBotToken != "" {
    botHandler, _ := telegram.NewBotHandler(token, expenseService, userService, linkService)
    go botHandler.Start()
    defer botHandler.Stop()
}
```

**Архитектура:**
```
cmd/server/main.go
├── HTTP handlers (goroutine 1) → services → postgres
└── Telegram handler (goroutine 2) → services → postgres
```

#### Шаг 10: Тест (15 мин)

/start → код → вводим на сайте → /add → /list → /total → /delete

**Проверка:** расходы из бота видны на сайте и наоборот (общая БД!)

---

## Внешние зависимости (все pure Go, без CGO)

| Пакет | Зачем | День |
|-------|-------|------|
| `github.com/lib/pq` | PostgreSQL драйвер | 2 |
| `golang.org/x/crypto` | bcrypt хеширование | 3 |
| `github.com/golang-jwt/jwt/v5` | JWT токены | 3 |
| `golang.org/x/oauth2` | OAuth2 клиент | 4 |
| `golang.org/x/time` | Rate limiter | 7 |
| `gopkg.in/telebot.v3` | Telegram Bot API | 8 |

---

## Финальная проверка (после Дня 8)

1. `go test ./...` — все тесты зелёные
2. `docker-compose up --build` стартует чисто
3. `go run scripts/seed.go` — 1200 расходов для demo
4. Браузер: login test@demo.com / 123456 → expenses list → dashboard с графиками
5. CLI: login → add/list/total/update/delete → всё работает
6. Telegram: /start → код → привязка на сайте → /add → /list → /total → /delete
7. OAuth Google/GitHub → вход через браузер
8. Rate limiting: спам curl → 429
9. `Ctrl+C` → "Server stopped gracefully" + бот остановлен
10. `docker-compose up` → данные на месте (PostgreSQL volume)
11. Все три клиента (браузер, CLI, Telegram) видят одни и те же данные
