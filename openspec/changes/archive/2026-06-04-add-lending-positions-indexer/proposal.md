# Proposal: Сервис индексации lending-позиций (Ethereum mainnet)

**Change ID:** `add-lending-positions-indexer`
**Created:** 2026-05-30
**Status:** Draft

---

## Problem Statement

- **Что решаем.** Нужен сервис, который по заданному списку кошельков в Ethereum mainnet
  на **каждом новом блоке** собирает актуальные lending-позиции и отдаёт по ним полный
  набор данных: протокол, адрес кошелька, идентификатор рынка/пула/волта, токены
  залога и долга, размер позиции, цену токена, health factor, номер блока и timestamp.
- **Кого касается.** Команды, которым нужен надёжный, верифицируемый источник состояния
  заёмных позиций (риск-мониторинг, ликвидации, аналитика) без зависимости от внешних
  индексеров.
- **Числа.** `amount` (position size) и `price` (token price) хранятся как точные значения
  (NUMERIC/строка), без потери точности на float.
- **Боль сейчас.** Готового решения в репозитории нет (greenfield). Внешние ценовые API
  (CoinGecko и аналоги) **запрещены** — цена должна браться исключительно из on-chain
  источников, что требует аккуратной работы с протокольными оракулами и Chainlink.

## Proposed Solution

Go-сервис, организованный по layered-архитектуре `ARCH.md`
(**Handlers → Managers → Repositories → DB**, плюс **Clients** к внешним сервисам и
**Processors** для асинхронных драйверов). Внутренние Яндекс-библиотеки заменены публичными
аналогами (chi, pgx, golang-migrate, `log/slog`, env-config), паттерны/слои/нейминг
сохранены. Внешний API — только HTTP/REST; gRPC/proto не делаем. Observability — простой
`/healthz` (без отдельного obs-сервера и метрик).

Слои в терминах нашего домена:

- **Clients** (`internal/clients/`) — обёртки над внешними зависимостями, один пакет на
  зависимость: `ethnode/` (ethclient: подписка `SubscribeNewHead` по ws, fallback-поллинг
  по http; вызовы с фиксацией block number), `aave/` и `euler/` (чтение позиций on-chain
  вместе с ценами из протокольного оракула — ценообразование протокол-специфично и живёт
  внутри адаптера). У каждого — интерфейс + stub для тестов.
- **Managers** (`internal/managers/positions/`) — бизнес-логика: на новый блок собрать
  позиции (уже с разрешёнными on-chain ценами) по всем протоколам × кошелькам и сохранить
  через repository. Протоколы инжектятся как `[]protocols.Reader` через конструктор (без
  глобального реестра — по правилу ARCH «зависимости через конструкторы, без глобалов/`init()`»).
- **Repositories** (`internal/repositories/positions/`) — чистый data access к PostgreSQL
  (interface + factory + `postgres/`), история позиций по блокам.
- **Processors** (`internal/processors/blocks/`) — драйвер цикла: подписка на новые блоки
  → `positions.Manager.UpdateForBlock(ctx, block)`. Глубина обработки настраивается через
  `CONFIRMATIONS` (по умолчанию 0 = голова) для устойчивости к реоргам.
- **API/Handlers** (`internal/api/http/`) — тонкий chi-роутер: `GET /positions`, `/healthz`.
- **Models** (`internal/models/`) — `Position` (leg-модель: строка на актив), `Token`,
  enum `ProtocolKind`, enum `Side` (collateral/debt), `errors.go` (sentinel-ошибки домена).
