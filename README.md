# Lending Positions Indexer

Сервис индексации lending-позиций в Ethereum mainnet. На каждом новом блоке собирает
актуальные позиции по заданному списку кошельков из **Aave v3** и **Euler v2**, разрешает
цены **только из on-chain источников** (протокольные оракулы + Chainlink), сохраняет историю
в PostgreSQL и отдаёт данные через REST.

Каждая строка позиции (leg-row) содержит: протокол, адрес кошелька, идентификатор рынка/пула/
волта, токен (address/symbol/decimals), сторону (`collateral`/`debt`), размер позиции, цену
токена, health factor, номер блока и timestamp. `amount` и `price` хранятся как точные
`NUMERIC` без потери точности.

## Архитектура

Layered-композиция (Handlers → Managers → Repositories → DB; плюс Clients и Processors):

```
cmd/positions/            startup: config → core → clients → migrations → repos → managers → processors → http
internal/
  api/http/               chi-роутер + тонкие handlers (/positions, /healthz)
  clients/ethnode/        обёртка ethclient (ws-подписка + http fallback)
  clients/aave/           адаптер Aave v3 (позиции + цены), interface + stub
  clients/euler/          адаптер Euler v2 (дискавери через EVC), interface + stub
  managers/positions/     бизнес-логика: UpdateForBlock по протоколам×кошелькам
  repositories/positions/ data access (interface + factory + postgres/)
  processors/blocks/      драйвер цикла на новые блоки
  protocols/              общий интерфейс Reader
  pricing/                helpers нормализации decimals + чтение Chainlink
  bindings/               загрузка ABI + generic caller
abi/                      ABI-файлы контрактов (embed)
migrations/postgres/      SQL-миграции (применяются при старте, golang-migrate)
```

Зависимости передаются через конструкторы явно, без глобалов и `init()`-реестров. Список
протоколов инжектится как `[]protocols.Reader` в `setup_managers.go`.

## Запуск через docker-compose

1. Скопируйте `.env.example` в `.env` и заполните `RPC_*` и `WALLETS`.
2. Поднимите стек:

```bash
docker compose up --build
```

Поднимутся `postgres` и `app`. Приложение дожидается готовности БД (healthcheck), применяет
миграции, подключается к RPC и начинает обрабатывать новые блоки. В логах видны записи
`block processed` с номером блока, числом позиций и длительностью.

Проверка:

```bash
curl "http://localhost:8080/healthz"
curl "http://localhost:8080/positions?wallet=0xYourWallet"
curl "http://localhost:8080/positions?wallet=0xYourWallet&protocol=euler-v2"
```

## Переменные окружения

| Переменная | Обязательная | По умолчанию | Описание |
|------------|--------------|--------------|----------|
| `RPC_WS_URL` | одна из RPC_* | — | websocket endpoint для подписки на блоки |
| `RPC_HTTP_URL` | одна из RPC_* | — | http endpoint (eth_call + fallback-поллинг) |
| `WALLETS` | да | — | список адресов через запятую |
| `DATABASE_URL` | да | — | DSN PostgreSQL |
| `CONFIRMATIONS` | нет | `0` | лаг подтверждений: обрабатывается блок `H-k` |
| `EULER_SUBACCOUNTS_DEPTH` | нет | `1` | глубина обхода суб-аккаунтов Euler (1 = только основной) |
| `LOG_LEVEL` | нет | `info` | `debug`/`info`/`warn`/`error` |
| `HTTP_PORT` | нет | `8080` | порт REST API |

## Тесты

```bash
go test ./...                       # unit-тесты (HF, конвертация, цены, repository, API)
go test -tags=integration ./...     # + integration (нужны RPC_HTTP_URL и TEST_WALLET)
```

Unit-тесты не требуют RPC/БД (используются stub'ы клиентов и `pgxmock`). Integration-тест за
build-tag `integration` пропускается, если не заданы `RPC_HTTP_URL`/`TEST_WALLET`.

## Как добавить новый протокол

1. Создайте пакет `internal/clients/<proto>/` с интерфейсом контрактов, on-chain реализацией
   и stub'ом; адаптер реализует `protocols.Reader` (`Name()`, `Positions(...)`), возвращая
   leg-строки уже с разрешёнными on-chain ценами.
2. Добавьте нужные ABI в `abi/` и зарегистрируйте их в `internal/bindings`.
3. Добавьте адаптер в срез `[]protocols.Reader` в `cmd/positions/setup_clients.go`.

Слои manager/repository/api менять не нужно — расширение через конструкторную инъекцию.

## Как добавить новую EVM-сеть

Адаптеры используют адреса контрактов из конфигурации сети (`MainnetConfig()` в каждом
адаптере). Для новой сети добавьте соответствующий набор адресов (chainID, Pool/EVC/oracle) и
RPC-endpoint; код адаптеров переиспользуется без изменений.

## Известные ограничения

- **Euler суб-аккаунты:** по умолчанию обходится только основной аккаунт
  (`EULER_SUBACCOUNTS_DEPTH=1`); увеличение глубины повышает полноту, но и нагрузку на RPC.
- **Реорги:** срез фиксируется на номере блока; для устойчивости используйте `CONFIRMATIONS>0`
  (обработка `H-k`).
- **Aave health factor:** при отсутствии долга у аккаунта HF записывается как `0`
  (трактуется как «без долга / бесконечный HF»).
- **Цена недоступна:** если on-chain источник цены временно недоступен, leg-строка
  пропускается с предупреждением в логах (сервис не падает).
- **Бэкфилл:** обработка идёт только «с текущего блока вперёд», исторический бэкфилл не входит
  в scope.
