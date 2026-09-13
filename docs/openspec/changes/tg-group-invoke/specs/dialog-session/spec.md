## MODIFIED Requirements

### Requirement: 每聊天最多一个进行中的填表会话
系统 MUST 为每个会话作用域保证同一时刻最多一个非终态 Dialog Session。私聊的作用域 MUST 为该 chat；group / supergroup 的作用域 MUST 为该 chat 加上操作者用户 id。浏览菜单与 Case 列表 MUST NOT 创建 Session。仅 `StartCase` 成功后进入 `collecting` 并上锁。

#### Scenario: 浏览菜单不创建会话
- **WHEN** 用户仅打开菜单或 Case 列表
- **THEN** 系统不创建 Dialog Session，用户仍可自由导航

#### Scenario: StartCase 上锁
- **WHEN** 用户对某 active Case 执行 StartCase 且当前无进行中 Session
- **THEN** 系统创建 `collecting` 会话并禁止再次 StartCase，直至退出或提交

#### Scenario: 同群两人互不抢锁
- **WHEN** 群成员 A 已在 collecting，群成员 B 在同一群 StartCase
- **THEN** 系统为 B 创建独立会话，A 的会话保持不变

### Requirement: 确认执行后结束填表会话
ConfirmRun 成功创建 Task 后，系统 MUST 结束填表 Session（`submitted` 或清除 active），从而解锁。进行中的执行 Task MUST NOT 阻止用户开启新的填表 Session。

#### Scenario: 提交后解锁
- **WHEN** ConfirmRun 成功
- **THEN** 该会话作用域不再处于填表锁，同一作用域可再次 StartCase

#### Scenario: 生成中可开新 Case
- **WHEN** 用户已有 running Task 且无填表 Session
- **THEN** StartCase 被允许
