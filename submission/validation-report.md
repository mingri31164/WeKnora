# 测试与验收报告

最终代码：`59c4e0b2ed3375bf69510339a19abc882f6ffe34`；Tag：`rhino-2026-final-2-v2`。

## 最终版本复核

本轮固定代码后重新运行下列检查。Go 数量按叶子测试计数，不含父级容器；环境 skip 单列。原始日志仅在本机保留，随包 JSON 记录命令、退出码、时间与日志 SHA-256。

| 检查 | 结果 | 测试数 / 跳过 |
| --- | --- | --- |
| go-full | PASS | 8134 / 33 |
| go-vet | PASS | - / 0 |
| go-race | PASS | 2156 / 0 |
| go-race-workbench-service | PASS | 33 / 0 |
| go-incremental-lint | PASS | - / 0 |
| go-integration-lint | PASS | - / 0 |
| python-files | PASS | 20 / 0 |
| python-probe | PASS | 5 / 0 |
| python-runner | PASS | 3 / 0 |
| python-builder | PASS | 27 / 0 |
| frontend-tests | PASS | - / 0 |
| frontend-types | PASS | - / 0 |
| frontend-build | PASS | - / 0 |
| offline-artifacts | PASS | - / 0 |
| go-real | PASS | 45 / 0 |
| go-real-race | PASS | 45 / 0 |
| 前端用例 | PASS | 928 / 0 |
| API / WebSocket | PASS | 24 / 0 |
| 两租户同时运行 | PASS | Docker、E2B 兼容后端各一组 |

终端测试包含提前输出、stdin、resize、中断、CPU/进程地址空间/进程树合计 RSS/时长限制及后代清理。新增多进程内存测试使单个子进程低于地址空间限额、合计 RSS 超过预算，核对自动终止与 memory_limit。采样 RSS 会重复计算共享页，也可能有瞬时超调，不替代容器硬配额。

同时运行测试先等待两个租户命令都进入 READY，再检查各自进程标记、不同 PID namespace、文件可见性、跨租户拒绝和服务端路径边界；逐条核对 accepted 与完成审计。测试会话执行删除清理。首次测试驱动把创建会话的 201 误写成 200，修正断言后通过，最初产生的空会话已单独核对并删除。

## 浏览器与 Agent 证据

以下录屏及 Agent 生成来自此前已验证的 `2a20930742945ab368da9c906f5b1da714cebac0`，未伪装为最终提交重新录制。本次合入官方 main 462999ec，解决依赖、语言包及产物抽屉冲突；预览兼容通过组件和类型回归复核，旧录屏不能替代新代码的浏览器实测。

实际浏览器记录为 26 PASS、4 次初始菜单驱动时序/依赖失败及对应纠正 PASS、2 skip。快捷键段警告、错误和 pageerror 均为 0；全程有一次非空目录删除被拒绝的 409。执行器报告 GPU/proc 访问限制，因此不记干净退出，也未继续绕过限制启动浏览器。

三类产物有界面预览证据：PPTX 按页、HTML 的 opaque iframe、CSV 表格。历史四页 Skill 稿截图的拍摄版本与产物来源均单列；新 Support Review 任务产物未重新录页面。

真实 Agent 从六行合成 CSV 生成 PPTX、HTML、XLSX、CSV，SSE 耗时 36.913 秒。独立核对 Platform=450/417、Search=300/280、Total=750/697，百分比与逐行数据一致，PPTX 四页，输入未变化且没有误收为回答产物。

一次可选 openpyxl 导入返回1；CSV/XLSX 未包含提示词要求的 synthetic 文件内标注。原产物未人工修正，不宣称所有工具调用成功或提示词完全遵循。离线校验器可复跑原产物与五个篡改反例。

## 版本与限制

`prior-validation-summary.json` 保留 `2a209307` 的测试与失败重跑记录，包括过长/软链接 TMPDIR 的问题。最终测试使用短物理临时目录，未修改安全断言。

`performance-samples.json` 同样来自 `2a209307`，仅每项五个串行样本；不将先前测量当作新代码的性能结论。`validation-5d3fa917.json` 保留上一版最终 Tag 的验证。旧 Tag rhino-2026-final-2 未移动，当前改用 v2 Tag。

实测后端为 Docker Engine 和 Kubernetes E2B 兼容接口。原生 Cube/E2B Cloud、MicroVM 隔离与多用户容量未验收。Cube 文件能力关闭；当前兼容环境模板目录返回404，首次配置依赖预配置。

官方五项验收对应关系见 00-official-delivery-checklist.md。任务中的宿主本地进程后端没有开放，遵循当前主线只在会话沙箱执行的边界；如评审要求恢复该后端，需要另行明确安全与部署条件。

本轮另跑真实 Agent 并复核合并失败和依赖告警，详情见 [合并收尾复核](closeout-verification.md)。
