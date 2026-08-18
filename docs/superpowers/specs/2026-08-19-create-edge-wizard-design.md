---
role: technical-design
status: accepted
---

# 添加计算节点三步向导技术设计

## 1. 目标与边界

把「计算节点」页的新建弹窗从单表单改为三步向导：

1. 填写节点信息（现有创建表单）
2. 部署节点（展示 AGENT_TOKEN 与部署命令，前端轮询节点在线 / Comfy 是否就绪）
3. 添加完成（结果页）

页头「新建」按钮与空态「新建节点」按钮统一改为「添加节点」。

**非目标：**

- 不改后端 API：复用 `POST /api/v1/edges` 创建、`GET /api/v1/edges/presence` 查在线/Comfy 状态
- 不改编辑表单（`EdgeForm` 的 edit 模式原样保留）
- 不做就绪后自动跳转、不自动关闭弹窗
- 不新增节点部署/探测的后端能力

## 2. 已确认决定

| 项 | 选择 |
|---|---|
| 按钮文案 | 页头与空态统一「添加节点」（en: Add node） |
| 第一步 | 复用现有创建表单，提交即创建节点，成功后进入第二步 |
| 第二步 | 展示 TOKEN（掩码可复制）与部署命令（可复制）+ Comfy 提示 |
| 轮询 | 每 3 秒查该节点 presence；`edge_online && comfy_running` 即就绪 |
| 就绪后 | 停在第二步等用户点「继续」，不自动跳转 |
| 跳过 | 「跳过」直接进第三步，同时停止轮询 |
| 第三步 | 只显示添加完成结果，不显示「已跳过部署」状态 |
| 第三步操作 | 「查看节点」（关闭并跳详情）/「完成」（关闭） |

## 3. 组件

### `CreateEdgeWizard`（新增）

三步状态机 `form → deploy → done`，持有已创建节点。

- `form`：渲染 `EdgeForm mode="create"`；`onSaved` 拿到含 `agent_token` 的节点后切到 `deploy`
- `deploy`：渲染 `DeployCredentials` + `PresenceTags`（实时状态）+ 轮询 query（`refetchInterval: 3000`，`enabled` 只在 deploy 步生效）；「继续」在就绪前禁用，「跳过」随时可用
- `done`：结果页，节点名称 + 「查看节点」/「完成」

弹窗关闭或组件卸载时，轮询 query 随之卸载，不残留后台轮询。

### `DeployCredentials`（从 `AgentCredentials` 抽出）

只读展示 AGENT_TOKEN（掩码、可复制）与部署命令（可复制）+ Comfy 提示。`AgentCredentials` 改为复用该组件，并把「重新生成」按钮作为 `tokenActions` 插槽注入，保持详情面板布局不变。

## 4. 文案

| key | zh | en |
|---|---|---|
| `edges.createNode` | 添加节点 | Add node |
| `edges.deployHeading` | 部署节点 | Deploy node |
| `edges.deploySkip` | 跳过 | Skip |
| `edges.deployContinue` | 继续 | Continue |
| `edges.deployWaitHint` | 在目标机器运行部署命令后，等待节点上线、Comfy 启动即可继续 | Run the deploy command on the target machine; continue once the node is online and Comfy is running |
| `edges.createDoneHeading` | 添加完成 | Node added |
| `edges.createDoneDesc` | 节点已添加 | The node has been added |
| `edges.createDoneView` | 查看节点 | View node |
| `edges.createDoneClose` | 完成 | Done |

弹窗标题随步骤变化：添加节点 → 部署节点 → 添加完成。

## 5. 测试

沿用仓库 contract-test 风格（读取源文件与 i18n JSON 断言结构/文案）：

- 按钮与空态用 `edges.createNode`，zh/en 文案为「添加节点 / Add node」
- 向导含三步结构，第二步渲染 `DeployCredentials` 与 `PresenceTags`，presence 轮询 `refetchInterval: 3000`
- 「继续」在就绪前禁用，「跳过」直接切到结果页
- i18n 新 key 齐全
- 既有 `agent-credentials` 断言改为新结构（`DeployCredentials` 承担 token/命令展示）

先写失败测试，再实现，最后跑全部相关测试。

## 6. 验收

- 新建按钮与空态按钮显示「添加节点」
- 提交节点信息后进入部署步骤，显示可复制的 TOKEN 与部署命令
- 部署机器后，3 秒内状态 Tag 变为「节点在线」「Comfy运行中」，「继续」可用
- 点「跳过」直接进结果页；点「查看节点」跳详情、点「完成」关弹窗
- 关闭弹窗后不再轮询
