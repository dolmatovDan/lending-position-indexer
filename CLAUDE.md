# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Что это

Lending Positions Indexer — Go-сервис, который на каждом новом блоке Ethereum mainnet читает текущие
lending-позиции по заданному списку кошельков из **Aave v3** и **Euler v2**, разрешает цены токенов
**только из on-chain источников** (протокольные оракулы + Chainlink), сохраняет историю в PostgreSQL и
отдаёт через REST. Каждая сохранённая строка — это один «leg» (протокол × кошелёк × рынок × токен ×
сторона). `amount` и `price` хранятся как точные `NUMERIC` / `shopspring/decimal` — никогда не float.

Код и идентификаторы на английском; README и прочая проза — на русском.

## Команды

```bash
go build ./...                      # сборка всего
go test ./...                       # unit-тесты (без RPC/БД — используются stub-клиенты + pgxmock)
go test -tags=integration ./...     # + integration-тесты (нужны RPC_HTTP_URL и TEST_WALLET, иначе skip)
go test ./internal/managers/...     # тест одного пакета
go test -run TestName ./path/...    # один тест
go vet ./...                        # vet
gofmt -l .                          # список неотформатированных файлов

docker compose up --build           # весь стек: postgres + app (сначала скопировать .env.example -> .env)
go run ./cmd/positions              # локальный запуск (нужны env-переменные; таблица в README)
```

Миграции применяются автоматически при старте (golang-migrate, встроенный SQL); отдельного шага migrate нет.

## Архитектура

Строгая слоёная композиция с явной инъекцией зависимостей через конструкторы — **без глобалов и
`init()`-реестров** (единственное исключение — `internal/bindings`, который грузит ABI в `init()`).

```
Processor (цикл блоков) → Manager (бизнес-логика) → Repository (доступ к данным) → Postgres
                              ↓ использует
                         []protocols.Reader  (адаптеры Aave, Euler)  → Clients → Ethereum node
HTTP handlers → Manager / Repository
```

Вся проводка зависимостей живёт в `cmd/positions/setup_*.go` и вызывается по порядку из `run()` в
`main.go`: `config → core(logger) → clients → repositories → managers → processors → http`. Каждый
`setup_*.go` собирает один слой и передаёт его следующему.

Ключевые швы:
- **`internal/protocols.Reader`** — абстракция протокола: `Name()` + `Positions(...)`. Любой lending-
  протокол — это просто `Reader`, возвращающий leg-строки с уже разрешёнными on-chain ценами. Manager
  обходит `[]protocols.Reader` конкурентно (по горутине на reader) и не знает деталей протоколов.
- **`internal/repositories/positions`** — интерфейс + фабрика `New()`, возвращающая реализацию из
  `postgres/`. Manager зависит от интерфейса, поэтому в тестах используется `pgxmock`.
- **`internal/clients/<proto>`** — каждый клиент протокола определяет *интерфейс контрактов* (нужные
  on-chain вызовы), реальную реализацию поверх `bindings` и `stub.go` (`StubContracts`) для unit-тестов.
  Экспортируемый адаптер (`aave.New`, `euler.New`) реализует `protocols.Reader`.
- **`internal/bindings`** — грузит встроенные ABI (`abi/*.json` через `//go:embed`) и даёт generic-
  caller контрактов; директивы abigen `//go:generate` присутствуют, но ABI грузятся в рантайме, а не
  через сгенерированные биндинги.
- **`internal/clients/ethnode`** — обёртка `ethclient`: ws-подписка на новые heads с fallback на http-
  поллинг (решает `SupportsSubscriptions()`). `blocks.Processor` ведёт цикл индексации, применяет лаг
  `CONFIRMATIONS` (обрабатывает блок `H-k`), добивает пропущенные блоки между heads и ретраит каждый
  блок с backoff.

Адреса контрактов по протоколам берутся из `MainnetConfig()` в каждом клиентском пакете (chainID,
Pool/EVC/oracle). Добавить новую EVM-сеть = добавить ещё один набор адресов; код адаптеров переиспользуется.

## Конвенции, специфичные для репозитория

- **Добавление протокола**: создать `internal/clients/<proto>/` (интерфейс контрактов + on-chain
  реализация + stub), добавить ABI в `abi/` и зарегистрировать в `internal/bindings`, затем дописать
  `<proto>.New(...)` в срез `[]protocols.Reader` в `cmd/positions/setup_clients.go`. Слои
  manager/repository/api трогать **не нужно** — расширение только через срез readers. Также добавить
  новый `ProtocolKind` в `models` и в `knownProtocol()` в манагере, чтобы фильтр `?protocol=` его принимал.
- Цены и размеры — это `shopspring/decimal` сквозняком и `NUMERIC` в SQL — не конвертировать через `float64`.
- Leg, у которого on-chain цена временно недоступна, **пропускается с warning'ом**, не фатально — сервис
  не должен падать из-за отсутствующей цены.
- Aave health factor пишется как `0`, когда у аккаунта нет долга (значит «без долга / бесконечный HF»).
- Индексация идёт только вперёд от текущего head; исторический бэкфилл вне scope.

## Workflow OpenSpec

Проект использует OpenSpec (`openspec/`) для spec-driven изменений; активное изменение лежит в
`openspec/changes/`. Для старта, реализации или финализации отслеживаемого изменения используйте скиллы
`openspec-*` / `opsx:*` (propose, apply, archive, explore), а не правьте specs вручную.
