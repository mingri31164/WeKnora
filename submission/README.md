# 课题四：知识网络与引导式学习

本提交在 WeKnora Wiki 中提供个人主题推荐、来源测验、掌握度与复习状态，并集成学习 Agent。可运行代码、设计说明、验证数据和复现方法分别列在下方。

## 版本

| 项目 | 内容 |
| --- | --- |
| 提交署名 / GitHub ID | `mingri31164` |
| 最终代码 Tag | [`rhino-2026-final-4`](https://github.com/mingri31164/WeKnora/tree/rhino-2026-final-4) |
| 完整代码 SHA | `600565c2f4947bb2a16b0dd025d7a9918d32ac7a` |
| 上游 PR | [Tencent/WeKnora #3145](https://github.com/Tencent/WeKnora/pull/3145) |
| 上游基线 | `081851c6b8802225ae968ba67d8de444e258c67b` |
| 材料分支 | `submission/guided-learning` |

署名使用已确认的 GitHub ID，未推定真实姓名。材料分支在最终代码后追加本目录及根目录 `submission.yaml`，不会移动最终 Tag，也不进入上游功能 PR。

## 材料导航

- [设计说明](design.md)：节点定义、信号、BKT、推荐、来源校验与隐私。
- [运行与验证](reproduction.md)：环境前提、自动化命令、首次夹具与真实模型验证。
- [验证结果](verification.md)：本次执行结果、跳过项、评估基线及限制。
- [功能截图](screenshots.md)：桌面、移动端、Graph、测验、Agent 卡片和删除。
- [邮件正文](email.txt)：发送前核对署名及链接，未代发。
- [提交元数据](../submission.yaml)：按交付指引记录最终代码版本。
- [机器可读证据](evidence/verification.json)、[离线评估输出](evidence/offline-evaluation.txt)、[证据校验值](evidence-sha256.json)。

## 完成范围

| 课题要求 | 实现与验证 |
| --- | --- |
| 节点与掌握度设计 | Wiki 页面 ID 为节点，访问/文档使用与 BKT 掌握度独立；设计说明解释数据来源和算法限制 |
| 可运行原型 | Wiki 面板、Graph 状态、三题测验、来源反馈、学习 Agent 卡片 |
| 有效性验证 | 固定合成数据的推荐与预测基线；真实模型/API、桌面与移动端验证 |
| 按租户隔离的个人画像 | 显式 Web 身份和租户作用域；跨租户读取/答题拒绝 |
| 查看、导出、删除 | 学习概览和状态、JSON 导出、知识库/工作区清除；已验证非空删除和对另一账号无影响 |

## 已知限制

仅支持已登录 Web 用户及当前工作区自有 Wiki 知识库，不覆盖 API Key、IM、Embed、共享 Agent 和跨租户共享库。BKT 未做真人校准，合成留出预测未优于固定先验。同模型盲答检查一致性，不能代替人工审核。推荐不构造先修课程，也不宣称最优学习路径。

真实验收使用一次性合成账号和来源，不包含真实学习者实验。证据中明确区分真实调用、模拟错误与跳过用例；无密钥、账号文件或原始服务日志。
