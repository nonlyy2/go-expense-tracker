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

### День 6: Dashboard + Chart.js + Seed Script (4-5 ч)

**Цель:** Дашборд с графиками (расходы по месяцам, по категориям). Seed-скрипт для демо-данных. Навигация между /expenses и /dashboard.

#### Шаг 1: Новые domain-типы для статистики (10 мин)

Создай `internal/domain/stats.go`:
```go
package domain

type MonthlyStat struct {
    Month string  `json:"month"` // "2025-01"
    Total float64 `json:"total"`
}

type CategoryStat struct {
    Category string  `json:"category"`
    Total    float64 `json:"total"`
    Count    int     `json:"count"`
}
```

#### Шаг 2: Расширь ExpenseRepository — stats-методы (20 мин)

Добавь в `internal/repository/expense_repository.go`:
```go
GetMonthlyStats(ctx context.Context, userID int) ([]domain.MonthlyStat, error)
GetCategoryStats(ctx context.Context, userID int) ([]domain.CategoryStat, error)
```

Реализация в `internal/repository/postgres/expense_repo.go`:
```go
func (r *expenseRepo) GetMonthlyStats(ctx context.Context, userID int) ([]domain.MonthlyStat, error) {
    query := `SELECT TO_CHAR(date, 'YYYY-MM') AS month, SUM(amount) AS total
              FROM expenses WHERE user_id = $1
              GROUP BY month ORDER BY month`
    rows, err := r.db.QueryContext(ctx, query, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var stats []domain.MonthlyStat
    for rows.Next() {
        var s domain.MonthlyStat
        if err := rows.Scan(&s.Month, &s.Total); err != nil {
            return nil, err
        }
        stats = append(stats, s)
    }
    return stats, rows.Err()
}

func (r *expenseRepo) GetCategoryStats(ctx context.Context, userID int) ([]domain.CategoryStat, error) {
    query := `SELECT category, SUM(amount) AS total, COUNT(*) AS count
              FROM expenses WHERE user_id = $1
              GROUP BY category ORDER BY total DESC`
    rows, err := r.db.QueryContext(ctx, query, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var stats []domain.CategoryStat
    for rows.Next() {
        var s domain.CategoryStat
        if err := rows.Scan(&s.Category, &s.Total, &s.Count); err != nil {
            return nil, err
        }
        stats = append(stats, s)
    }
    return stats, rows.Err()
}
```

**Ловушка:** не забудь добавить оба метода и в интерфейс `ExpenseRepository`, и в реализацию. Иначе — ошибка компиляции `*expenseRepo does not implement repository.ExpenseRepository`.

#### Шаг 3: Расширь ExpenseService — stats-методы (10 мин)

Добавь в `internal/service/expense_service.go`:
```go
func (s *ExpenseService) GetMonthlyStats(ctx context.Context, userID int) ([]domain.MonthlyStat, error) {
    return s.repo.GetMonthlyStats(ctx, userID)
}

func (s *ExpenseService) GetCategoryStats(ctx context.Context, userID int) ([]domain.CategoryStat, error) {
    return s.repo.GetCategoryStats(ctx, userID)
}
```

#### Шаг 4: StatsHandler — API endpoints (20 мин)

Создай `internal/handler/api_stats.go`:
```go
package handler

import (
    "net/http"

    "go-expense-tracker/internal/middleware"
    "go-expense-tracker/internal/service"
)

type StatsHandler struct {
    service *service.ExpenseService
}

func NewStatsHandler(s *service.ExpenseService) *StatsHandler {
    return &StatsHandler{service: s}
}

func (h *StatsHandler) HandleMonthlyStats(w http.ResponseWriter, r *http.Request) {
    userID := middleware.GetUserID(r.Context())
    if userID == 0 {
        writeError(w, http.StatusUnauthorized, "unauthorized")
        return
    }

    stats, err := h.service.GetMonthlyStats(r.Context(), userID)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    writeJSON(w, http.StatusOK, stats)
}

func (h *StatsHandler) HandleCategoryStats(w http.ResponseWriter, r *http.Request) {
    userID := middleware.GetUserID(r.Context())
    if userID == 0 {
        writeError(w, http.StatusUnauthorized, "unauthorized")
        return
    }

    stats, err := h.service.GetCategoryStats(r.Context(), userID)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    writeJSON(w, http.StatusOK, stats)
}

func (h *StatsHandler) RegisterRoutes(mux *http.ServeMux, requireAuth func(http.HandlerFunc) http.HandlerFunc) {
    mux.HandleFunc("GET /api/v1/stats/monthly", requireAuth(h.HandleMonthlyStats))
    mux.HandleFunc("GET /api/v1/stats/by-category", requireAuth(h.HandleCategoryStats))
}
```

**Заметка:** `writeJSON` и `writeError` уже определены в `internal/handler/api_expense.go`. Они в том же пакете `handler`, поэтому доступны без импорта.

#### Шаг 5: Dashboard template (45 мин)

