# 运行与验证

代码以 `submission.yaml` 中的最终 Tag 和完整 Commit SHA 为准。Tag 指向功能代码；提交材料分支在其后追加材料，不移动最终 Tag。

```bash
git clone --branch rhino-2026-final-4 --single-branch https://github.com/mingri31164/WeKnora.git WeKnora-topic4
cd WeKnora-topic4
git rev-parse HEAD
# 应为 600565c2f4947bb2a16b0dd025d7a9918d32ac7a
```

克隆到独立目录，不要放在另一个仓库的 `.runtime/` 内。当前测试路径校验不支持嵌套 `.runtime` 根目录。

## 环境

- Linux 或 macOS，Go 1.26+、CGO 编译工具、Node.js 24、Python 3.11+。
- PostgreSQL、Redis 及仓库支持的检索后端。引导式学习使用 Wiki 页面链接，不要求 Neo4j。
- 一个支持 OpenAI 兼容对话和 Tool Calling 的模型，用于文档处理、出题和 Agent。模型调用会产生费用。

部署沿用仓库 `README_CN.md` 和 `website-docs/06-development/01-dev-guide.md`。使用专用测试实例，打开注册，设置 `WEKNORA_AUTH_DEFAULT_TENANT_MODE=create_personal`、`WEKNORA_TENANT_ENABLE_CROSS_TENANT_ACCESS=false`，确认迁移和异步任务正常。前端代理须指向同一后端。

## 离线测试

在代码仓库根目录执行，以下测试不需要模型密钥：

```bash
go test -p 8 ./...
go vet ./...
go test -race ./internal/application/service/learning ./internal/application/repository ./internal/types
go test ./internal/application/service/learning -run '^TestLearningOfflineEvaluation$' -count=1 -v
python3 -B -m unittest discover -s scripts -p 'test_guided_learning_fixtures.py' -v
npm --prefix frontend ci
npm --prefix frontend test
npm --prefix frontend run type-check
npm --prefix frontend run build
npm --prefix frontend run test:e2e:config
```

仓储测试默认使用临时 SQLite 数据库。验证 PostgreSQL 时，给 `LEARNING_TEST_POSTGRES_DSN` 和 `WEKNORA_WIKI_RENAME_TEST_DSN` 提供专用测试数据库连接后，重跑仓储 `-race` 测试。测试创建、删除独立临时 schema，不应配置生产数据库。

## 首次初始化

按实际服务配置填写端口。脚本只允许 HTTP loopback 地址；运行目录须位于当前 checkout 的 `.runtime/` 中。

```bash
export GL_RUNTIME_DIR=.runtime/guided-learning
export GL_API_BASE=http://127.0.0.1:8080
export GL_FRONTEND_BASE=http://127.0.0.1:5173
python3 -B scripts/seed-guided-learning.py seed --consent-create-fixtures --model-env /path/to/model.env
python3 -B scripts/seed-guided-learning.py opt-in --user learner_a
python3 -B scripts/seed-guided-learning.py configure-agent --user learner_a
python3 -B scripts/test-guided-learning-api.py inspect
```

`model.env` 必须为当前用户所有、权限 `0600`，包含 `MODEL_NAME`、`MODEL_BASE_URL`、`MODEL_API_KEY`；Base URL 使用 HTTPS。也可在当前进程环境中提供这些变量并省略 `--model-env`。不要把密钥写进 Git、邮件、命令截图或提交包。

初始化通过公开 API 创建两个一次性用户、两个独立租户、各自知识库和模型，以及六份合成来源文档和六个 Wiki 页面。等待真实解析完成且分块引用有效后才发布清单。重复初始化复用资源；不要删除账号文件后假装是首次执行。

## 真实 API 与浏览器

```bash
python3 -B scripts/test-guided-learning-api.py run --consent-learner-b
npm --prefix frontend exec -- playwright install chromium
GL_RUN_QUIZ=1 GL_CREATE_CARD=1 GL_RUN_LEARNER_B_CHECKS=1 \
  npm --prefix frontend run test:e2e:guided-learning
GL_RUN_QUIZ=1 GL_QUIZ_ACTOR=learner_b GL_ALLOW_LEARNER_B_CLEAR=1 \
  node --test --test-name-pattern='APPROVAL: real quiz|APPROVAL: learner_b' \
  frontend/e2e/guided-learning.spec.mjs
```

API 验收实际调用模型、提交一题答案、核对 BKT 和幂等性、运行学习 Agent，最后清理 `learner_b` 并确认 `learner_a` 未受影响。`observed` 仅记录事实，不计为通过；模型失败会保留失败结果。

第一组浏览器测试覆盖桌面与移动端、真实出题、三题提交、来源、刷新恢复、实际 Agent 卡片及跨租户读取。第二组在 `learner_b` 上先产生真实作答，再通过界面删除，并比较 `learner_a` 的导出摘要。

模拟错误恢复单独标记：

```bash
GL_RUN_MOCK_ERRORS=1 node --test --test-name-pattern='MOCK ERROR ONLY' \
  frontend/e2e/guided-learning.spec.mjs
```

部分开发机禁止 Chromium 子进程的 OOM 优先级操作，可设置 `GL_SINGLE_PROCESS=1`。这是测试环境兼容开关，不能据此声称验证了浏览器进程隔离。

不要并行运行 seed、API 和浏览器验收。输出位于所选运行目录的 `evidence/`。账号和原始运行文件按私有数据保管，提交材料只含检查过的汇总及脱敏截图。

## 人工体验

1. 登录一次性 `learner_a`，打开其 Wiki 知识库，在「引导学习」面板开启个人学习记录。
2. 打开推荐主题并检查来源，点击「练习当前页面」，等待三题生成。
3. 提交答案，查看正误、解释和引文；刷新后确认答案仍在。
4. 打开 Graph，比较「已接触」与「学习中」标记。阅读本身不会成为「已掌握」。
5. 选择内置「引导式学习」Agent，询问下一主题并准备练习，展开测验卡片。
6. 从「学习数据」导出记录。仅在一次性账号上验证清空，确认不再返回已删除的题目和答案。
