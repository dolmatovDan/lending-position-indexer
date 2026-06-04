# Protocols Specification

## Purpose

Абстракция протокола (`protocols.Reader`) и адаптеры Aave v3 и Euler v2: единый интерфейс сбора leg-строк позиций, расширяемость на новые протоколы и EVM-сети без изменения слоёв manager/repository/api.

## Requirements

### Requirement: Абстракция протокола и расширяемость

Система ДОЛЖНА предоставлять единый интерфейс протокола (`protocols.Reader`), позволяющий
добавлять новые протоколы и новые EVM-сети без изменения слоёв manager/repository/api.
`Positions(ctx, wallets, block)` возвращает `[]models.Position` — leg-строки (по активу),
**уже с разрешёнными on-chain ценами** (ценообразование протокол-специфично и живёт внутри
адаптера). Согласно ARCH зависимости передаются явно через конструкторы — глобальный реестр
и `init()` не используются.

#### Scenario: Подключение протокола через явную инъекцию
- GIVEN определён интерфейс `protocols.Reader` (`Name()`, `Positions(ctx, wallets, block) []models.Position`)
- WHEN новый client-адаптер реализует этот интерфейс и добавляется в срез
      `[]protocols.Reader` в `setup_managers.go`
- THEN `positions.Manager` начинает собирать по нему позиции без изменений в самом
      manager/repository/api (open/closed через конструкторную инъекцию, без глобалов)

#### Scenario: Изоляция и тестируемость адаптера
- GIVEN каждый протокол реализован отдельным пакетом `internal/clients/<proto>/` с
      интерфейсом и stub
- WHEN пишутся тесты manager или адаптера
- THEN адаптер подменяется stub'ом без обращения к реальному RPC

#### Scenario: Добавление новой EVM-сети через конфиг
- GIVEN адаптеры используют адреса контрактов и RPC из конфигурации сети (chainID, адреса)
- WHEN добавляется конфигурация новой EVM-сети
- THEN существующие адаптеры могут работать в новой сети без изменения их кода

---

### Requirement: Адаптер Aave v3

Адаптер Aave v3 ДОЛЖЕН собирать позиции пользователей из контрактов Aave v3 в Ethereum
mainnet, включая health factor.

#### Scenario: Сбор leg-строк по резервам пользователя
- GIVEN кошелёк с залогом и/или долгом в Aave v3
- WHEN адаптер читает `getReservesList`, по каждому резерву `getUserReserveData`
      (баланс aToken, variable+stable debt), `getUserAccountData` (HF) и цену из `AaveOracle`
- THEN формируются leg-строки: по резерву с ненулевым aToken-балансом — строка
      `side=collateral`, с ненулевым долгом — строка `side=debt`; каждая со своим
      `token`/`amount`/`price`; на всех строках один `health_factor` аккаунта (масштаб 1e18)

#### Scenario: Пропуск пустых позиций
- GIVEN кошелёк без активности в Aave v3
- WHEN адаптер собирает данные
- THEN по этому кошельку leg-строки Aave не создаются (нулевые балансы отфильтрованы)

#### Scenario: Идентификатор рынка и scope Aave
- GIVEN собраны leg-строки Aave для кошелька
- WHEN заполняется `market_id`
- THEN он идентифицирует пул Aave (адрес Pool), общий для всех строк кошелька — это scope
      аккаунта, по которому строки группируются и разделяют `health_factor`

---

### Requirement: Адаптер Euler v2

Адаптер Euler v2 ДОЛЖЕН обнаруживать позиции пользователя on-chain через Ethereum Vault
Connector (EVC) и собирать данные из соответствующих vault'ов, включая health factor.

#### Scenario: On-chain дискавери позиций через EVC
- GIVEN кошелёк (и его суб-аккаунты) в Euler v2
- WHEN адаптер вызывает `EVC.getControllers(account)` (vault долга) и
      `EVC.getCollaterals(account)` (vault'ы залога)
- THEN список релевантных vault'ов определяется без внешнего списка рынков; если контроллер
      отсутствует — позиция без долга (только залог)

#### Scenario: Сбор leg-строк из vault'ов
- GIVEN обнаруженные collateral- и controller-vault'ы аккаунта
- WHEN адаптер читает `convertToAssets(balanceOf(account))` (залог в активах),
      `debtOf(account)` (долг в активах), `asset()` каждого vault'а и цену из oracle-роутера
      vault'а
- THEN формируются leg-строки: по каждому collateral-vault — строка `side=collateral`, по
      controller-vault — строка `side=debt`; каждая со своим `token`/`amount`/`price`.
      `market_id` всех строок scope — адрес controller-vault (для чистого залога без
      контроллера — каждый collateral-vault сам себе scope, `market_id` = адрес vault)

#### Scenario: Расчёт health factor через accountLiquidity
- GIVEN контроллер-vault аккаунта с долгом
- WHEN адаптер вызывает `accountLiquidity(account, false)` → `(collateralValue, liabilityValue)`
      в unit of account vault'а
- THEN `health_factor = collateralValue / liabilityValue`; при `liabilityValue == 0` позиция
      трактуется как «без долга»

#### Scenario: Обход суб-аккаунтов
- GIVEN EVC использует суб-аккаунты (адрес владельца XOR id, до 256 на владельца)
- WHEN адаптер сканирует суб-аккаунты до глубины `EULER_SUBACCOUNTS_DEPTH` (по умолчанию 1 —
      только основной аккаунт)
- THEN позиции собираются по всем непустым суб-аккаунтам в пределах глубины; глубина —
      конфигурируемый компромисс полноты и нагрузки на RPC
