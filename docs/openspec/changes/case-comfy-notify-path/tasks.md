## 1. Case 快照与注入

- [x] 1.1 实现 CaseSnapshot：按 Task 加载 Case，深拷贝 `bindings.workflow`，从 InputPrefix 读取 staged 文本/数值/布尔/图片元数据
- [x] 1.2 实现按 `bindings.inputs` 注入；必填缺失、绑定缺失、空 workflow、节点不存在时在 Submit 前失败并上报 status
- [x] 1.3 为文本注入与映射失败编写单元测试（先红后绿）

## 2. Comfy 客户端与图片

- [x] 2.1 扩展 Client（Mock + HTTP）以支持图片上传/引用；Mock 返回稳定假名并保持 `comfy_mock` 开关
- [x] 2.2 注入路径对 `image` 字段走上传后再写节点；补充 Mock/HTTP 选型与上传相关测试
- [x] 2.3 在 `main` 将 Worker.Workflows 接到 CaseSnapshot（Tasks/Cases/Blob），移除默认空 StaticWorkflows 成功主路径

## 3. 协议种子与校验

- [x] 3.1 增加或更新「文本 + 图片」混合 Case 种子（含 bindings）；校验拒绝「图片字段仅文本」
- [x] 3.2 提供可关 mock 验收的最小真实 workflow fixture（或文档化仓库内 graph），与种子绑定一致

## 4. TG 采集与投递

- [x] 4.1 RegisterHandlers 支持 Photo（及常见 image Document）；当前字段为 image 时写入 Blob Draft
- [x] 4.2 当前字段为 number/boolean 时解析标量；image 字段收到普通文本时提示而非误写入
- [x] 4.3 SendPhoto 使用无路径分隔符的安全文件名；补适配器测试

## 5. 端到端验收

- [ ] 5.1 更新/新增集成冒烟：ConfirmRun → 注入可观测 → Mock 成功 → notify（含图片输入场景）
- [ ] 5.2 验证 `comfy_mock: true` 主路径与 `comfy_mock: false` 在可达 Comfy（或 HTTP fixture）下 Submit/Wait 行为符合规格
- [ ] 5.3 全量相关包 `go test` 通过，并确认 Mock 与真实路径代码同步演进
