# Storage API Specification

## Purpose

Хранение позиций в PostgreSQL (repository с историей по блокам) и REST API (chi-роутер) для чтения позиций и проверки состояния.

## Requirements

### Requirement: Хранение позиций в PostgreSQL (repository)

`repositories/positions` (interface + factory + `postgres/`) ДОЛЖЕН сохранять обновлённые
позиции в PostgreSQL с историей по блокам. Manager обращается к данным только через этот
интерфейс (без прямых SQL-вызовов в бизнес-логике).

#### Scenario: Запись leg-строк блока
- GIVEN на блоке N собран набор leg-строк
- WHEN manager сохраняет результат через repository
- THEN каждая строка записывается в таблицу `positions` с ключом
      (protocol, wallet, market_id, token, side, block_number), со всеми полями модели

#### Scenario: Идемпотентность повторной обработки блока
- GIVEN блок N обрабатывается повторно (например, после переподключения)
- WHEN строки сохраняются снова
- THEN выполняется upsert по ключу (protocol, wallet, market_id, token, side, block_number)
      без дублирования записей

#### Scenario: Применение миграций при старте
- GIVEN чистая база данных
- WHEN сервис стартует (шаг migrations в `main.go`, до repositories/managers)
- THEN схема создаётся применением SQL-миграций из `migrations/postgres` (golang-migrate)
      до начала индексации

#### Scenario: Точное хранение числовых значений
- GIVEN leg-строка с `amount` (position size) и `price` (token price)
- WHEN значения записываются в БД
- THEN они хранятся как точные числа (`NUMERIC`/строка) без приведения к float, чтобы
      исключить потерю точности на больших суммах и малых ценах

---

### Requirement: REST API для чтения позиций (http servant + handlers)

`api/http` ДОЛЖЕН предоставлять HTTP REST-интерфейс (chi-роутер) для чтения позиций и
проверки состояния. Handlers тонкие: валидация входа → вызов одного метода manager →
сериализация; domain-ошибки маппятся в HTTP-статусы только здесь.

#### Scenario: Чтение позиций по кошельку
- GIVEN сохранённые позиции
- WHEN клиент вызывает `GET /positions?wallet=0x...`
- THEN handler вызывает `positions.Manager`, возвращается JSON с актуальными позициями
      кошелька, включая все требуемые поля

#### Scenario: Фильтр по протоколу
- GIVEN сохранённые позиции по нескольким протоколам
- WHEN клиент вызывает `GET /positions?wallet=0x...&protocol=euler-v2`
- THEN возвращаются только позиции указанного протокола

#### Scenario: Healthcheck
- GIVEN запущенный сервис
- WHEN клиент вызывает `GET /healthz`
- THEN возвращается статус готовности (доступность БД и RPC отражены в ответе)
