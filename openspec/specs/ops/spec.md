# Ops Specification

## Purpose

Эксплуатация сервиса: структура запуска по ARCH, docker-compose, конфигурация через переменные окружения, структурное логирование, тесты и README.

## Requirements

### Requirement: Структура запуска по ARCH

`cmd/positions` ДОЛЖЕН следовать layered-композиции ARCH: `Config` struct в `config.go`,
разнесённая инициализация по `setup_core/clients/repositories/managers/processors.go`, и
строгая последовательность в `main.go`.

#### Scenario: Порядок инициализации в main.go
- GIVEN сервис стартует
- WHEN выполняется `main.go`
- THEN шаги идут строго: signal-context → загрузка `Config` → `setupCore` (logger) →
      `setupClients` (ethnode, протокол-адаптеры aave/euler с ценами) → миграции БД → `setupRepositories` →
      `setupManagers` (инжект `[]protocols.Reader`) → `setupProcessors` (blocks) →
      запуск HTTP-сервера → блокировка на `ctx.Done()` → graceful shutdown в обратном порядке

#### Scenario: Явная конструкторная инъекция
- GIVEN слои собираются в `setup_*.go`
- WHEN создаются managers/clients/repositories
- THEN зависимости передаются через конструкторы явно, без глобалов и `init()`

---

### Requirement: Запуск через docker-compose

Сервис ДОЛЖЕН запускаться через docker-compose вместе с PostgreSQL.

#### Scenario: Поднятие стека одной командой
- GIVEN заполненный `.env` (RPC, кошельки, БД)
- WHEN выполняется `docker compose up`
- THEN поднимаются сервис `app` и `postgres`; приложение применяет миграции, подключается к
      RPC и начинает обрабатывать новые блоки

#### Scenario: Зависимость от готовности БД
- GIVEN сервис `app` зависит от `postgres`
- WHEN postgres ещё не готов
- THEN `app` дожидается готовности БД (healthcheck/`depends_on`) перед началом работы

---

### Requirement: Конфигурация через переменные окружения

Поведение сервиса ДОЛЖНО настраиваться переменными окружения.

#### Scenario: Обязательные переменные
- GIVEN запуск сервиса
- WHEN не заданы обязательные переменные (`RPC_*`, `WALLETS`, `DATABASE_URL`)
- THEN сервис завершается с понятной ошибкой конфигурации до начала индексации

#### Scenario: Список кошельков
- GIVEN `WALLETS` задан списком адресов
- WHEN сервис стартует
- THEN он индексирует ровно эти кошельки (позиции внутри протоколов дискаверятся on-chain)

#### Scenario: Обработка с лагом подтверждений
- GIVEN задан `CONFIRMATIONS=k`
- WHEN приходит новый блок высотой H
- THEN сервис обрабатывает блок `H-k` (по умолчанию `k=0` — голову), снижая риск реоргов

---

### Requirement: Структурное логирование

Сервис ДОЛЖЕН вести структурные логи ключевых событий.

#### Scenario: Логи обработки блока
- GIVEN сервис обрабатывает блок N
- WHEN цикл завершён
- THEN в лог пишется структурная запись с номером блока, числом собранных позиций и
      длительностью; ошибки RPC/БД логируются с контекстом

---

### Requirement: Тесты

Кодовая база ДОЛЖНА содержать тесты, проверяющие ключевую логику.

#### Scenario: Unit-тесты расчётов
- GIVEN реализованы расчёты HF (Aave/Euler), конвертация активов, масштабирование цен
- WHEN запускается `go test ./...`
- THEN тесты на эталонных значениях со stub'ами клиентов (без реального RPC) проходят

#### Scenario: Тесты repository и API
- GIVEN repository Postgres и REST-handlers
- WHEN выполняются их тесты
- THEN проверяются upsert/выборки и ответы эндпоинтов

#### Scenario: Опциональный integration-тест
- GIVEN integration-тест с реальным RPC за build-tag
- WHEN не задан RPC/тег
- THEN тест пропускается по умолчанию и не ломает CI

---

### Requirement: README

Репозиторий ДОЛЖЕН содержать README с описанием запуска и расширения.

#### Scenario: Содержание README
- GIVEN новый разработчик открывает проект
- WHEN он читает README
- THEN он находит: обзор архитектуры, запуск через docker-compose, список env-переменных,
      инструкции по добавлению нового протокола и новой EVM-сети, известные ограничения
      (напр. глубина обхода суб-аккаунтов Euler, лаг `CONFIRMATIONS`)
