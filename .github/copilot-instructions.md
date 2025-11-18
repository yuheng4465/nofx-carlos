<!-- .github/copilot-instructions.md - 为 AI 编码助手提供此仓库的快速上手指令 -->
# NOFX — Copilot 指南（简明）

下面的说明旨在帮助 AI 编码代理（例如 Copilot / 自动化助理）快速在此仓库中变更、调试或实现新功能。

- **项目主页与入口**: `main.go` 是后端入口，前端在 `web/`（Vite + React）。后台主要模块位于 `nofx/` 目录（例如 `api/`、`decision/`、`manager/`、`market/`、`trader/` 等）。
- **核心设计要点**:
  - 配置以数据库为准：`config.json` 在启动时会被 `main.go` 同步到 SQLite（见 `loadConfigFile`、`syncConfigToDatabase`）。因此优先通过 Web 界面或数据库修改运行时配置，而不是直接编辑 `config.json`（仍可用于启动时的默认值）。
  - AI 与提示词：AI 客户端实现位于 `mcp/client.go`，提示模板在 `prompts/` 目录并通过 API `GET /api/prompt-templates` 暴露（见 `api/server.go`）。决策引擎位于 `decision/engine.go`，它负责将市场数据、历史回溯与提示拼接成给 AI 的 System/User prompt。
  - 决策记录：所有 AI 决策会写入 `decision_logs/{trader_id}/{timestamp}.json`，并由 `logger/decision_logger.go` 处理，重要用于回放与分析。
  - 加密/密钥：加密服务从 `secrets/rsa_key` 加载（见 `crypto.NewCryptoService`），敏感字段在数据库中以加密方式存储并通过 `api/CryptoHandler` 提供加/解密接口。

- **快速运行（推荐 Docker）**:
  - 一键（推荐）: `./start.sh start --build` 或 `docker compose up -d --build`。
  - 查看日志 / 控制: `./start.sh logs`, `./start.sh status`, `./start.sh stop`。

- **手动开发（本地）**:
  - 后端（可能需本机安装 TA-Lib）:
    - 安装依赖后：`go test ./...`（运行测试）
    - 本地运行：`go run main.go` 或 `go build -o nofx && ./nofx`（可传入可选数据库路径参数）
  - 前端：
    - `cd web && npm install`（或 `pnpm` / `yarn`）
    - `cd web && npm run dev`（开发服务器，默认 `http://localhost:3000`）

- **常用环境变量与重要文件**:
  - `NOFX_BACKEND_PORT` — 后端端口覆盖（`main.go` 中读取）
  - `JWT_SECRET` — JWT 密钥可通过环境变量覆盖（或从 SQLite 中读取）
  - `NOFX_ADMIN_PASSWORD` — 在 `admin_mode` 下用于初始化管理员密码（见 README 的 Admin Mode）
  - 重要文件/目录：`config.json`（模板 `config.json.example`）、`secrets/`（密钥）、`decision_logs/`、`prompts/`、`web/`。

- **开发约定 / 模式**:
  - 配置优先级：Env > DB(system_config) > `config.json` 默认值。
  - 多 Trader 管理：实例化 `manager.NewTraderManager()`，通过 `traderManager.LoadTradersFromDatabase(database)` 加载并管理 trader 生命周期（见 `main.go`）。
  - 市场数据与流：`market.NewWSMonitor(...)` 启动 websocket 监控，市场数据通过 `market.Get(symbol)` 提供给 `decision` 层（见 `decision/engine.go` 的 `fetchMarketDataForContext`）。
  - 风险控制 & 决策：AI 生成建议后，决策会经过本地 `Decision` 校验/风险验证（实现分散在 `decision/`、`manager/`、`trader/` 中），勿直接将未验证的 AI 输出下发到交易所。
  - 插件点：`hook/` 提供 Hook 插件机制（例如 IP 查询、Webhook 等），使用 `hook.HookExec` 的方式触发。

- **修改提示词 / 模型**:
  - 编辑模板：`prompts/` 下的文件是常用模板；也可通过前端 UI 编辑并保存到 DB。
  - AI 客户端：`mcp/client.go` 包含对 DeepSeek/Qwen/custom API 的封装，添加新模型时遵循该客户端的抽象。

- **API 快速示例**:
  - 健康检查: `curl http://localhost:8080/api/health`
  - 创建 Trader: `POST /api/traders`（参考 `api/server.go` 的 `CreateTraderRequest` 结构）

- **测试注意事项**:
  - 单元/集成测试: 使用 `go test ./...`。一些测试或模块依赖系统级库（如 TA-Lib），在 CI 中通常使用 Docker 镜像来保证一致性。

- **安全与审计提醒**:
  - 密钥与私钥应放在 `secrets/` 并受限访问。不要将生产密钥写入 `config.json` 或提交到仓库。
  - 修改与扩展与交易相关的代码（`trader/`、`manager/`、`decision/`）时，务必在沙盒或纸盘（paper）账户上做充分回测。

如果你需要我把这个内容合并到现有 `.github/copilot-instructions.md`（如果仓库已有旧版本），或把某一部分扩展为更详细的“修改指南 / 贡献者流程”，告诉我需要补充的方向或目标受众（如：新开发者 / 代码审计 / 自动化测试）。