Создай `templates/pages/dashboard.html`:
```html
{{define "content"}}
<div class="space-y-8">
    <h1 class="text-2xl font-bold">Dashboard</h1>

    <!-- summary cards -->
    <div id="summary-cards" class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div class="bg-white p-6 rounded-lg shadow-md">
            <p class="text-sm text-gray-500">Total this month</p>
            <p class="text-3xl font-bold text-indigo-600" id="month-total">...</p>
        </div>
        <div class="bg-white p-6 rounded-lg shadow-md">
            <p class="text-sm text-gray-500">Average per day</p>
            <p class="text-3xl font-bold text-indigo-600" id="avg-daily">...</p>
        </div>
        <div class="bg-white p-6 rounded-lg shadow-md">
            <p class="text-sm text-gray-500">Top category</p>
            <p class="text-3xl font-bold text-indigo-600" id="top-category">...</p>
        </div>
    </div>

    <!-- charts -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div class="bg-white p-6 rounded-lg shadow-md">
            <h2 class="text-lg font-bold mb-4">Monthly expenses</h2>
            <canvas id="monthlyChart"></canvas>
        </div>
        <div class="bg-white p-6 rounded-lg shadow-md">
            <h2 class="text-lg font-bold mb-4">By category</h2>
            <canvas id="categoryChart"></canvas>
        </div>
    </div>
</div>

<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
<script>
async function loadDashboard() {
    try {
        const [monthlyRes, categoryRes] = await Promise.all([
            fetch('/api/v1/stats/monthly', { credentials: 'same-origin' }),
            fetch('/api/v1/stats/by-category', { credentials: 'same-origin' })
        ]);

        if (!monthlyRes.ok || !categoryRes.ok) {
            console.error('failed to load stats');
            return;
        }

        const monthly = await monthlyRes.json();
        const categories = await categoryRes.json();

        // summary cards
        if (monthly && monthly.length > 0) {
            const lastMonth = monthly[monthly.length - 1];
            document.getElementById('month-total').textContent =
                parseFloat(lastMonth.total).toFixed(0) + ' ₸';

            const daysInMonth = new Date(
                parseInt(lastMonth.month.split('-')[0]),
                parseInt(lastMonth.month.split('-')[1]),
                0
            ).getDate();
            document.getElementById('avg-daily').textContent =
                (lastMonth.total / daysInMonth).toFixed(0) + ' ₸';
        }

        if (categories && categories.length > 0) {
            document.getElementById('top-category').textContent = categories[0].category;
        }

        // monthly bar chart
        if (monthly && monthly.length > 0) {
            new Chart(document.getElementById('monthlyChart'), {
                type: 'bar',
                data: {
                    labels: monthly.map(m => m.month),
                    datasets: [{
                        label: 'Expenses',
                        data: monthly.map(m => m.total),
                        backgroundColor: 'rgba(79, 70, 229, 0.6)',
                        borderColor: 'rgba(79, 70, 229, 1)',
                        borderWidth: 1
                    }]
                },
                options: {
                    responsive: true,
                    scales: { y: { beginAtZero: true } }
                }
            });
        }

        // category doughnut chart
        if (categories && categories.length > 0) {
            const colors = [
                '#6366f1','#ec4899','#f59e0b','#10b981',
                '#3b82f6','#8b5cf6','#ef4444','#14b8a6'
            ];
            new Chart(document.getElementById('categoryChart'), {
                type: 'doughnut',
                data: {
                    labels: categories.map(c => c.category),
                    datasets: [{
                        data: categories.map(c => c.total),
                        backgroundColor: colors.slice(0, categories.length)
                    }]
                },
                options: { responsive: true }
            });
        }

    } catch (err) {
        console.error('dashboard error:', err);
    }
}
loadDashboard();
</script>
{{end}}
```

**Ловушка (Chart.js):** `<script src="https://cdn.jsdelivr.net/npm/chart.js">` — CDN, как Tailwind. Если забудешь — canvas будет пустым без ошибок в консоли (Chart не определён).

#### Шаг 6: Зарегистрируй dashboard.html в WebPageHandler (15 мин)

В `internal/handler/web_page.go`:

1. Добавь `"dashboard.html"` в массив `pages`:
```go
pages := []string{"login.html", "register.html", "expenses.html", "dashboard.html"}
```

2. Добавь новый handler метод:
```go
func (h *WebPageHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
    userID := middleware.GetUserID(r.Context())
    if userID == 0 {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }

    user, err := h.authService.GetUserByID(r.Context(), userID)
    if err != nil {
        http.SetCookie(w, &http.Cookie{Name: "token", MaxAge: -1, Path: "/"})
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }

    h.render(w, "dashboard.html", PageData{
        User: map[string]interface{}{"Name": user.Name},
    })
}
```

#### Шаг 7: Обнови навигацию в base.html (10 мин)

В `templates/layouts/base.html`, внутри блока `{{if .User}}` добавь ссылку на Dashboard:
```html
<a href="/dashboard" class="hover:text-indigo-200 font-medium">Dashboard</a>
<a href="/expenses" class="hover:text-indigo-200 font-medium">Expenses</a>
```

Также обнови `HandleIndex` в `web_page.go` — redirect на `/dashboard` вместо `/expenses`:
```go
func (h *WebPageHandler) HandleIndex(w http.ResponseWriter, r *http.Request) {
    if _, err := r.Cookie("token"); err != nil {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }
    http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
```

#### Шаг 8: Wiring в main.go (10 мин)

В `cmd/server/main.go`:

1. Создай StatsHandler:
```go
statsHandler := handler.NewStatsHandler(svc)
```

2. Зарегистрируй роуты:
```go
// dashboard page
mux.HandleFunc("GET /dashboard", authMiddleware(webPageHandler.HandleDashboard))

// stats API
statsHandler.RegisterRoutes(mux, authMiddleware)
```

#### Шаг 9: Seed script (30 мин)

Создай `cmd/seed/main.go`:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "math/rand"
    "os"
    "time"

    "go-expense-tracker/internal/domain"
    postgresrepo "go-expense-tracker/internal/repository/postgres"
    "go-expense-tracker/internal/service"
)