- **cmd/positions/** — `main.go` (строгая startup-последовательность ARCH) +
  `config.go`, `setup_core.go`, `setup_clients.go`, `setup_repositories.go`,
  `setup_managers.go`, `setup_processors.go`.

**Адаптеры протоколов:** обязательный **Aave v3** и **Euler v2** (позиции дискаверятся
on-chain через EVC — внешний список рынков не нужен). **Цены** — только on-chain: оракул
Aave (USD, под капотом Chainlink) и oracle-роутер vault'а Euler (`getQuote`).

**Модель `Position` (leg-row).** Единый формат для всех протоколов — **строка на актив**:
`(protocol, wallet, market_id, token, side[collateral|debt], amount, price, health_factor,
block_number, timestamp)`. Так пулный кросс-коллатерал Aave (множество залогов/долгов, один
HF на аккаунт) и изолированные vault'ы Euler ложатся в одну плоскую модель без синтетических
пар; `token`/`amount`/`price` всегда относятся к одному активу. `health_factor` задаётся на
scope (аккаунт Aave / контроллер Euler) и повторяется на строках этого scope. «Позиция»
пользователя в UI/REST — это группа строк по `(wallet, protocol, market_id)`.

**Ожидаемый результат:** на каждом блоке позиции по списку кошельков обновляются,
сохраняются в БД и доступны через REST.

## Scope

### In Scope
- Подписка на новые блоки mainnet и пер-блочное обновление позиций по списку кошельков.
- Адаптеры **Aave v3** (обязательный) и **Euler v2**.
- Сбор всех требуемых полей позиции, включая health factor.
- On-chain цены (protocol oracle + Chainlink).
- PostgreSQL-хранилище с историей по блокам.
- REST API для чтения, healthcheck.
- docker-compose, тесты, структурные логи, README.
- Расширяемость по протоколам и EVM-сетям на уровне интерфейсов/конфига.

### Out of Scope
- Реализация третьего и далее протоколов (Morpho/Fluid и пр.) — архитектурно возможна,
  но не делается в этом change.
- Подключение других сетей (L2 и т.д.) — закладывается архитектурой, но конкретные сети
  не добавляются.
- Реорганизация исторических данных, бэкфилл по старым блокам (только «с текущего блока
  вперёд»; бэкфилл — потенциальное расширение).
- Алертинг/ликвидации/торговая логика поверх позиций.
- Авторизация/мультитенантность REST API.

## Impact Analysis

| Component | Change Required | Details |
|-----------|-----------------|---------|
| Database  | Yes | Новая схема Postgres: таблица `positions` (история по блокам) + индексы. |
| API       | Yes | Новый REST: `GET /positions`, `GET /healthz`. |
| State     | Yes | `positions.Manager` обновляет позиции на каждом блоке; срез фиксируется на block number. |
| UI        | No  | UI не предусмотрен. |
| Infra     | Yes | Dockerfile, docker-compose (app + postgres), env-конфиг. |

## Architecture Considerations

Следуем layered-паттернам `ARCH.md`, адаптированным под standalone docker-compose проект
(внутренние Яндекс-библиотеки → публичные аналоги).

**Канонический layout (адаптированный под наш домен):**
```
cmd/positions/
  main.go                    startup-последовательность (signal → config → core →
                             clients → migrations → repos → managers → processors →
                             http → graceful shutdown)
  config.go                  Config struct (env-теги)
  setup_core.go              logger (slog), общие зависимости
  setup_clients.go           ethnode, protocol- и pricing-клиенты
  setup_repositories.go      positions repository
  setup_managers.go          positions manager (инжект []protocols.Reader)
  setup_processors.go        blocks processor
internal/
  api/http/servant.go        chi-роутер + middleware; handlers
  clients/ethnode/           обёртка ethclient (interface + stub)
  clients/aave/              чтение позиций + цены Aave v3 (interface + stub)
  clients/euler/             чтение позиций + цены Euler v2 (interface + stub)
  managers/positions/        бизнес-логика обновления позиций
  repositories/positions/    interface + factory + postgres/
  models/                    Position, Token, ProtocolKind, Side, errors.go
  protocols/                 общий интерфейс Reader (контракт адаптеров протоколов)
  pricing/                   общие helpers нормализации цен (decimals/scaling, Chainlink reader)
  bindings/                  abigen Go-биндинги (из abi/)
abi/                         ABI-файлы контрактов
migrations/postgres/         SQL-миграции (применяются при старте)
```

**Ключевые принципы из ARCH:**
- **Тонкие handlers** — только валидация входа, вызов одного метода manager, сериализация.
- **Managers** — вся бизнес-логика; композируют repositories и clients; без прямых вызовов БД.
- **Repositories** — чистый data access; interface + factory (Postgres-реализация сейчас,
  место под другие БД на будущее).
- **Clients** — один пакет на внешнюю зависимость; у каждого интерфейс + stub.
- **Явная DI через конструкторы**, без глобалов и `init()` → вместо глобального реестра
  протоколов manager получает `[]protocols.Reader` явным списком из `setup_managers.go`.
- **Точки расширения:** новый протокол = новый `clients/<proto>/` + строка в
  `setup_managers.go`; новая EVM-сеть = новый конфиг сети (chainID, RPC, адреса).
- Контракт-вызовы абстрагированы интерфейсом (caller) для тестируемости (stub/моки).
- **Sentinel-ошибки** домена в `internal/models/errors.go`, маппинг в HTTP-статусы — в handler.

**Замены библиотек (ARCH → публичные):** dbwrap/sqx → `jackc/pgx/v5`; migrada →
`golang-migrate`; grpc/http servant → `go-chi/chi` (только HTTP); xslog/log3 → `log/slog`;
observability → простой `/healthz`; rex/molniya (hot-reload конфиг) → статический env-конфиг;
tvm → не нужен. Числа — `shopspring/decimal`. Тесты — `stretchr/testify`. EVM —
`github.com/ethereum/go-ethereum`.

## Success Criteria

- [ ] `docker compose up` поднимает сервис и Postgres; в логах виден разбор новых блоков.
- [ ] По каждой строке позиции заполнены все требуемые поля (protocol, wallet, market id,
      token, side[collateral/debt], amount, on-chain price, health factor, block number, timestamp).
- [ ] Цены приходят только из on-chain источников (нет вызовов CoinGecko/внешних API).
- [ ] Работают адаптеры Aave v3 и Euler v2.
- [ ] `GET /positions?wallet=0x...` возвращает актуальные позиции.
- [ ] `go test ./...` зелёный; есть unit-тесты на расчёты (HF, конвертация активов, цены).
- [ ] README описывает запуск и добавление нового протокола/сети.

## Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Нет публичного RPC, выдерживающего пер-блочные запросы | High | High | Конфиг RPC через env (Alchemy/Infura/собственный нод); ws-подписка + backoff. |
| Кол-во суб-аккаунтов Euler на кошелёк (до 256) | Med | Med | Конфиг `EULER_SUBACCOUNTS_DEPTH` (по умолч. 1); дискавери позиций через EVC on-chain, без внешних списков. |
| Расхождения в decimals/масштабировании цен (Aave 8d USD, Euler unitOfAccount) | Med | High | Изолировать математику в `pricing`/адаптерах; покрыть unit-тестами на эталонных значениях. |
| Реорги блокчейна искажают срез | Med | Med | Конфиг `CONFIRMATIONS` (обработка блока N-k, по умолч. 0); хранение block number/timestamp. |
| Тяжёлые `eth_call` на каждый кошелёк×vault | Med | Med | Ограничение конкуррентности, кэш метаданных токенов; опц. Multicall3 как расширение. |
