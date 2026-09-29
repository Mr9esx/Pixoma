# Admin 图标清单

适用范围：`web/admin` 的后台导航、创作 Studio、页面空状态、对象引用和常用操作。图标组件来自 `lucide-react`；Pixoma 品牌使用 `web/admin/src/assets/logo.tsx` 中的 `Logo`。

## 使用规则

- 同一业务对象在导航、选择按钮、空状态和引用标记中使用同一个图标。图标只表达对象或操作，状态仍由文字和状态组件表达。
- 对象图标与操作图标按用途选择。例如，`Workflow` 表示工作流，`Plus` 表示新建工作流。
- 图标位于 shadcn/ui 按钮、菜单项或侧栏项内时，由组件控制尺寸；只有独立展示的图标按所在区域设置尺寸。
- 只有图标的按钮必须提供可访问名称，例如 `aria-label` 或可见的 tooltip。
- 增加或修改图标时，检查同一业务对象在后台导航、Studio、空状态和引用标记中的用法，并更新本清单。

## 后台主导航

主导航的图标配置位于 `web/admin/src/config/menu.ts`。`cases` 是代码中的页面标识，页面名称为「工作流」。

- `LayoutDashboard`：仪表盘入口。
- `Zap`：快速新建入口。
- `WandSparkles`：创作 Studio 入口。
- `Workflow`：工作流入口、工作流页面空状态及配置关系图中的工作流节点。
- `Radio`：消息平台入口、消息平台页面空状态及配置关系图中的平台节点。
- `Waypoints`：任务队列入口、任务队列页面空状态及配置关系图中的任务队列节点。
- `Server`：计算节点入口、计算节点页面空状态及配置关系图中的计算节点。
- `ListTodo`：任务入口。
- `MessagesSquare`：平台会话入口。
- `Users`：用户入口。
- `Settings`：平台设置入口。

## 创作 Studio

- `Workflow`：聊天输入框的工作流选择、工作流引用、Agent 工作流设置及流程展示。
- `Boxes`：聊天输入框中的资产选择与资产引用。
- `Library`：资产库入口、资产库空状态和保存到资产库的操作。
- `Sparkles`：技能选择、技能引用和技能空状态。
- `BrainCircuit`：模型空状态。
- `Cable`：MCP 连接器空状态。
- `MessageCircle`：Studio 对话入口和资产树中的对话节点。
- `Folder`：Studio 项目、项目选择和资产树中的分类节点。
- `MessageSquarePlus`：新建 Studio 对话。
- `MoreHorizontal`：Studio 项目与对话的更多操作。
- `Settings2`：AI 设置入口和资产树组织方式设置。
- `Star`：资产评分及资产树中的评分节点；`Tag`：资产树中的标签节点。
- `ImagePlus`：在聊天输入框添加图片。
- `ShieldCheck`：聊天输入框中的 Agent 操作权限。
- `TriangleAlert`：完全访问权限确认弹窗中的风险提示。
- `ListTree`：打开制作流程面板。

## 常用操作与内容类型

- `Plus`：新建项目或增加内容；`Upload`：上传文件；`Download`：下载文件。
- `Pencil`：编辑内容；`Trash2`：删除内容；`Search`：搜索内容。
- `ArrowLeft`：返回上一级；`ChevronDown`：展开选择项和项目；`ChevronRight`：折叠项目；`Check`：标记当前选择；`X`：关闭或移除内容。
- `ImageIcon`：图片资产类型；`FileText`：文档资产类型。
- `Package`：计算节点中的 ComfyUI 版本信息。
