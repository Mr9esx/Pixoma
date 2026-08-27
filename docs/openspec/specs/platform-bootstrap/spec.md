# platform-bootstrap Specification

## Purpose
TBD - created by archiving change pixoma-guided-deploy. Update Purpose after archive.
## Requirements
### Requirement: 零配置控制面可启动
系统 MUST 提供控制面入口二进制（产品名 `pixoma`），在无用户预先编写业务 YAML 的情况下能够启动；启动成功后 MUST 在日志中输出可打开的后台地址，以及仅用于首启的默认管理员账号与密码（或一次性初始密码）。

#### Scenario: 空目录首次启动
- **WHEN** 用户在空数据目录启动 `pixoma` 且未提供业务库配置
- **THEN** 进程就绪，日志含后台 URL 与默认管理员凭证信息

### Requirement: 引导态与未初始化门闩
系统 MUST 在本机维护引导态（bootstrap），至少包含：是否已完成初始化、管理员凭证、业务库连接信息、向导进度。未完成初始化时，管理 HTTP MUST 仅允许登录与向导相关接口；MUST NOT 对外提供完整业务管理与任务主路径能力。

#### Scenario: 未初始化拒绝业务 API
- **WHEN** 平台尚未完成初始化向导，客户端调用非向导/非登录的管理业务 API
- **THEN** 请求被拒绝或重定向到初始化流程，且不执行该业务副作用

### Requirement: 默认凭证仅首启可见
系统 MUST 仅在未初始化（或尚未完成首次改密）时于日志打印默认/初始管理员密码；初始化完成且管理员已改密后，MUST NOT 再次在常规启动日志中打印明文密码。

#### Scenario: 初始化后不再打印明文密码
- **WHEN** 初始化完成且管理员已修改密码后再次启动控制面
- **THEN** 启动日志不包含该管理员明文密码

### Requirement: 启动初始化默认 Topic
控制面启动 MUST 在业务库就绪后幂等确保默认 Topic（`default`）存在；已存在则跳过。该初始化 MUST NOT 阻塞未完成初始化向导的路径。

#### Scenario: 空库启动创建默认 Topic
- **WHEN** 业务库中无任何 Topic 记录且控制面启动
- **THEN** 创建 `default` Topic；再次启动不重复创建

### Requirement: 系统账号是登录主体
平台初始化完成后，管理后台登录、改密、会话与授权 MUST 以系统账号（`console_users`）为准；引导态 bootstrap 仅承担未初始化阶段的门闩与向导时序，MUST NOT 承担初始化后的登录认证主体。

#### Scenario: 初始化后登录走系统账号
- **WHEN** 平台已完成初始化，管理员登录后台
- **THEN** 认证校验系统账号表，成功则签发标识系统账号 id 与角色的会话

#### Scenario: 引导态仍可引导登录
- **WHEN** 平台尚未初始化或处于向导阶段
- **THEN** 仍走引导态登录与向导接口，完成初始化后再以系统账号登录

