---
comet_change: case-comfy-notify-path
role: technical-design
canonical_spec: openspec
---

# case-comfy-notify-path 技术设计

## 1. 目标

打通：开始 Case → 收集文本与用户图片 → 执行工作流 → 取回产物图 → 完成任务 → 通知用户。

- `comfy_mock: true`：同一条代码路径，假上传、假产物图，端到端必须成功。
- `comfy_mock: false`：连真实 Comfy（例如局域网 `192.168.31.131:8188`），用预置 Case 对应的工作流跑通。

非目标：Session/Task 落盘后重启恢复、视频输入、多实例调度、管理后台。

## 2. 预置 Case 与工作流

| 项 | 值 |
|---|---|
| 工作流文件 | `klein9b-edit.api.json`（从远程 Comfy 拷贝进仓库，建议路径 `configs/workflows/`） |
| 用户图片 | 节点 `10`，类型 LoadImage，字段 `image` |
| 文本 | 节点 `20`，类型 CLIPTextEncode，字段 `text` |
| 产物图 | 节点 `60` SaveImage |

Case 配置（启动时写入目录库）：

- 输入顺序建议：先用户图片（必填），再文本（必填）；可选再加数字种子等。
- `bindings.workflow`：上述 API 工作流 JSON 的内容（启动时从文件读入或直接写在 Case JSON 里）。
- `bindings.inputs`：`reference` → `10`/`image`；`prompt` → `20`/`text`。
- TG 展示名称与描述可中性；不必与工作流文件名一致。

## 3. 架构与数据流

```
TG 更新
  → 适配器（按当前字段类型：文本 / 用户图片 / 数字…）
  → 应用层确认
       校验 → 输入写入本地对象存储 → 创建任务 → 发布「任务已创建」
  → 调度
       派发到本机执行器（命令里带任务 ID 与输入前缀）
  → 执行器
       查任务 → 查 Case → 复制工作流 JSON
       → 读已存输入
       → 用户图片：读本地对象存储 → Comfy UploadImage → 得到 filename
       → 按绑定把文本与 filename 写入节点
       → Submit → Wait → 产物写入本地对象存储 → 上报状态
  → 调度收敛终态 → 通知
  → TG 发送产物图
```

### 3.1 执行时如何得到工作流（已确认做法 A）

新增「按任务准备可提交工作流」的组件（实现名可用现有接口 `CaseSnapshotProvider` / 结构体名在实现时与包内风格一致，对外行为如下）：

依赖：任务仓库、Case 仓库、本地对象存储。

步骤：

1. 用任务 ID 取任务（得到 CaseID、输入前缀）。
2. 取 Case；若 `bindings.workflow` 为空 → 失败状态，不调用 Comfy。
3. 深拷贝工作流 JSON。
4. 按 Case 的输入定义，从 `输入前缀/字段名` 读已存文件（`.txt` / `.num` / `.bool` / `.blob.json`）。
5. 对每个有值的输入：查 `bindings.inputs`；无绑定 → 失败。
6. 类型为用户图片：从 Blob 读字节 → 调用客户端 `UploadImage` → 把返回的 filename 写入节点字段。
7. 类型为文本/数字/布尔：直接写入节点字段（字段路径默认写到该节点的 `inputs.<field_path>`）。
8. 返回写好的工作流，供 Submit。

`main` 接线：执行器不再使用「未知任务就返回空工作流」作为成功主路径。

### 3.2 Comfy 客户端

在现有 `Submit` / `Wait` 上增加：

```text
UploadImage(ctx, filename, mime, data) → (remoteFilename, error)
```

- HTTP：`POST /upload/image`（multipart），解析返回的 `name`（或 Comfy 实际返回字段）。
- Mock：返回稳定假文件名（例如 `mock-upload.png`），不访问网络。
- 选型仍走现有 `NewClient` + `comfy_mock` / `COMFY_MOCK`。

### 3.3 TG

- 注册 Photo（以及常见 `image/*` 的 Document）处理。
- 仅当当前待填字段类型为 `image` 时：下载 → 写入本地对象存储 → 以 Blob 草稿提交。
- 当前字段为 `image` 时若收到普通文本：提示发图片，不要当文本写入。
- 当前字段为 `number` / `boolean`：解析后再提交草稿。
- 发送产物图：上传文件名只用路径最后一段（避免 key 里的 `/` 导致发送失败）。

## 4. 错误与边界

| 情况 | 行为 |
|---|---|
| 空工作流 / 缺必填绑定 / 节点不存在 | 上报失败，不 Submit |
| 必填用户图片未存到对象存储 | 上报失败，不 Submit |
| Upload 或 Submit / Wait 失败 | 上报失败，带可诊断信息 |
| 真实 Comfy 缺模型文件 | Comfy 返回错误 → 失败状态透出 |
| 可选字段已跳过 | 不写入对应节点 |

## 5. 测试策略

1. 单元：复制工作流后文本写入正确节点；用户图片路径调用 Upload 后写入 filename；缺绑定/空工作流失败。
2. 单元：Mock / HTTP 的 UploadImage；`NewClient` 选型不变。
3. TG：收照片写入 Blob；数字解析；发图文件名无 `/`。
4. 联调：Mock 下「文本 + 用户图片 → 通知收图」。
5. 真机：关 Mock，base URL 指向可达 Comfy，用该预置 Case 至少完成一次 Submit/Wait 成功（或记录环境不可达时的失败信息，不假装成功）。

## 6. 实现顺序（与 tasks.md 对齐）

1. 按任务准备工作流 + 写入输入（含失败用例）
2. UploadImage（Mock + HTTP）并接入写入路径；`main` 接线
3. 预置 Case + 工作流文件入库
4. TG 收用户图片与标量解析、发图文件名
5. 联调与回归测试

## 7. Spec Patch

无。行为已由 open 阶段 delta 覆盖；实现中若发现验收场景缺口再补。