func main() {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        dsn = "postgres://postgres:qwerty@localhost:5433/expense_tracker?sslmode=disable"
    }

    repo, err := postgresrepo.NewExpenseRepo(dsn)
    if err != nil {
        log.Fatalf("db connection failed: %v", err)
    }
    db := repo.DB()
    if err := postgresrepo.RunMigrations(db, "migrations"); err != nil {
        log.Fatalf("migrations failed: %v", err)
    }

    userRepo := postgresrepo.NewUserRepo(db)
    authSvc := service.NewAuthService(userRepo, "super-secret-dev-key")
    expSvc := service.NewExpenseService(repo)

    ctx := context.Background()

    // create demo user (skip if exists)
    email := "demo@demo.com"
    password := "Demo1234!"
    _, err = authSvc.Register(ctx, email, password, "Demo User")
    if err != nil {
        fmt.Printf("user may already exist: %v\n", err)
    }

    // login to get userID
    token, err := authSvc.Login(ctx, email, password)
    if err != nil {
        log.Fatalf("login failed: %v", err)
    }
    userID, err := authSvc.VerifyToken(token)
    if err != nil {
        log.Fatalf("verify failed: %v", err)
    }

    categories := []string{"Food", "Transport", "Entertainment", "Shopping", "Health", "Education", "Utilities", "Other"}
    comments := map[string][]string{
        "Food":          {"lunch", "groceries", "coffee", "dinner", "snacks"},
        "Transport":     {"taxi", "bus", "fuel", "metro", "parking"},
        "Entertainment": {"cinema", "concert", "games", "books", "streaming"},
        "Shopping":      {"clothes", "electronics", "gifts", "home", "tools"},
        "Health":        {"pharmacy", "gym", "doctor", "vitamins", "insurance"},
        "Education":     {"course", "books", "tutoring", "software", "exam"},
        "Utilities":     {"electricity", "water", "internet", "phone", "rent"},
        "Other":         {"misc", "donation", "fee", "repair", "subscription"},
    }

    now := time.Now()
    count := 0

    for m := 11; m >= 0; m-- {
        monthStart := time.Date(now.Year(), now.Month()-time.Month(m), 1, 0, 0, 0, 0, time.Local)
        daysInMonth := time.Date(monthStart.Year(), monthStart.Month()+1, 0, 0, 0, 0, 0, time.Local).Day()
        numExpenses := 80 + rand.Intn(40) // 80-120 per month

        for i := 0; i < numExpenses; i++ {
            cat := categories[rand.Intn(len(categories))]
            cmts := comments[cat]
            comment := cmts[rand.Intn(len(cmts))]

            amount := 200 + rand.Float64()*9800 // 200-10000
            day := 1 + rand.Intn(daysInMonth)
            hour := 8 + rand.Intn(14) // 8:00-22:00

            date := time.Date(monthStart.Year(), monthStart.Month(), day, hour, rand.Intn(60), 0, 0, time.Local)

            exp := &domain.Expense{
                UserID:   userID,
                Date:     date,
                Amount:   float64(int(amount*100)) / 100, // round to 2 decimal places
                Category: cat,
                Comment:  comment,
            }

            if err := repo.Create(ctx, exp); err != nil {
                log.Printf("failed to create expense: %v", err)
                continue
            }
            count++
        }
    }

    fmt.Printf("created %d expenses for %s\n", count, email)
    fmt.Printf("login: %s / %s\n", email, password)
}
```

**Ловушка:** не забудь добавить `"go-expense-tracker/internal/domain"` в импорты (используется `domain.Expense`).

**Ловушка:** пароль `Demo1234!` — пройдёт валидацию (8+ символов, upper, lower, digit, special). Если выберешь простой пароль типа `123456` — Register вернёт ошибку.

#### Шаг 10: Проверка

```bash
# 1. Запусти сервер
go run ./cmd/server

# 2. В другом терминале — seed
go run ./cmd/seed

# 3. В браузере
# login: demo@demo.com / Demo1234!
# /dashboard — должны быть 2 графика + 3 карточки с числами
# /expenses — список расходов (должно быть ~1000+)

# 4. API проверка
curl -c cookies.txt -X POST localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@demo.com","password":"Demo1234!"}'
curl -b cookies.txt localhost:8080/api/v1/stats/monthly    # JSON array
curl -b cookies.txt localhost:8080/api/v1/stats/by-category # JSON array
```

---

### День 7: Тесты + Graceful Shutdown + Logging + Rate Limiting + Docker (5-6 ч)

**Цель:** Production-ready. Unit/integration тесты. Graceful shutdown. Structured logging. Rate limiting. Docker финализация.

**Зависимость:** `go get golang.org/x/time`

#### Шаг 1: Mock repository для тестов (30 мин)

Создай `internal/repository/mock/expense_repo.go`:
```go
package mock

import (
    "context"
    "sync"

    "go-expense-tracker/internal/domain"
)

type ExpenseRepo struct {
    mu       sync.Mutex
    expenses []domain.Expense
    nextID   int
}

func NewExpenseRepo() *ExpenseRepo {
    return &ExpenseRepo{nextID: 1}
}

func (r *ExpenseRepo) Create(ctx context.Context, expense *domain.Expense) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    expense.ID = r.nextID
    r.nextID++
    r.expenses = append(r.expenses, *expense)
    return nil
}

func (r *ExpenseRepo) GetAll(ctx context.Context, userID int) ([]domain.Expense, error) {
    r.mu.Lock()
    defer r.mu.Unlock()
    var result []domain.Expense
    for _, e := range r.expenses {
        if e.UserID == userID {
            result = append(result, e)
        }
    }
    return result, nil
}

func (r *ExpenseRepo) GetByID(ctx context.Context, id int, userID int) (*domain.Expense, error) {
    r.mu.Lock()
    defer r.mu.Unlock()
    for _, e := range r.expenses {
        if e.ID == id && e.UserID == userID {
            return &e, nil
        }
    }
    return nil, domain.ErrNotFound
}

func (r *ExpenseRepo) Update(ctx context.Context, expense *domain.Expense) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    for i, e := range r.expenses {
        if e.ID == expense.ID && e.UserID == expense.UserID {
            r.expenses[i] = *expense
            return nil
        }
    }
    return domain.ErrNotFound
}

func (r *ExpenseRepo) Delete(ctx context.Context, id int, userID int) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    for i, e := range r.expenses {
        if e.ID == id && e.UserID == userID {
            r.expenses = append(r.expenses[:i], r.expenses[i+1:]...)
            return nil
        }
    }
    return domain.ErrNotFound
}

func (r *ExpenseRepo) GetMonthlyStats(ctx context.Context, userID int) ([]domain.MonthlyStat, error) {
    return nil, nil
}

func (r *ExpenseRepo) GetCategoryStats(ctx context.Context, userID int) ([]domain.CategoryStat, error) {
    return nil, nil
}
```

**Ловушка:** mock ОБЯЗАН реализовать ВСЕ методы интерфейса `ExpenseRepository`, включая `GetMonthlyStats` и `GetCategoryStats` (добавленные на День 6). Если забудешь — компиляция тестов упадёт.

**Заметка:** mock stats-методы возвращают `nil, nil` — для текущих тестов (создание/удаление/получение) этого достаточно. Если в будущем захочешь тестировать dashboard/графики — добавь туда фейковую логику с реальными данными.

#### Шаг 2: Unit-тесты service (45 мин)

Создай `internal/service/expense_service_test.go`:
```go
package service

import (
    "context"
    "testing"

    "go-expense-tracker/internal/domain"
    "go-expense-tracker/internal/repository/mock"
)

func TestCreateExpense(t *testing.T) {
    repo := mock.NewExpenseRepo()
    svc := NewExpenseService(repo)
    ctx := context.Background()

    tests := []struct {
        name     string
        category string
        amount   float64
        wantErr  error
    }{
        {"valid expense", "Food", 500, nil},
        {"zero amount", "Food", 0, domain.ErrInvalidAmount},
        {"negative amount", "Food", -100, domain.ErrInvalidAmount},
        {"empty category", "", 500, domain.ErrEmptyCategory},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := svc.CreateExpense(ctx, tt.category, tt.amount, "test", 1)
            if err != tt.wantErr {
                t.Errorf("got %v, want %v", err, tt.wantErr)
            }
        })
    }
}

