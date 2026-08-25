# Pixoma Voice Profile

> 写任何 Pixoma 面向用户的中文文案(alert / 空状态 / 错误页 / 按钮 / 步骤说明 / README)前读这一页。
> Source of truth。`pixoma-voice` skill 里的速查表是从这里摘的稳定子集,可以扩但不删。

## Voice Summary

同组工程师在 Slack 上交代"先做这件事、再做那件事"。技术细节默认对方听得懂,不懂的才解释;行动指令永远是动作起头,不用"请"和"麻烦"。

## Core Personality Traits

- **Peer, not instructor**: 默认对面会 grep,会读错误,会用 zsh。
- **Specific, not general**: 文件名 / 命令 / 路径 / 字段名都写出来,不用"之类的"糊弄。
- **Action-direct**: 按钮和动作是动词原形或短动词("保存" / "下一步" / "勾 Mock"),不用"保存一下"。
- **Concise, not curt**: 短句不是冷,是省时间。每段 1–3 句,信息密度高。
- **Neutral, not performative**: 不卖萌、不感叹、不"温馨提示",失败就是"操作失败"。

## Tone Spectrum

| Dimension | Position | Notes |
|---|---|---|
| Formal ↔ Casual | 略 casual 但克制 | 用「」括产品名、混英文技术词,不用"亲 / 您 / 小伙伴" |
| Serious ↔ Playful | 严肃,零卖萌 | 不用感叹号、不用 emoji、不用"啊 / 呀"语气词 |
| Reserved ↔ Bold | 略 bold | 关键判断直接给(默认路径**不需要 Redis**),不绕弯子 |
| Simple ↔ Sophisticated | 简单词装复杂事 | 短句 + 技术词,不用"进行 / 完成 / 实施"等泛动词 |
| Warm ↔ Direct | 直接,不远不近 | 不用"请",但也不冷到命令式 |

## Vocabulary

### Use

- 后台(不用"管理界面")
- 本机 / 远程(干净二分,不加"计算机")
- 勾 / 点(短动词,不用"勾选 / 点击")
- …(中文省略号,不是 ASCII `...`)
- 「」 括产品名 / 功能名
- `` `code` `` 括命令 / 文件 / 字段
- 中 表进行(登录中…,不用"正在登录")
- 你(不用"您")

### Avoid

- "请 / 烦请 / 麻烦"(hand-holding)
- "前往"(远距动词,说明书腔)
- "查找关键字 / 检索关键字"(名词化包装)
- "搜 / 那行"(口语过头,平台用"搜索")
- "进行 / 完成 / 实施 / 执行" 等无信息动词
- "温馨提示 / 哎呀" 等卖萌
- "首先 / 其次 / 最后"(直接列)
- 感叹号 / emoji
- "您 / 用户 / 小伙伴"(远距)

**Jargon level:** Heavy — DSN / ComfyUI / SQLite / S3 / TOS / Edge / Mock / claim / blob 都不解释,直接用。

**Profanity:** Never.

## Rhythm & Structure

**Sentences:** 4–12 字为主。常用"事实句 + 后果 / 动作句"两段式:

- "任务、用户这些记录存在哪。本机先用 SQLite 文件即可。"
- "Comfy 和后台是否在同一台电脑。远程 GPU 必须用对象存储。"

**Paragraphs:** 1–3 句。**绝对不超 4 句**。

**Openings:** 常用 bold 主张或直接动作。"默认路径**不需要 Redis**。" / "先登录、改密,再走初始化向导。"

**Closing:** 永远落到一个可执行动作。"保存密码" / "下一步" / "勾 Mock" / "按部署命令跑 agent"。

**去重原则:** desc / placeholder 出现之前,先扫一眼同一 UI 区域的 title / Field label / button 文案,已经写过的信息不重复。例:Field label "再输一遍" 已经说过"输两次",desc 就不再写"输两遍确认"。如果砍完只剩"复述 title 加一个新信息点"(如"新的"区分默认密码),就接受它短——1 句 11 字的 desc 比 2 句 18 字的废话干净。

**Formatting:**

- 关键概念 bold
- 产品 / 功能名 「」
- 命令 / 路径 / 字段 `` `code` ``
- 列表用 `:` 起头 + 短项
- 二分(本机 / 远程、登录 / 未登录)用 `|` 对齐

## POV & Address

**First person:** 极少"我"。主体是"系统"或隐含主语。
**Reader address:** **你**。从不"用户 / 您 / 小伙伴"。
**Relationship stance:** **Peer engineer to peer engineer**。假定你会读 log / grep / `cat /var/log/...`。

### 例外

Admin 后台在描述 **bot 端真人(Telegram 用户)** 时,可用"用户"作为实体名,不视为远距代词。例:

- `用户输入`(workflow 节点的输入类型)
- `用户名` / `用户列表`(admin 在管理 bot 用户)
- `通知用户` / `对用户生效` / `用户看到`(admin 在向 bot 端描述效果)

判定标准:"用户"在句中是**实体名词**(描述一类对象),不是**称呼**(对面那个人)。如果是称呼,改用"你"。

## Example Phrases

### On-brand

- "默认路径**不需要 Redis**,也不再让你选 allinone / split。"
- "日志会打出后台地址和**仅首次**的默认管理员账密。"
- "本机先用 SQLite 文件即可。"
- "远程 GPU 必须用对象存储。"
- "新密码(至少 8 位)"
- "登录中…"
- "保存后重启 `pixoma` 才按新配置装配。"

### Off-brand

- "请您先登录管理界面,然后再进行密码修改操作。" — "请" + "管理界面" + "进行" 三踩
- "温馨提示:首次启动时系统会自动为您生成一个默认密码哦~" — 卖萌 + 远距代词 + 拖尾虚词
- "找不到?试试看哦~" — 感叹 + 卖萌 + 口语
- "恭喜!工作流已成功删除!" — 感叹号 + 恭喜(成功本身就够)

## Do's and Don'ts

**DO:**

- 短句、动作起头、写具体路径 / 命令
- 失败直接说"X 失败",成功直接说"已 X"
- 二分(本机 / 远程、登录 / 未登录)用对仗
- 关键词用 `` `code` `` / `bold` / 「」 三件套之一

**DON'T:**

- 用"请 / 麻烦 / 烦请"开头
- 用"进行 / 完成 / 实施"等无信息动词
- 用"您 / 用户 / 小伙伴"
- 用感叹号、emoji、卖萌语气词
- 把 4 句话塞进一段

## Changelog

- 2026-08-25: 初版,从 `web/admin/src` 现成中文 + README 第一段抽出。
