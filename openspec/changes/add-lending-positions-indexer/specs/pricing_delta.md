# Delta: On-chain ценообразование

**Change ID:** `add-lending-positions-indexer`
**Affects:** `internal/clients/aave`, `internal/clients/euler`, `internal/pricing` (общие helpers)

---

## ADDED

### Requirement: Цены только из on-chain источников

Цена токена ДОЛЖНА браться исключительно из on-chain источников (протокольный оракул,
Chainlink, pool pricing). Использование CoinGecko и аналогичных внешних HTTP-API
запрещено.

#### Scenario: Цена токена Aave через протокольный оракул
- GIVEN токен резерва Aave v3
- WHEN запрашивается цена
- THEN она берётся из `AaveOracle.getAssetPrice(asset)` (USD, 8 decimals), что под капотом
      использует Chainlink

#### Scenario: Цена токена Euler через oracle-роутер vault'а
- GIVEN токен (asset) vault'а Euler v2
- WHEN запрашивается цена
- THEN она берётся из oracle-роутера vault'а: `vault.oracle().getQuote(amount, asset,
      vault.unitOfAccount())` — полностью on-chain

#### Scenario: Отсутствие внешних ценовых API
- GIVEN сборка и зависимости сервиса
- WHEN проверяется код/конфигурация
- THEN отсутствуют обращения к CoinGecko и подобным внешним ценовым API; единственные
      сетевые вызовы для цен — RPC к контрактам

---

### Requirement: Ценообразование внутри адаптеров + общие helpers

Цена токена ДОЛЖНА разрешаться **внутри адаптера протокола** (источник цены протокол-
специфичен и читается теми же контрактами, что и позиция), без отдельного слоя/шага
ценообразования в manager. Общая математика нормализации (decimals/масштабирование, чтение
Chainlink-агрегатора) выносится в helper-пакет `internal/pricing`, используемый адаптерами.

#### Scenario: Согласованное масштабирование цен
- GIVEN источники с разным масштабом и unit of account (Aave 8 decimals USD; Euler — quote
      в unitOfAccount vault'а, часто USD или WETH)
- WHEN адаптер нормализует цену для поля `Position.price` через helpers `internal/pricing`
- THEN значение приводится к единому документированному представлению с учётом decimals
      токена и источника, и хранится как точное число (NUMERIC/строка)

#### Scenario: Цена уже в строке позиции
- GIVEN адаптер сформировал leg-строку
- WHEN `Positions(...)` возвращает результат в manager
- THEN поле `price` уже заполнено on-chain ценой; manager не делает отдельных ценовых вызовов

#### Scenario: Недоступность источника цены
- GIVEN временно недоступен ценовой источник для токена
- WHEN адаптер не может получить цену
- THEN ошибка логируется, строка либо пропускается, либо сохраняется с явной отметкой
      отсутствия цены (поведение документировано), без падения сервиса

---

## MODIFIED

(None — greenfield)

## REMOVED

(None)
