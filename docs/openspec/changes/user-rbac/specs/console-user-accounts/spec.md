## ADDED Requirements

### Requirement: 控制台系统账号模型
系统 MUST 提供面向后台控制台登录的系统账号模型，与渠道终端用户目录相区分；每个系统账号 MUST 包含：账号名（唯一）、邮箱（可选唯一）、昵称、头像、启用状态、角色；密码 MUST 以 bcrypt 哈希存储，不得明文或可逆明文落库。

#### Scenario: 新建系统账号
- **WHEN** 管理员通过用户管理新建一个系统账号并填写账号名、邮箱、昵称、密码
- **THEN** 系统创建该账号并分配默认角色，密码以 bcrypt 哈希持久化

#### Scenario: 账号名唯一冲突
- **WHEN** 新建或改名的账号名与既有系统账号冲突
- **THEN** 系统拒绝并返回可诊断错误，不写入该账号

### Requirement: bootstrap 初始 admin 迁移为系统账号
系统初始化时 MUST 将引导态初始 `admin` 账号迁移为系统账号（角色 `Admin`），保留其用户名与凭据；后续管理后台登录与鉴权 MUST 以系统账号为准，而非 bootstrap 单账号。

#### Scenario: 升级后首次登录
- **WHEN** 平台升级后管理员用既有 `admin` 账号与密码登录
- **THEN** 登录通过且该账号在系统账号表中为 `Admin` 角色

#### Scenario: 迁移幂等
- **WHEN** 已迁移过初始 admin 后再次启动迁移过程
- **THEN** 不重复创建账号，迁移幂等

### Requirement: 系统账号与渠道终端用户隔离
系统账号 MUST NOT 出现在渠道终端用户目录（`users` 资源页）中，二者数据与语义分离。

#### Scenario: 目录互不包含
- **WHEN** 管理员查看渠道终端用户目录或系统账号列表
- **THEN** 两类账号互不混淆，各按其来源与语义展示

### Requirement: 渠道终端用户表更名保留数据
渠道终端用户内部表 MUST 由 `users`/`user_external_identities` 更名为 `channel_users`/`channel_user_external_identities`，更名 MUST 幂等且保留既有数据；系统账号表 MUST 为 `console_users`，三表语义不混。

#### Scenario: 既有库更名保留数据
- **WHEN** 含既有渠道用户数据的 `users` 表被迁移至 `channel_users`
- **THEN** 旧表被重命名（非清空/重建），既有渠道用户记录完整保留

#### Scenario: 新装库幂等
- **WHEN** 在全新数据库上执行初始化/迁移
- **THEN** 直接按新表名建表，更名步骤为无害幂等跳过
