# 官方要求与交付清单

已于 2026 年 9 月 12 日读取课题原文 revision 10 和交付指引 revision 5。本文只对应课题二，课题四应另发邮件。群内最新通知优先于指引；本次没有读取参赛群，不能确认存在或不存在补充通知。

## 提交规则

- 截止：2026 年 9 月 13 日 00:00，北京时间，即 9 月 12 日结束前发送成功。
- 收件人：`wxg_prc_cpg@tencent.com`。
- 标题：`【犀牛鸟实战提交】【2】【刘德宝 / mingri31164】`。
- 正文：姓名、GitHub ID、选题编号和题目、成果类型、成果链接、Tag 与完整 Commit SHA、运行说明、完成情况和已知问题。
- 代码材料：仓库链接、最终 Tag、完整 SHA、README、运行/测试命令、`submission.yaml`。源码 ZIP 只能作为补充。
- 顺序：先固定最终代码并创建 annotated Tag，再提交 YAML。元数据提交会比 Tag 多一个 commit，属于指引认可的正常状态。不得移动、删除或覆盖最终 Tag。
- 指引未要求五分钟视频、配音或指定 PPT 模板，也未给出附件大小上限。随包答辩稿和录屏作为补充证据。

## 课题二任务

| 官方任务 | 对应实现 | 边界 |
| --- | --- | --- |
| 统一终端会话接口，屏蔽执行后端差异 | CommandTerminal、SessionBoundManager、Docker TTY、E2B/Cube envd 适配 | 当前主线不开放宿主本地进程执行；文件和终端运行在会话沙箱内 |
| 页面交互终端、实时输出、中断与缩放 | WorkbenchTerminal、WebSocket 首帧 Ticket、stdin、resize、interrupt | 每条命令独立，不替代上游可重连 Shell |
| 文件浏览、上传下载、重命名、删除 | 固定产物根目录、目录描述符操作、不覆盖原子发布 | Cube 文件能力关闭；删除只允许文件或空目录 |
| 按类型展示演示文稿、网页、表格 | artifact kind、逐页 PPTX、opaque iframe、表格预览 | PDF/DOCX/音视频降级下载，不在题目列出的三类验收内 |
| 演示文稿 Skill 到生成、预览、下载 | presentation-builder、Agent read_file/shell_exec、ArtifactCollector | 原始 Agent 产物及界面证据分别注明生成与拍摄版本 |

## 五项验收

| 官方验收 | 证据与核对口径 |
| --- | --- |
| 至少两种后端终端可用 | Docker 与 Kubernetes E2B 兼容后端实际 PTY/API 验证；原文未指定 E2B Cloud 与 Cube 必须同时验收 |
| 两租户同时开终端，只能看到自身进程和文件 | verify-concurrent-tenants.mjs 同时保持两个真实命令，检查 PID namespace、进程标记、文件、跨租户 HTTP、每条命令审计 |
| 服务端拒绝产物目录外路径 | 直接请求绝对路径、`..`、中间 symlink 的 API 与 helper 测试，不只依赖前端拒绝 |
| 三类产物在线查看，网页无法获取主站数据和登录态 | 实际逐页 PPTX、表格和 HTML 截图；opaque iframe 无同源/脚本权限并启用禁网 CSP。浏览器执行器限制、历史证据版本均保留 |
| CPU、内存、时长超限自动终止且命令有审计 | 命令及后代的超限、清理与保存期内分页审计通过；不是整沙箱会话累计配额。若按后者解释，该条未全部满足，见 task-review.md |

完整验收是对上述具体行为的判断，不等价于生产容量、MicroVM 隔离、所有云供应商兼容或任意恶意 Shell 的不可绕过限制。

## 交付动作

仓库与提交版本、远端 Tag、YAML 位置、可直接发送的邮件稿及精简附件包由最终材料索引列出。邮件准备完成不等于发送成功；本次不自动发信，发送后需保留已发送邮件。

来源：[课题原文](https://bytedance.larkoffice.com/docx/AgPRdXbYcoSh1zxwJlnc5CvWnab)；[交付指引](https://bytedance.larkoffice.com/wiki/IDYpwdHyCiCU0ukdtDHcZeBanEe)。两份原文的读取版本和内容摘要哈希见 `official-requirements.json`，受权限约束的原文不随公开包分发。
