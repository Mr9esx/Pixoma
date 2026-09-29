# Pixoma 资产库新版与 Eagle 能力矩阵

基准日期：2026-09-28。Eagle 一栏依据官方帮助中心与产品公告，包含已发布的 Eagle 4.0 插件能力。Pixoma 一栏依据[资产库 PRD](2026-09-28-studio-asset-library-prd.md)，表示新版目标；当前交互演示只展示其中一部分能力。

| 能力模块 | Eagle 已有能力 | Pixoma 新版目标 | 能力判断 |
| --- | --- | --- | --- |
| 资产收集 | 本机导入、浏览器扩展批量保存、网页收藏、截图、剪贴板、链接导入及自动导入。[官方资料](https://eagle.cool/support/desktop/collect)、[浏览器扩展](https://eagle.cool/extensions) | Studio 对话生成、手动创建及上传的文件自动进入所属项目资产库；在资产库上传时使用左侧树当前选中的项目，并记录来源与固定版本。 | Eagle 提供多种外部素材收集入口；Pixoma 自动收录 Studio 创作产物。 |
| 项目与对话 | 以资源库组织素材，可在资源库内使用文件夹、标签和智能文件夹。[官方资料](https://en.eagle.cool/support/desktop/organize) | 左侧常驻项目资产树，设置按钮可切换「项目 → 资产 / 对话 / 分类 / 格式 / 评分 / 标签」；点击分组筛选右侧资产。来源对话与资产库共用资产数据，跨项目使用时为目标项目建立条目。 | Pixoma 的项目、对话和资产元数据参与文件树组织与创作复用。 |
| 文件类型 | 支持图片、视频、音频、字体、网页、文档及部分 3D 文件，部分格式通过插件扩展。[官方资料](https://www.eagle.cool/blog/post/eagle4)、[预览说明](https://en.eagle.cool/support/desktop/browse) | 图片、视频、音频、文档及其他文件；提取可用的尺寸、格式和时间信息。 | Eagle 的格式覆盖与专用预览更广；Pixoma 的文件信息需以实际可读取的格式逐项验收。 |
| 分类组织 | 文件夹、子文件夹、智能文件夹、批量创建及移动文件夹。[官方资料](https://en.eagle.cool/support/desktop/organize) | 项目分类置于资产区顶部，支持未分类、多级分类、创建、改名、移动和删除；每个项目资产条目归属一个分类。 | Pixoma 的分类用于项目内筛选；Eagle 还提供智能文件夹。 |
| 标签与评分 | 标签管理、星级评分、批量改名、自动标签及 AI Action 批量整理。[组织说明](https://en.eagle.cool/support/desktop/organize)、[AI Action](https://eagle.cool/blog/post/eagle-plugin-ai-action) | 项目资产条目独立保存名称、标签和 0–5 星评分；支持分类改名、批量标签和评分。 | 两者均覆盖日常编辑；Eagle 还提供批量改名和 AI 自动整理。 |
| 基本信息 | 检查器展示文件属性；插件可补充 EXIF 与媒体信息。[Eagle 4.0 说明](https://www.eagle.cool/blog/post/eagle4) | 显示评分、尺寸、文件大小、核验后的格式、添加日期、创建日期、修改日期；格式可在下拉选择中筛选。 | Pixoma 将格式同时用于详情、筛选及文件树分组。 |
| 图片与视频调色盘 | 用户提供的截图展示横向排列的代表色。 | 按固定文件版本分析图片可见像素和视频多个画面，详情展示最多 10 个色块；分析失败可重试。 | Pixoma 的调色盘与不可变版本关联，跨项目引用可保持结果一致。 |
| 搜索与筛选 | 关键词、保存筛选、标签、颜色、形状、尺寸、比例、类型、评分、日期、大小等条件；高级搜索支持 OR 与括号。[搜索说明](https://en.eagle.cool/support/desktop/search)、[高级搜索](https://en.eagle.cool/blog/post/eagle4-build12) | 当前项目内按来源对话搜索名称和标签；资产类型与格式分别使用下拉选择，分类采用顶部切换栏，并组合尺寸、大小、日期、评分、重复文件及排序条件。 | Pixoma 提供项目内来源对话筛选及格式分组；Eagle 的保存筛选与高级搜索条件更多。 |
| 智能搜索 | AI Search 插件支持以图找图与文本信息的自然语言搜索，并在本机运行。[AI Search 公告](https://en.eagle.cool/blog/post/eagle-plugin-ai-search) | 本版以结构化条件和关键词检索为主。 | Eagle 已提供视觉相似检索；Pixoma 新版尚无对应目标。 |
| 重复文件 | 扫描内容相同的文件，并查找视觉相似的重复图片。[官方资料](https://en.eagle.cool/support/desktop/organize) | 使用 SHA-256 提示内容完全相同的版本，可筛选并查看同组资产。 | Pixoma 覆盖完全相同文件；视觉相似图片的识别仍有差距。 |
| 预览与查看 | 图片详情、焦点放大、视频与 GIF 播放、旋转和裁剪；支持其他应用打开。[官方资料](https://en.eagle.cool/support/desktop/browse) | 大尺寸详情弹窗，左侧资产预览约占三分之二；可放大至占满弹窗，右侧查看信息并编辑名称、标签、评分。 | Pixoma 覆盖资产查看和管理编辑；媒体专用查看与图像处理能力较少。 |
| 批量管理 | 批量改名、分类、标签、文件夹调整及 Action 操作。[官方资料](https://en.eagle.cool/support/desktop/organize) | 在项目范围内批量移动分类、修改标签和评分、添加到其他项目、归档、恢复及导出；逐项报告失败结果。 | Pixoma 支持项目之间复用资产；Eagle 还提供批量改名和可保存的自动化动作。 |
| 导出与交换 | 原文件导出、Eaglepack、CSV，以及带格式、尺寸、质量和预设的批量格式转换。[导出说明](https://en.eagle.cool/support/desktop/import-and-export)、[自定义导出](https://en.eagle.cool/blog/post/eagle4-build12) | 单项导出固定版本原文件；批量导出 ZIP，重复名称增加序号。 | Pixoma 保证创作引用版本的一致性；格式转换、导出预设及元数据交换尚未纳入目标。 |
| 来源与版本 | 文件替换时可保留分类、标签和评分。[Eagle 更新说明](https://en.eagle.cool/blog/post/eagle4-build12) | 记录来源类型、Session、操作和固定 AssetVersion；每个项目条目可独立更新所指版本，历史 Session 引用保持原版本。 | Pixoma 保留创作来源与各项目使用的固定版本。 |
| 创作复用 | 可复制文件、在其他应用中打开，并通过插件扩展处理能力。[Eagle 4.0 说明](https://www.eagle.cool/blog/post/eagle4) | 在 Studio 中将固定版本用于新 Session 或已有 Session；跨项目使用时复用文件内容，并建立目标项目资产条目。 | Pixoma 将项目资产库与创作对话连接；Eagle 支持跨应用素材使用。 |
| 扩展与自动化 | 插件系统、Web API v2、Eagle Skill / MCP、AI Action。[Web API v2](https://eagle.cool/blog/post/eagle4-build21)、[Skill / MCP](https://www.eagle.cool/blog/post/eagle-plugin-mcp-skill)、[AI Action](https://eagle.cool/blog/post/eagle-plugin-ai-action) | 资产库与 Studio、Session 和 Flow 的内部流程连接；PRD 未规定面向第三方的资产库 API 或插件机制。 | Eagle 对外部工具开放的能力更完整；Pixoma 新版聚焦内部创作链路。 |

## 产品范围建议

1. 对话产物自动进入所属项目资产库；Session 资产面板与项目资产库共用资产数据。项目移动、项目删除及跨项目引用需要共同验收。
2. 保持来源、固定版本与 Session 引用的一致性。这些能力贯穿保存、查看、再次创作和导出，需要在同一版产品中共同验收。
3. 项目内分类、标签、评分、改名、组合筛选和批量操作需要接入服务端持久化；UI 演示目前只呈现交互。
4. 浏览器采集、智能文件夹、视觉相似搜索、格式转换和开放 API 可分别讨论；本次 PRD 未将它们列入新版范围。
