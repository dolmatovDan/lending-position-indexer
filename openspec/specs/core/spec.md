# Core Specification

## Purpose

Core indexing цикл: blocks processor, positions manager и модель `Position` (leg-row). Описывает подписку на новые блоки Ethereum mainnet, обновление позиций по всем протоколам и кошелькам и единый формат строки позиции.

## Requirements

### Requirement: Подписка на новые блоки (blocks processor)

`internal/processors/blocks` ДОЛЖЕН отслеживать новые блоки Ethereum mainnet (через
`clients/ethnode`) и на каждом блоке вызывать `positions.Manager.UpdateForBlock`.

#### Scenario: Новый блок через websocket-подписку
- GIVEN сконфигурирован `RPC_WS_URL` и сервис запущен
- WHEN в сети появляется новый блок высотой H
- THEN сервис берёт целевой блок `H - CONFIRMATIONS` (по умолчанию `H`), получает его
      заголовок (number, timestamp) и запускает цикл обновления позиций для этого блока

#### Scenario: Fallback на polling при отсутствии websocket
- GIVEN `RPC_WS_URL` недоступен/не задан, но задан `RPC_HTTP_URL`
- WHEN сервис не может установить ws-подписку
- THEN сервис периодически опрашивает текущую голову блока по HTTP и обрабатывает каждый
      новый номер блока ровно один раз

#### Scenario: Консистентный срез на номере блока
- GIVEN обрабатывается блок N
- WHEN выполняются контракт-вызовы для сбора позиций
- THEN все вызовы выполняются с фиксацией `blockNumber = N`, чтобы данные относились к
      одному состоянию сети

---

### Requirement: Обновление позиций (positions manager)

`positions.Manager.UpdateForBlock` ДОЛЖЕН на каждом обрабатываемом блоке для каждого
инжектированного протокола (`[]protocols.Reader`) собрать позиции (адаптеры возвращают
строки уже с разрешёнными on-chain ценами) по всему списку кошельков и сохранить результат
через repository.

#### Scenario: Обновление по всем протоколам и кошелькам
- GIVEN список кошельков из конфига и инжектированные протоколы (Aave v3, Euler v2)
- WHEN обрабатывается новый блок N
- THEN для каждого протокола вызывается сбор позиций по всем кошелькам (с ценами из
      on-chain источников внутри адаптера), и строки сохраняются с `block_number = N` и
      timestamp блока

#### Scenario: Отказ одного протокола не роняет остальные
- GIVEN при обработке блока N один из адаптеров протокола вернул ошибку
- WHEN manager обрабатывает остальные протоколы
- THEN ошибка логируется, обработка прочих протоколов и кошельков продолжается, сервис не
      завершает работу

#### Scenario: Устойчивость к временным сбоям RPC
- GIVEN временная ошибка RPC при обработке блока
- WHEN выполняется вызов с ретраями/backoff
- THEN при восстановлении соединения обработка продолжается со следующего блока без падения
      процесса

---

### Requirement: Модель позиции (leg-row)

`Position` — единый формат для всех протоколов: **строка на актив**. Пулный кросс-коллатерал
(Aave) и изолированные vault'ы (Euler) представляются одинаково, без синтетических пар.

#### Scenario: Полнота полей строки позиции
- GIVEN собрана строка позиции по кошельку в некотором протоколе на блоке N
- WHEN формируется запись `Position`
- THEN она содержит: `protocol`, `wallet_address`, `market_id` (market/pool/vault),
      `token` (address/symbol/decimals), `side` (`collateral`|`debt`), `amount` (position
      size актива), `price` (token price, on-chain), `health_factor`, `block_number`,
      `timestamp`

#### Scenario: Health factor на scope
- GIVEN у кошелька в протоколе несколько активов залога/долга (Aave) или пара vault'ов (Euler)
- WHEN формируются строки одного scope (`market_id`)
- THEN все строки этого scope несут один и тот же `health_factor` (аккаунт Aave /
      контроллер Euler); HF — атрибут scope, а не отдельного актива

#### Scenario: Разбивка collateral/debt по строкам
- GIVEN кошелёк с залогом в одном активе и долгом в другом
- WHEN формируются строки `Position`
- THEN залог и долг — это **отдельные строки** с `side=collateral` и `side=debt`
      соответственно, каждая со своим `token`/`amount`/`price`
