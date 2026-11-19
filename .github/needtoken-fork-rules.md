# NeedToken Fork 开发与上游同步规则（needtoken-storefront）

本仓是上游 `dujiao-next/dujiao-next` 的生产 Fork。在遵守上游 `AGENTS.md` 的前提下，所有本地开发与 AI 协助必须同时遵守以下规则。

## 1. Remote 与分支规范

- `origin`：`NeedTokenAi/needtoken-storefront`（日常推送与提 PR）。
- `upstream`：`dujiao-next/dujiao-next`（只读 `fetch`，本地必须 `git remote set-url --push upstream no_push` 禁止误推）。
- `main` 是唯一发布基线，**禁止本地直接 commit 或 push 到 `main`**；合入 `main` 会触发 Cloudflare Workers Builds 自动构建与发布。
- 日常开发一律在 `dev`（单线迭代）或 `feat/<topic>` / `fix/<topic>`（并行需求）进行，提 PR 合并进 `main`。
- 每次 PR 或上游同步合入 `main` 后，及时将 `origin/main` 合并/对齐回本地 `dev`，避免长期分叉。

## 2. 减少上游冲突的定制隔离原则

- 自研功能优先采用**独立新增文件 / 独立店面模板 / 独立中间件或目录**（例如 `cloudflare/`、`frontend/user/src/templates/` 等），对上游核心文件只做最小侵入式入口修改。
- **严禁**顺手格式化、重命名或重构与当前任务无关的上游代码。
- 不要直接大改上游 `AGENTS.md`；NeedToken 专属规则统一维护在本文件（`.github/needtoken-fork-rules.md`）及主仓 `needtoken/AGENTS.md`。

## 3. 上游同步与自动合并红线

- 自动同步工作流：`.github/workflows/sync-upstream.yml`（默认通道 `stable-tags`，仅跟 `vMAJOR.MINOR.PATCH` 稳定语义化版本，不跟 `/releases/latest` 与预发布 tag）。
- **上游同步 PR（`sync/upstream/<40位sha>`）严禁 Squash / Rebase，必须使用 `Create a merge commit`**；Squash 会丢失上游父提交并导致后续反复假冲突。
- 非高危路径自动合并门禁：`.github/workflows/auto-merge-upstream.yml` 在 `main` 上以 `workflow_run` 运行，仅在全部必需 CI 通过、双父 merge commit 校验通过、变更文件数未超阈值且未命中 `.github/scripts/lib/high-risk-paths.sh` 高危路径时，自动执行 `Merge Commit` 并触发 Cloudflare 自动构建。
- 命中高危路径（`.github/`、`cloudflare/`、`Dockerfile*`、`*.sql`、数据库模型/存储层、`payment/`、`wallet/`、`order/`、`authz/`、`upstreamapi/` 等）或产生冲突时，必须人工审阅/解冲突后再以 `Merge Commit` 合并；需要临时关闭自动合并时，设置仓库 Variable `UPSTREAM_AUTO_MERGE=false`。
