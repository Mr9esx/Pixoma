## Context

参见 `proposal.md` 的 Why。当前仓库已有 Catalog（SQLite）、ConfirmRun 物化输入到 Blob、Orchestrator 同步内存队列、Comfy `NewClient(Mock|HTTP)`，但 `main` 将 Actuator `Workflows` 设为 `StaticWorkflows{}`，不读 Case、不注入；TG 仅注册文本 Update。规格层面一期已写明注入与 image 类型，实现未闭环。

## Goals / Non-Goals

**Goals（设计层）**

- 单一执行路径：Task → Case 快照 → 加载 staged 输入 → 注入 → Client.Submit/Wait → Blob 产物 → status
- 文本与图片注入共用绑定模型；图片在客户端侧完成 upload（真实）或等价引用（Mock）
- TG 按 Case 字段类型分流文本/图片 Update

**Non-Goals（设计层）**

- Session/Task 持久化与进程崩溃恢复
- 视频采集与复杂多图相册编辑
- 更换队列实现或引入分布式调度

## Decisions

1. **CaseSnapshot 提供方替代 StaticWorkflows 默认空图**  
   - 选择：`CaseSnapshot{Tasks, Cases, Blob}` 实现 `WorkflowForTask`，深拷贝 `bindings.workflow` 后注入。  
   - 备选：ConfirmRun 时把图塞进 StaticWorkflows map — 拒绝，因与 Case 版本/重试脱节。

2. **图片注入：真实 Comfy 先 Upload 再写节点字段；Mock 接受占位文件名**  
   - 选择：在执行面注入阶段，对 `image` 类型调用 Client 扩展或旁路 `UploadImage`（若现有 Client 无此能力则最小扩展接口）；Mock 返回稳定假名。  
   - 备选：只把本地路径塞进图 — 真实 Comfy 不可用。

3. **TG 媒体：RegisterHandlers 增加 Photo/Document 匹配，按当前字段类型提交 Blob Draft**  
   - 选择：Adapter 查当前 Case 字段类型；`image` 才收下媒体。  
   - 标量：`number`/`boolean` 在 Adapter 解析后再 `SubmitInput`。

4. **种子 Case**  
   - 至少一个「文本+图片」混合 Case（mock 可用 Stub 图 + 绑定）；另保留可切真实 Comfy 的 workflow fixture（或文档化的最小 API graph），由 `comfy_mock: false` 验收。

5. **发图文件名**  
   - `filepath.Base(ref.Key)`（或等价），避免 Telegram 拒收。

## Risks / Trade-offs

- [真实 Comfy 节点/模型环境差异] → 提供可配置种子 graph + mock 默认；真实路径以「绑定注入 + HTTP 提交成功」为验收，不强绑定特定 checkpoint 名  
- [图片 Document MIME 多样] → 先支持 Photo + 常见 image/* Document；其余明确拒绝  
- [Client 接口扩展] → 保持 Mock 同步演进（`comfy_mock` 规则）

## Migration Plan

- 部署后默认 `comfy_mock: true` 行为向好（注入后仍出 mock 图）；关 mock 需本机 Comfy 与种子 graph  
- 回滚：可临时切回 StaticWorkflows，但本 change 验收不依赖该回滚路径

## Open Questions

- 真实 Comfy 种子 graph 以仓库内 fixture 为准，还是运行时从外部路径加载？（实现时优先仓库内最小 fixture，外部路径可作为后续增强）
