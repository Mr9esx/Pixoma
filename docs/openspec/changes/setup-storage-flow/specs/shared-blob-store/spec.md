## ADDED Requirements

### Requirement: 对象存储连通性检查
系统 MUST 提供对象存储连通性检查能力，供 Setup 向导在保存前校验配置：`localfs` MUST 校验目录存在且可写；`s3` 与 `tos` MUST 使用对应 SDK 校验 endpoint/region/bucket/密钥（HeadBucket），无需读写业务对象；检查失败 MUST 给出可诊断错误。

#### Scenario: S3/TOS 连通性检查通过
- **WHEN** 向导提交 S3 或 TOS 配置，endpoint/region/bucket/密钥正确且 bucket 可达
- **THEN** 连通性检查通过，向导可继续

#### Scenario: 密钥错误被识别
- **WHEN** 向导提交的 S3/TOS 密钥无效（如 InvalidAccessKeyId、SignatureDoesNotMatch）
- **THEN** 连通性检查失败，错误信息可诊断

#### Scenario: bucket 不存在被识别
- **WHEN** 向导提交的 S3/TOS bucket 不存在（如 NoSuchBucket、404）
- **THEN** 连通性检查失败，错误信息可诊断

#### Scenario: localfs 目录校验
- **WHEN** 向导提交 localfs 配置
- **THEN** 目录存在且可写时检查通过；目录不可写或无法创建时检查失败并给出可诊断错误

### Requirement: bucket 不存在时可确认自动创建
`s3` 与 `tos` 的连通性检查发现 bucket 不存在时，系统 MUST 返回可识别的 `bucket_not_found` 结果并附带 bucket 名；向导 MUST 提示用户是否代为创建；用户确认后系统 MUST 创建该 bucket（S3 `CreateBucket`、TOS `CreateBucketV2`）并再次校验，创建成功后才标记连接正常；用户未确认时 MUST NOT 创建。

#### Scenario: bucket 不存在时提示创建
- **WHEN** 向导测试连通性且 bucket 不存在（404 / NoSuchBucket）
- **THEN** 向导提示「bucket 不存在，是否帮你创建」，未确认前不创建

#### Scenario: 确认后自动创建 bucket
- **WHEN** 用户确认创建 bucket 且账号具备建桶权限
- **THEN** 系统创建 bucket 并复检通过，向导显示连接正常

#### Scenario: 建桶权限不足
- **WHEN** 用户确认创建 bucket 但账号无建桶权限
- **THEN** 创建失败并给出可诊断错误，不标记连接正常
