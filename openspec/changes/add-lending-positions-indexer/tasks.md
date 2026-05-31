# Implementation Tasks: Сервис индексации lending-позиций

**Change ID:** `add-lending-positions-indexer`

---

## Phase 1: Foundation (git, каркас, models, config, bindings)

- [x] 1.1 Инициализировать git-репозиторий: `git init`, `.gitignore` (Go/bin/`.env`),
      привязать remote `origin` → `https://github.com/dolmatovDan/lending-position-indexer.git`,
      первый коммит на ветке (не в `main` напрямую).
- [x] 1.2 Инициализировать модуль: `go.mod` (Go 1.22+), структура по ARCH: `cmd/positions`,
      `internal/{api/http,clients,managers,repositories,models,protocols,bindings}`, `abi/`,
      `migrations/postgres/`.
- [x] 1.3 `cmd/positions/config.go`: `Config` struct с env-тегами (`RPC_WS_URL`,
      `RPC_HTTP_URL`, `WALLETS`, `CONFIRMATIONS`, `EULER_SUBACCOUNTS_DEPTH`, `DATABASE_URL`,
      `LOG_LEVEL`, `HTTP_PORT`); загрузка и валидация.
- [x] 1.4 `internal/models`: `Position` (leg-модель: строка на актив — `token`, `side`,
      `amount`, `price`, `health_factor`, scope `market_id`; числа — `decimal`/строка),
      `Token{addr,symbol,decimals}`, enum `ProtocolKind`, enum `Side` (collateral/debt),
      `errors.go` (sentinel-ошибки).
- [x] 1.5 `internal/bindings`: ABI-файлы в `abi/` (Aave Pool, AaveProtocolDataProvider,
      AaveOracle, Euler EVC, Euler EVault, EulerOracle, ERC20) + `go:generate` abigen.
- [x] 1.6 `cmd/positions/setup_core.go`: инициализация `log/slog` (JSON, уровень из конфига).

**Quality Gate:**
- [x] `go vet ./...` / `go build ./...` проходят
- [x] Unit-тесты конфига зелёные

---

## Phase 2: Clients (ethnode, протоколы, pricing)

- [x] 2.1 `internal/clients/ethnode`: обёртка ethclient — `SubscribeNewHead` (ws) + fallback-
      поллинг (http); caller с фиксацией block number; block number/timestamp. Interface + stub.
- [x] 2.2 `internal/protocols`: общий интерфейс `Reader` (`Name`, `Positions(ctx, wallets, block) []models.Position`)
      — возвращает leg-строки уже с разрешёнными ценами.
- [x] 2.3 `internal/pricing`: общие helpers — нормализация decimals/масштабирование, чтение
      Chainlink-агрегатора; используются адаптерами (отдельного pricing-клиента нет).
- [x] 2.4 `internal/clients/aave`: адаптер Aave v3 — `getReservesList`, `getUserReserveData`,
      `getUserAccountData` (HF аккаунта), цены через AaveOracle; формирует leg-строки
      (side collateral/debt). Interface + stub.
- [x] 2.5 `internal/clients/euler`: адаптер Euler v2 — дискавери через `EVC.getControllers`/
      `getCollaterals` (+ суб-аккаунты), `convertToAssets`/`debtOf`, HF через `accountLiquidity`,
      цены через oracle-роутер vault'а; формирует leg-строки. Interface + stub.
- [x] 2.6 Unit-тесты: математика HF (Aave и Euler), конвертация активов, масштабирование цен,
      формирование leg-строк — на эталонных значениях, со stub'ами клиентов.

**Quality Gate:**
- [x] Оба адаптера покрыты тестами на расчёты
- [x] Нет обращений к внешним ценовым API (проверка по коду/зависимостям)

---

## Phase 3: Repository, Manager, Processor, API

- [x] 3.1 `internal/repositories/positions`: interface + factory + `postgres/` (pgx); таблица
      `positions` (история по блокам), индексы, upsert/выборки; миграции в `migrations/postgres`.
- [x] 3.2 `internal/managers/positions`: `Manager` с конструктором (инжект `[]protocols.Reader`,
      repository); `UpdateForBlock` — сбор leg-строк по протоколам×кошелькам (строки уже с
      ценами) → сохранение; ограничение конкуррентности, обработка ошибок.
- [x] 3.3 `internal/processors/blocks`: драйвер цикла — на новый блок вызывает
      `Manager.UpdateForBlock` (с учётом `CONFIRMATIONS`), ретраи/backoff.
- [x] 3.4 `internal/api/http/servant.go`: chi-роутер + тонкие handlers — `GET /positions?wallet=&protocol=`,
      `GET /healthz`; маппинг domain-ошибок в HTTP-статусы; сериализация `Position` в JSON.
- [x] 3.5 `cmd/positions/{main.go,setup_clients,setup_repositories,setup_managers,setup_processors}.go`:
      startup-последовательность ARCH, graceful shutdown.
- [x] 3.6 Тесты repository (sqlmock/dockertest) и handlers API.

**Quality Gate:**
- [x] Миграции применяются на чистой БД
- [x] API-handlers покрыты тестами

---

## Phase 4: Integration & Polish

- [ ] 4.1 `Dockerfile` (многостадийный build → минимальный runtime-образ).
- [ ] 4.2 `docker-compose.yml`: сервисы `app` + `postgres` (volume, healthcheck, depends_on),
      `.env.example`.
- [ ] 4.3 Сквозные логи ключевых событий (новый блок, число позиций, ошибки RPC/БД).
- [ ] 4.4 `README.md`: архитектура (слои ARCH), запуск через docker-compose, переменные
      окружения, как добавить новый протокол и новую EVM-сеть, известные ограничения.
- [ ] 4.5 Опциональный integration-тест за build-tag с реальным RPC (skip по умолчанию).

**Quality Gate:**
- [ ] `go test ./...` зелёный
- [ ] `docker compose up` поднимает app + postgres, видна обработка блоков
- [ ] README актуален

---

## Completion Checklist

- [ ] Все фазы завершены
- [ ] Все quality gates пройдены
- [ ] Документация синхронизирована
- [ ] Готово к `/openspec-archive`