func TestGetAllExpenses(t *testing.T) {
    repo := mock.NewExpenseRepo()
    svc := NewExpenseService(repo)
    ctx := context.Background()

    // user 1 creates 2 expenses
    svc.CreateExpense(ctx, "Food", 500, "lunch", 1)
    svc.CreateExpense(ctx, "Transport", 200, "taxi", 1)

    // user 2 creates 1
    svc.CreateExpense(ctx, "Food", 300, "dinner", 2)

    expenses, err := svc.GetAllExpenses(ctx, 1)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(expenses) != 2 {
        t.Errorf("got %d expenses, want 2", len(expenses))
    }
}

func TestDeleteExpense(t *testing.T) {
    repo := mock.NewExpenseRepo()
    svc := NewExpenseService(repo)
    ctx := context.Background()

    exp, _ := svc.CreateExpense(ctx, "Food", 500, "test", 1)

    // delete existing
    if err := svc.DeleteExpense(ctx, exp.ID, 1); err != nil {
        t.Errorf("unexpected error: %v", err)
    }

    // delete non-existing
    if err := svc.DeleteExpense(ctx, 999, 1); err != domain.ErrNotFound {
        t.Errorf("got %v, want ErrNotFound", err)
    }
}

func TestGetTotal(t *testing.T) {
    repo := mock.NewExpenseRepo()
    svc := NewExpenseService(repo)
    ctx := context.Background()

    svc.CreateExpense(ctx, "Food", 500, "", 1)
    svc.CreateExpense(ctx, "Transport", 200, "", 1)

    total, err := svc.GetTotal(ctx, 1)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if total != 700 {
        t.Errorf("got %.2f, want 700", total)
    }
}
```

#### Шаг 3: Integration-тесты handler (45 мин)

Создай `internal/handler/api_expense_test.go`:
```go
package handler

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "go-expense-tracker/internal/middleware"
    "go-expense-tracker/internal/repository/mock"
    "go-expense-tracker/internal/service"
)

// helper: inject userID into request context
func withUserID(r *http.Request, userID int) *http.Request {
    ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
    return r.WithContext(ctx)
}

func setupTestHandler() *ExpenseHandler {
    repo := mock.NewExpenseRepo()
    svc := service.NewExpenseService(repo)
    return NewExpenseHandler(svc)
}

func TestCreateExpenseHandler(t *testing.T) {
    h := setupTestHandler()

    body := `{"category":"Food","amount":500,"comment":"lunch"}`
    req := httptest.NewRequest("POST", "/api/v1/expenses", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    req = withUserID(req, 1)

    rr := httptest.NewRecorder()
    h.CreateExpense(rr, req)

    if rr.Code != http.StatusCreated {
        t.Errorf("got status %d, want %d", rr.Code, http.StatusCreated)
    }
}

func TestCreateExpenseHandler_InvalidAmount(t *testing.T) {
    h := setupTestHandler()

    body := `{"category":"Food","amount":-100,"comment":"bad"}`
    req := httptest.NewRequest("POST", "/api/v1/expenses", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    req = withUserID(req, 1)

    rr := httptest.NewRecorder()
    h.CreateExpense(rr, req)

    if rr.Code != http.StatusBadRequest {
        t.Errorf("got status %d, want %d", rr.Code, http.StatusBadRequest)
    }
}

func TestGetAllExpenses_Empty(t *testing.T) {
    h := setupTestHandler()

    req := httptest.NewRequest("GET", "/api/v1/expenses", nil)
    req = withUserID(req, 1)

    rr := httptest.NewRecorder()
    h.GetAllExpenses(rr, req)

    if rr.Code != http.StatusOK {
        t.Errorf("got status %d, want %d", rr.Code, http.StatusOK)
    }

    var expenses []interface{}
    json.NewDecoder(rr.Body).Decode(&expenses)
    if expenses != nil && len(expenses) != 0 {
        t.Errorf("expected empty array, got %d items", len(expenses))
    }
}
```

**Ловушка:** `withUserID` нужен потому что в тестах нет middleware. Мы вручную ставим userID в context. Если забудешь — все handlers вернут 401 (unauthorized) потому что `middleware.GetUserID()` вернёт 0.

#### Шаг 4: Graceful shutdown в main.go (20 мин)

В `cmd/server/main.go` замени `http.ListenAndServe(port, mux)` на:
```go
import (
    "context"
    "os/signal"
    "syscall"
    "time"
)

// ... в main():

srv := &http.Server{
    Addr:    port,
    Handler: mux,
}

// start in goroutine
go func() {
    fmt.Printf("Server running on http://localhost%s\n", port)
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("server crashed: %v", err)
    }
}()

// wait for SIGINT or SIGTERM
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

fmt.Println("shutting down server...")
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
if err := srv.Shutdown(ctx); err != nil {
    log.Fatalf("server forced to shutdown: %v", err)
}
fmt.Println("server stopped gracefully")
```

**Ловушка:** `http.ErrServerClosed` — это НЕ ошибка, а нормальный результат `Shutdown()`. Если не проверишь `err != http.ErrServerClosed` — при Ctrl+C будет ложный `log.Fatalf`.

#### Шаг 5: Logging middleware с slog (25 мин)

Создай `internal/middleware/logging.go`:
```go
package middleware

import (
    "log/slog"
    "net/http"
    "time"
)

type responseWriter struct {
    http.ResponseWriter
    status int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.status = code
    rw.ResponseWriter.WriteHeader(code)
}

func Logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

        next.ServeHTTP(wrapped, r)

        slog.Info("request",
            "method", r.Method,
            "path", r.URL.Path,
            "status", wrapped.status,
            "duration", time.Since(start).String(),
            "ip", r.RemoteAddr,
        )
    })
}
```

В `cmd/server/main.go` оберни mux:
```go
loggedMux := middleware.Logging(mux)

