---
name: pixoma-voice
description: 写 Pixoma 任何面向用户的中文文案(alert / 空状态 / 错误页 / 按钮 / 步骤 / README)前使用。先读 docs/voice-profile.md 拿到完整规则,再按其中的 Use/Avoid 词表和 Rhythm 规则出稿,不要凭直觉。
---

# Pixoma Voice

写任何 Pixoma 面向用户的中文文案前,**先读一遍** [`docs/voice-profile.md`](../../../../docs/voice-profile.md)。

读完后按 profile 的 Use/Avoid 词表、Rhythm 规则、Example Phrases 出稿。如果用户给的草稿和 profile 冲突,先按 profile 改一版给用户看,再解释为什么改。

## 速查(摘自语料,稳定子集,docs 可扩不删)

**Use:** 后台 / 本机 / 远程 / 勾 / 点 / 你 / 「」 / `code` / … / 中(进行中)
**Avoid:** 请 / 烦请 / 麻烦 / 前往 / 查找关键字 / 搜(平台场景)/ 那行 / 进行 / 完成(作泛动词) / 实施 / 温馨提示 / 首先 其次 最后 / 感叹号 / emoji / 您 / 用户 / 小伙伴

**句长:** 4–12 字为主,段 1–3 句,**不超 4 句**。
**闭合:** 永远落到可执行动作("保存密码" / "下一步" / "勾 Mock" / "按部署命令跑 agent")。
**POV:** 你,从不"用户 / 您 / 小伙伴"。
**关系:** Peer engineer to peer engineer。

## 注意

- 不凭"我觉着更顺"出稿,profile 是基准。
- profile 没覆盖的场景,按 profile 精神延伸,不另起风格。
- 用户明确说"按我这句话原文用"时,profile 退让,直接用用户原文。
- profile 文件本身是 source of truth,这里的速查表是稳定子集,docs 可扩但不删;只有当速查跟 profile 严重背离时才重写本文件。
