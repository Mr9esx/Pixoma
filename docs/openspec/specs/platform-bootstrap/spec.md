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