srv := &http.Server{
    Addr:    port,
    Handler: loggedMux,
}
```

**Ловушка:** `middleware.Logging` принимает и возвращает `http.Handler` (не `http.HandlerFunc`). Переменная `loggedMux` будет `http.Handler`, и её можно передать в `srv.Handler`.

#### Шаг 6: Rate limiting middleware (30 мин)

Создай `internal/middleware/ratelimit.go`:
```go
package middleware

import (
    "net/http"
    "sync"
    "time"

    "golang.org/x/time/rate"
)

type RateLimiter struct {
    visitors map[string]*visitor
    mu       sync.Mutex
    limit    rate.Limit
    burst    int
}

type visitor struct {
    limiter  *rate.Limiter
    lastSeen time.Time
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
    rl := &RateLimiter{
        visitors: make(map[string]*visitor),
        limit:    rate.Limit(rps),
        burst:    burst,
    }

    // cleanup old entries every minute
    go func() {
        for {
            time.Sleep(time.Minute)
            rl.mu.Lock()
            for ip, v := range rl.visitors {
                if time.Since(v.lastSeen) > 3*time.Minute {
                    delete(rl.visitors, ip)
                }
            }
            rl.mu.Unlock()
        }
    }()

    return rl
}

func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    v, exists := rl.visitors[ip]
    if !exists {
        limiter := rate.NewLimiter(rl.limit, rl.burst)
        rl.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
        return limiter
    }
    v.lastSeen = time.Now()
    return v.limiter
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := r.RemoteAddr
        limiter := rl.getVisitor(ip)
        if !limiter.Allow() {
            http.Error(w, "too many requests", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

В `cmd/server/main.go`:
```go
rl := middleware.NewRateLimiter(10, 20) // 10 req/sec, burst 20
loggedMux := middleware.Logging(rl.Middleware(mux))
```

**Ловушка:** cleanup goroutine не останавливается при Shutdown — для pet-project это ок. В production использовали бы context для graceful stop.

#### Шаг 7: Docker финализация (20 мин)

Обнови `Dockerfile`:
```dockerfile
# Stage 1: Build
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server

# Stage 2: Run
FROM alpine:latest
COPY --from=builder /app/server /server
COPY --from=builder /app/migrations /migrations
COPY --from=builder /app/templates /templates
EXPOSE 8080
CMD ["/server"]
```

**Ловушка (критическая):** строка `COPY --from=builder /app/templates /templates` — БЕЗ неё сервер в Docker упадёт при попытке загрузить templates (panic: template не найден). Текущий Dockerfile НЕ копирует templates.

Обнови `docker-compose.yml`:
```yaml
services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: qwerty
      POSTGRES_DB: expense_tracker
    ports:
      - "5433:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-ONLY", "pg_isready", "-U", "postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  app:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      db:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://postgres:qwerty@db:5432/expense_tracker?sslmode=disable
      JWT_SECRET: super-secret-dev-key
    restart: unless-stopped

volumes:
  pgdata:
```

**Ловушка:** `depends_on` без `condition: service_healthy` НЕ гарантирует что PostgreSQL готов принимать соединения. App запустится раньше и упадёт на `db.Ping()`. Healthcheck + condition решает это.

Создай `.env.example`:
```
DATABASE_URL=postgres://postgres:qwerty@localhost:5433/expense_tracker?sslmode=disable
JWT_SECRET=super-secret-dev-key
SERVER_PORT=8080
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
TELEGRAM_BOT_TOKEN=
```

#### Шаг 8: Проверка

```bash
# 1. unit + integration тесты
go test ./... -v

# 2. graceful shutdown
go run ./cmd/server
# в другом терминале: curl localhost:8080/api/v1/stats/monthly
# Ctrl+C → "shutting down server..." → "server stopped gracefully"

# 3. rate limiting
for i in $(seq 1 30); do curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/login; done
# после ~20 запросов должен быть 429

# 4. logging — в stdout сервера должны быть строки вида:
# INFO request method=GET path=/login status=200 duration=1.2ms ip=127.0.0.1:54321

# 5. docker
docker-compose up --build
# в браузере: localhost:8080 → login → dashboard
# Ctrl+C → graceful shutdown
```

---

### День 8: Telegram Bot — `gopkg.in/telebot.v3` (5-6 ч)

**Цель:** Telegram бот для добавления/просмотра/удаления расходов. Привязка аккаунта через 6-значный код (безопасно — пароль не вводится в Telegram).

**Зависимость:** `go get gopkg.in/telebot.v3`

**Где взять токен:** @BotFather → /newbot → скопируй токен → `export TELEGRAM_BOT_TOKEN=...`

#### Шаг 1: Миграция — telegram_chat_id (10 мин)

Создай `migrations/000005_add_telegram_chat_id.up.sql`:
```sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS telegram_chat_id BIGINT UNIQUE;
```

Создай `migrations/000005_add_telegram_chat_id.down.sql`:
```sql
ALTER TABLE users DROP COLUMN IF EXISTS telegram_chat_id;
```

#### Шаг 2: Обнови domain.User (5 мин)

В `internal/domain/user.go` добавь поле:
```go
type User struct {
    ID             int       `json:"id"`
    Email          string    `json:"email"`
    PasswordHash   string    `json:"-"`
    Name           string    `json:"name"`
    OAuthProvider  string    `json:"oauth_provider,omitempty"`
    OAuthID        string    `json:"oauth_id,omitempty"`
    TelegramChatID *int64    `json:"telegram_chat_id,omitempty"`
    CreatedAt      time.Time `json:"created_at"`
}
```

**Ловушка:** `*int64` (pointer) — потому что NULL в PostgreSQL. Если сделать просто `int64`, то для пользователей без Telegram будет 0, и Scan упадёт с `converting NULL to int64 is unsupported`.

#### Шаг 3: Обнови UserRepository (15 мин)

Добавь методы в `internal/repository/user_repository.go`:
```go
GetByTelegramChatID(ctx context.Context, chatID int64) (*domain.User, error)
LinkTelegram(ctx context.Context, userID int, chatID int64) error
```

Реализация в `internal/repository/postgres/user_repo.go`:
```go
func (r *userRepo) GetByTelegramChatID(ctx context.Context, chatID int64) (*domain.User, error) {
    query := `SELECT id, email, password_hash, name, COALESCE(oauth_provider,''), COALESCE(oauth_id,''), telegram_chat_id, created_at
              FROM users WHERE telegram_chat_id = $1`
    var u domain.User
    err := r.db.QueryRowContext(ctx, query, chatID).Scan(
        &u.ID, &u.Email, &u.PasswordHash, &u.Name,
        &u.OAuthProvider, &u.OAuthID, &u.TelegramChatID, &u.CreatedAt,
    )
    if err == sql.ErrNoRows {
        return nil, domain.ErrNotFound
    }
    return &u, err
}

func (r *userRepo) LinkTelegram(ctx context.Context, userID int, chatID int64) error {
    query := `UPDATE users SET telegram_chat_id = $1 WHERE id = $2`
    _, err := r.db.ExecContext(ctx, query, chatID, userID)
    return err
}
```

**Ловушка:** ВСЕ существующие SELECT-ы в user_repo.go (GetByEmail, GetByID, GetByOAuth) теперь тоже должны включать `telegram_chat_id` в SELECT и Scan. Иначе — `Scan error: expected 8 destination arguments, not 7`. Обнови ВСЕ три метода:

```go
// пример для GetByEmail (аналогично для GetByID и GetByOAuth):
query := `SELECT id, email, password_hash, name, COALESCE(oauth_provider,''), COALESCE(oauth_id,''), telegram_chat_id, created_at
          FROM users WHERE email = $1`
err := r.db.QueryRowContext(ctx, query, email).Scan(
    &u.ID, &u.Email, &u.PasswordHash, &u.Name,
    &u.OAuthProvider, &u.OAuthID, &u.TelegramChatID, &u.CreatedAt,
)
```

#### Шаг 4: Config — Telegram token (5 мин)

В `internal/config/config.go` добавь:
```go
type Config struct {
    // ... existing fields ...
    TelegramBotToken string
}

// в LoadConfig():
TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
```

#### Шаг 5: Link Code Store — in-memory с TTL (20 мин)

Создай `internal/service/link_code_store.go`:
```go
package service

import (
    "crypto/rand"
    "fmt"
    "sync"
    "time"
)

type LinkCode struct {
    Code      string
    ChatID    int64
    ExpiresAt time.Time
}

type LinkCodeStore struct {
    mu    sync.Mutex
    codes map[string]*LinkCode // code -> LinkCode
}

func NewLinkCodeStore() *LinkCodeStore {
    store := &LinkCodeStore{
        codes: make(map[string]*LinkCode),
    }

    // cleanup expired codes every minute
    go func() {
        for {
            time.Sleep(time.Minute)
            store.mu.Lock()
            for code, lc := range store.codes {
                if time.Now().After(lc.ExpiresAt) {
                    delete(store.codes, code)
                }
            }
            store.mu.Unlock()
        }
    }()

    return store
}

func (s *LinkCodeStore) Generate(chatID int64) string {
    s.mu.Lock()
    defer s.mu.Unlock()

    // remove any existing code for this chatID
    for code, lc := range s.codes {
        if lc.ChatID == chatID {
            delete(s.codes, code)
        }
    }

    // generate 6-digit code
    b := make([]byte, 3)
    rand.Read(b)
    code := fmt.Sprintf("%06d", int(b[0])<<16|int(b[1])<<8|int(b[2])%1000000)
    code = code[len(code)-6:] // ensure 6 digits

    s.codes[code] = &LinkCode{
        Code:      code,
        ChatID:    chatID,
        ExpiresAt: time.Now().Add(10 * time.Minute),
    }
    return code
}

func (s *LinkCodeStore) Verify(code string) (int64, bool) {
    s.mu.Lock()
    defer s.mu.Unlock()

    lc, exists := s.codes[code]
    if !exists || time.Now().After(lc.ExpiresAt) {
        return 0, false
    }

    chatID := lc.ChatID
    delete(s.codes, code) // one-time use
    return chatID, true
}
```

#### Шаг 6: Web endpoint для привязки кода (15 мин)

В `internal/handler/web_expense.go` добавь handler:
```go
// HandleLinkTelegram processes the link code form from dashboard
func (h *WebExpenseHandler) HandleLinkTelegram(w http.ResponseWriter, r *http.Request) {
    userID := middleware.GetUserID(r.Context())
    if userID == 0 {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    r.ParseForm()
    code := r.FormValue("code")

    chatID, ok := h.linkCodeStore.Verify(code)
    if !ok {
        w.Write([]byte(`<p class="text-red-500 text-sm mt-2">Invalid or expired code</p>`))
        return
    }

    if err := h.userRepo.LinkTelegram(r.Context(), userID, chatID); err != nil {
        // UNIQUE constraint on telegram_chat_id — this Telegram account is already linked
        if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
            w.Write([]byte(`<p class="text-red-500 text-sm mt-2">This Telegram account is already linked to another user</p>`))
            return
        }
        w.Write([]byte(`<p class="text-red-500 text-sm mt-2">Failed to link account</p>`))
        return
    }

    w.Write([]byte(`<p class="text-green-500 text-sm mt-2">Telegram linked successfully!</p>`))
}
```

Для этого нужно расширить `WebExpenseHandler`:
```go
type WebExpenseHandler struct {
    expenseService *service.ExpenseService
    authService    *service.AuthService
    linkCodeStore  *service.LinkCodeStore
    userRepo       repository.UserRepository
    rowTemplate    *template.Template
    listTemplate   *template.Template
}

func NewWebExpenseHandler(
    expenseService *service.ExpenseService,
    authService *service.AuthService,
    linkCodeStore *service.LinkCodeStore,
    userRepo repository.UserRepository,
) *WebExpenseHandler {
    // ... same template loading ...
    return &WebExpenseHandler{
        expenseService: expenseService,
        authService:    authService,
        linkCodeStore:  linkCodeStore,
        userRepo:       userRepo,
        rowTemplate:    rowTmpl,
        listTemplate:   listTmpl,
    }
}
```

**Ловушка:** конструктор `NewWebExpenseHandler` меняет сигнатуру (добавляются 2 аргумента). Обнови вызов в `cmd/server/main.go`:
```go
linkCodeStore := service.NewLinkCodeStore()
webExpenseHandler := handler.NewWebExpenseHandler(svc, authSvc, linkCodeStore, userRepo)
```

Добавь imports `"strings"` и `"go-expense-tracker/internal/repository"` в web_expense.go.

#### Шаг 7: Telegram link form на Dashboard (10 мин)

В `templates/pages/dashboard.html` добавь перед `</div>` (в конце, перед Chart.js script):
```html
<!-- telegram link -->
<div class="bg-white p-6 rounded-lg shadow-md">
    <h2 class="text-lg font-bold mb-4">Link Telegram</h2>
    <p class="text-sm text-gray-600 mb-4">Enter the 6-digit code from the Telegram bot to link your account.</p>
    <form hx-post="/telegram/link" hx-target="#telegram-result" hx-swap="innerHTML">
        <div class="flex gap-2">
            <input type="text" name="code" placeholder="000000" maxlength="6" pattern="[0-9]{6}"
                   class="px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 w-32 text-center text-lg tracking-widest" required>
            <button type="submit" class="bg-indigo-600 text-white font-bold py-2 px-4 rounded-lg hover:bg-indigo-700">Link</button>
        </div>
    </form>
    <div id="telegram-result" class="mt-2"></div>
</div>
```

Зарегистрируй роут в `cmd/server/main.go`:
```go
mux.HandleFunc("POST /telegram/link", authMiddleware(webExpenseHandler.HandleLinkTelegram))
```

#### Шаг 8: Telegram Bot (60 мин)

Создай `internal/bot/bot.go`:
```go
package bot

import (
    "context"
    "fmt"
    "log/slog"
    "strconv"
    "strings"
    "sync"
    "time"

    "go-expense-tracker/internal/domain"
    "go-expense-tracker/internal/repository"
    "go-expense-tracker/internal/service"

    tele "gopkg.in/telebot.v3"
)

type userSession struct {
    state    string // "", "awaiting_category", "awaiting_amount", "awaiting_comment"
    category string
    amount   float64
}

type Bot struct {
    bot            *tele.Bot
    expenseService *service.ExpenseService
    userRepo       repository.UserRepository
    linkCodeStore  *service.LinkCodeStore
    sessions       map[int64]*userSession
    mu             sync.RWMutex
}

func New(token string, expSvc *service.ExpenseService, userRepo repository.UserRepository, linkStore *service.LinkCodeStore) (*Bot, error) {
    pref := tele.Settings{
        Token:  token,
        Poller: &tele.LongPoller{Timeout: 10 * time.Second},
    }

    b, err := tele.NewBot(pref)
    if err != nil {
        return nil, err
    }

    handler := &Bot{
        bot:            b,
        expenseService: expSvc,
        userRepo:       userRepo,
        linkCodeStore:  linkStore,
        sessions:       make(map[int64]*userSession),
    }

    b.Handle("/start", handler.handleStart)
    b.Handle("/help", handler.handleHelp)
    b.Handle("/add", handler.handleAdd)
    b.Handle("/list", handler.handleList)
    b.Handle("/total", handler.handleTotal)
    b.Handle("/cancel", handler.handleCancel)
    b.Handle(tele.OnText, handler.handleText)

    return handler, nil
}

func (h *Bot) Start() {
    slog.Info("telegram bot started")
    h.bot.Start()
}

func (h *Bot) Stop() {
    h.bot.Stop()
    slog.Info("telegram bot stopped")
}

// getUser finds linked user by chatID
func (h *Bot) getUser(chatID int64) (*domain.User, error) {
    return h.userRepo.GetByTelegramChatID(context.Background(), chatID)
}

func (h *Bot) handleStart(c tele.Context) error {
    chatID := c.Chat().ID
    user, err := h.getUser(chatID)
    if err == nil {
        return c.Send(fmt.Sprintf("Welcome back, %s! Use /help to see commands.", user.Name))
    }

    // generate link code
    code := h.linkCodeStore.Generate(chatID)
    return c.Send(fmt.Sprintf(
        "Welcome! To link your account:\n\n"+
            "1. Go to the Dashboard on the website\n"+
            "2. Enter this code: *%s*\n"+
            "3. Click \"Link\"\n\n"+
            "Code expires in 10 minutes.",
        code,
    ), &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}

func (h *Bot) handleHelp(c tele.Context) error {
    return c.Send(
        "/add - add new expense\n"+
            "/list - last 20 expenses\n"+
            "/total - total expenses\n"+
            "/cancel - cancel current action\n"+
            "/start - link account",
    )
}

func (h *Bot) handleAdd(c tele.Context) error {
    chatID := c.Chat().ID
    if _, err := h.getUser(chatID); err != nil {
        return c.Send("Link your account first. Use /start")
    }

    h.mu.Lock()
    h.sessions[chatID] = &userSession{state: "awaiting_category"}
    h.mu.Unlock()

    return c.Send("Enter category (e.g. Food, Transport, Shopping):")
}

func (h *Bot) handleCancel(c tele.Context) error {
    chatID := c.Chat().ID
    h.mu.Lock()
    delete(h.sessions, chatID)
    h.mu.Unlock()
    return c.Send("Cancelled.")
}

func (h *Bot) handleList(c tele.Context) error {
    chatID := c.Chat().ID
    user, err := h.getUser(chatID)
    if err != nil {
        return c.Send("Link your account first. Use /start")
    }

    expenses, err := h.expenseService.GetAllExpenses(context.Background(), user.ID)
    if err != nil {
        return c.Send("Error loading expenses.")
    }

    if len(expenses) == 0 {
        return c.Send("No expenses yet. Use /add to create one.")
    }

    // show last 20
    limit := 20
    if len(expenses) < limit {
        limit = len(expenses)
    }

    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("Last %d expenses:\n\n", limit))
    for i := 0; i < limit; i++ {
        e := expenses[i]
        sb.WriteString(fmt.Sprintf(
            "%d. %.0f ₸ — %s",
            e.ID, e.Amount, e.Category,
        ))
        if e.Comment != "" {
            sb.WriteString(fmt.Sprintf(" (%s)", e.Comment))
        }
        sb.WriteString(fmt.Sprintf(" [%s]\n", e.Date.Format("02.01")))
    }
    return c.Send(sb.String())
}

func (h *Bot) handleTotal(c tele.Context) error {
    chatID := c.Chat().ID
    user, err := h.getUser(chatID)
    if err != nil {
        return c.Send("Link your account first. Use /start")
    }

    total, err := h.expenseService.GetTotal(context.Background(), user.ID)
    if err != nil {
        return c.Send("Error calculating total.")
    }

    return c.Send(fmt.Sprintf("Total expenses: %.0f ₸", total))
}

func (h *Bot) handleText(c tele.Context) error {
    chatID := c.Chat().ID

    h.mu.RLock()
    session, exists := h.sessions[chatID]
    h.mu.RUnlock()

    if !exists {
        return c.Send("Use /help to see available commands.")
    }

    text := strings.TrimSpace(c.Text())

    switch session.state {
    case "awaiting_category":
        if text == "" {
            return c.Send("Category cannot be empty. Try again:")
        }
        h.mu.Lock()
        session.category = text
        session.state = "awaiting_amount"
        h.mu.Unlock()
        return c.Send(fmt.Sprintf("Category: %s\nEnter amount:", text))

    case "awaiting_amount":
        amount, err := strconv.ParseFloat(text, 64)
        if err != nil || amount <= 0 {
            return c.Send("Invalid amount. Enter a positive number:")
        }
        h.mu.Lock()
        session.amount = amount
        session.state = "awaiting_comment"
        h.mu.Unlock()
        return c.Send(fmt.Sprintf("Category: %s\nAmount: %.0f ₸\nEnter comment (or /skip):", session.category, amount))

    case "awaiting_comment":
        comment := text
        if text == "/skip" {
            comment = ""
        }

        user, err := h.getUser(chatID)
        if err != nil {
            return c.Send("Account error. Try /start")
        }

        exp, err := h.expenseService.CreateExpense(
            context.Background(),
            session.category,
            session.amount,
            comment,
            user.ID,
        )
        if err != nil {
            return c.Send(fmt.Sprintf("Error: %s", err.Error()))
        }

        h.mu.Lock()
        delete(h.sessions, chatID)
        h.mu.Unlock()

        return c.Send(fmt.Sprintf(
            "Added: %.0f ₸ — %s%s",
            exp.Amount, exp.Category,
            func() string {
                if comment != "" {
                    return fmt.Sprintf(" (%s)", comment)
                }
                return ""
            }(),
        ))
    }

    return nil
}
```

#### Шаг 9: Wiring в main.go (15 мин)

В `cmd/server/main.go`:

1. Добавь import `"go-expense-tracker/internal/bot"`

2. После создания handlers, перед `srv := &http.Server{...}`:
```go
// telegram bot (optional)
if cfg.TelegramBotToken != "" {
    tgBot, err := bot.New(cfg.TelegramBotToken, svc, userRepo, linkCodeStore)
    if err != nil {
        log.Printf("telegram bot init failed: %v", err)
    } else {
        go tgBot.Start()
        defer tgBot.Stop()
    }
}
```

**Ловушка:** `go tgBot.Start()` — в отдельной goroutine! `Start()` блокирует (long polling). Если вызовешь без `go` — HTTP сервер не запустится.

**Ловушка:** `defer tgBot.Stop()` — нужен чтобы бот остановился при graceful shutdown. Defer выполнится когда main() вернётся (после `srv.Shutdown`).

**Ловушка:** `log.Printf` (не `log.Fatalf`) для ошибки бота — сервер должен работать даже без Telegram. Бот опционален.

#### Шаг 10: Проверка

```bash
# 1. Установи токен
export TELEGRAM_BOT_TOKEN=<your-token-from-botfather>

# 2. Запусти сервер
go run ./cmd/server

# 3. В Telegram
# /start → получишь 6-значный код
# На сайте: Dashboard → введи код → "Telegram linked successfully!"

# 4. В Telegram
# /add → Food → 500 → lunch → "Added: 500 ₸ — Food (lunch)"
# /list → видишь все расходы (включая те что с сайта)
# /total → общая сумма

# 5. На сайте — расход из Telegram должен появиться в /expenses

# 6. docker-compose (обнови environment):
# TELEGRAM_BOT_TOKEN=<token>
docker-compose up --build
```

---

### Общие ловушки (уроки из Дней 1-5)

1. **JWT claims:** всегда `"sub"` (не `"user_id"`). `VerifyToken` ожидает `"sub"`.
2. **HTMX error display:** возвращай 200 + HTML с ошибкой. HTMX по умолчанию не swap-ает на 4xx/5xx.
3. **Cookie:** `HttpOnly: true, Path: "/", MaxAge: 86400, SameSite: Lax`.
4. **Пароль:** 8-32 символа, upper + lower + digit + special. Для seed-скрипта используй `Demo1234!`.
5. **Template field names:** проверяй что `.Comment` (не `.Description`), `.Amount`, `.Category` совпадают с `domain.Expense`.
6. **Form field names:** `name="comment"` (не `name="description"`), `name="amount"`, `name="category"`.
7. **OAuth:** скрывай кнопки если env vars не установлены (`IsGoogleConfigured()`).
8. **Интерфейс:** добавил метод в interface → ОБЯЗАН добавить в ВСЕ реализации (postgres + mock).
9. **SQL Scan:** количество полей в SELECT ДОЛЖНО совпадать с количеством аргументов в Scan. При добавлении `telegram_chat_id` — обнови ВСЕ SELECT-ы.
10. **NULL fields:** используй `*int64` для nullable BIGINT, `COALESCE` для nullable VARCHAR.
11. **Docker templates:** `COPY --from=builder /app/templates /templates` — без этого templates не найдутся.
12. **Комментарии:** на английском, минималистичные, lowercase (кроме имён функций).
13. **Graceful shutdown:** проверяй `err != http.ErrServerClosed` чтобы не логировать ложную ошибку.
14. **Telegram bot:** `go tgBot.Start()` (с `go`!) — иначе заблокирует main.
15. **UNIQUE constraint:** `telegram_chat_id` — UNIQUE в БД. При попытке привязать уже занятый Telegram — перехвати ошибку и покажи user-friendly сообщение, а не 500.
16. **fetch credentials:** в dashboard.html используй `{ credentials: 'same-origin' }` в fetch для надёжной передачи cookie.

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
3. `go run ./cmd/seed` — 1200 расходов для demo
4. Браузер: login demo@demo.com / Demo1234! → expenses list → dashboard с графиками
5. CLI: login → add/list/total/update/delete → всё работает
6. Telegram: /start → код → привязка на сайте → /add → /list → /total → /delete
7. OAuth Google/GitHub → вход через браузер
8. Rate limiting: спам curl → 429
9. `Ctrl+C` → "Server stopped gracefully" + бот остановлен
10. `docker-compose up` → данные на месте (PostgreSQL volume)
11. Все три клиента (браузер, CLI, Telegram) видят одни и те же данные
