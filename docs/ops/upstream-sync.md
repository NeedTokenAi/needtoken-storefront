# 上游同步（dujiao-next）

工作流：

- `.github/workflows/sync-upstream.yml`：每天 `0 0 * * *`（UTC 00:00，北京时间 08:00）以及手动 `workflow_dispatch`。只在 `refs/heads/main` 上运行，只做 fetch、merge、开 PR / issue，不编译、不执行合并进来的代码。
- `.github/workflows/auto-merge-upstream.yml`：通过 `workflow_run` 监听 `ci` 完成事件，在受信任的 `main` 上执行非高危路径自动合并门禁；同样不 checkout 或执行 PR 分支代码。

两个工作流都没有 `id-token`，不接触任何生产云凭证，也不要改成 `pull_request_target`。

## 通道

仓库变量 `UPSTREAM_CHANNEL`，不设置时是 `stable-tags`。可选仓库变量 `UPSTREAM_MIN_TAG`（格式 `vMAJOR.MINOR.PATCH`）用于设置最低版本底线，低于底线的旧稳定 tag 会安全跳过。

上游是 `dujiao-next/dujiao-next`。历史 tag 里有 `v0.0.5-beta` 到 `v0.0.9-beta`。当前 GitHub Release 没有 prerelease，但 **不要用 `/releases/latest`**：

- “latest” 是可以改指的指针，不一定是最高的稳定版本。
- Release API 的 `target_commitish` 对 `v1.5.2` 返回的是分支名 `main`，不是提交。

`stable-tags`：列出 git tag，只留 `vMAJOR.MINOR.PATCH`（没有 `-rc` / `-beta` / `-alpha`），用版本号排序取最高的一个，再把这个 tag 解析成 commit SHA。轻量 tag 直接是 commit；附注 tag 会再剥一层。

`releases`：在非 draft、非 prerelease 的 Release 里用同一条 semver 规则取最高 tag，再解析成 commit SHA。仍然不调用 `/releases/latest`。

其他值会让工作流失败。

## 开 PR 的规则

- 该 SHA 已经在我们的 `main` 里：成功退出，并在 summary 里说明。
- 已经有打开的 `sync/upstream/*` PR：不再开第二个。
- 从当时的 `main` 建分支 `sync/upstream/<40位sha>`。只 `git fetch` 这一个上游提交，不把上游 tag 推到 fork。
- `git merge --no-ff`，普通 merge commit。不用 squash，不用 `-X ours` / `-X theirs`。
- 冲突：`git merge --abort`，不推送。开一条 issue，标题含上游 SHA，正文有我们的 main SHA、上游 SHA、冲突文件。不要把带冲突标记的分支 force-push 到 `main`。同一条 issue 不重复开。冲突还在时，之后的定时运行会失败并指向它。
- 干净：只推这个分支。PR 说明里要求用 **Create a merge commit**，不要 Squash、不要 Rebase。Squash 会丢掉两个父提交，下次无法判断上游提交是否已经在 main 里。

远程如果已经有同名分支且包含当前 main 和该上游提交，就只补 PR，不 force-push。对不上就失败并停下。

## 非高危路径自动 Merge Commit（自动触发 Cloudflare 构建）

当 `ci` 在 `sync/upstream/<40位sha>` 上成功完成时，`.github/workflows/auto-merge-upstream.yml` 会运行 `.github/scripts/auto-merge-upstream.sh`（fail-closed）：

1. 校验开关：默认开启；在仓库 Variables 设置 `UPSTREAM_AUTO_MERGE=false` 可一键熔断关闭。
2. 校验提交拓扑：PR 必须是打开的非草稿 PR，目标为 `main`，头提交 SHA 与通过 CI 的 SHA 完全一致，且为双父提交 merge commit（`parent[1] == 分支上游 SHA`、`parent[0]` 在 `main` 祖先链上）。
3. 校验全部必需 CI：`Verify installer`、`Verify API`、`Verify release config`、`Verify fullstack build` 必须全部 `completed + success`，且该 SHA 上无其他进行中或失败检查。
4. 重新拉取 PR 完整变更文件（校验总数与 API 返回数一致，防止 3000 文件截断；变更文件数默认不得超过 `UPSTREAM_AUTO_MERGE_MAX_FILES=300`）并通过 `.github/scripts/lib/high-risk-paths.sh` 判定：
   - 命中高危路径（`.github/`、`cloudflare/`、`Dockerfile*`、`docker-compose*.yml`、`.goreleaser*.yaml`、`config.yml.example`、`*.sql`、`migrations/`、`internal/models/`、`internal/database/`、`internal/authz/`、`internal/modules/{payment,wallet,order,upstreamapi}/`、`internal/upstream/`、`internal/bootstrap/`、`internal/workflows/`、`*/gormstore/` 等）或超文件数阈值时：**拒绝自动合并**，并在 PR 下幂等留言列出命中文件，等待人工审阅后点击 **Create a merge commit**。
   - 未命中高危路径且全部门禁通过：以 `merge_method=merge` + 精确 `sha` 原子合入 `main`，并清理已合并的 `sync/upstream/<sha>` 分支。合入 `main` 后由 **Cloudflare Workers Builds** 自动构建并发布。

## 让 CI 跑起来

用默认 `GITHUB_TOKEN` 开的 PR **不会**触发其他工作流，所以 `ci` 不会自动出现。

优先做法：配置 Actions secret `SYNC_GITHUB_TOKEN`。

- 细粒度 PAT 或 GitHub App installation token，仅授予需要同步的 NeedToken Fork 仓库。
- 权限只给 Contents（读写）和 Pull requests（读写）。不要给 Administration，不要给生产云凭证。
- 工作流用这个 token checkout、推分支、开 PR、执行低风险自动合并。`pull_request` 会以这个身份触发 `ci`，合入 `main` 后同时触发 Cloudflare Workers Builds 与 `main` 上的 `push` 工作流。

没有这个 secret 时，`sync-upstream.yml` 仍会用 `GITHUB_TOKEN` 开 PR 并对头分支执行 `workflow_dispatch` 跑 `.github/workflows/ci.yml`。

不要把同步改成 `pull_request_target`。那个事件会在 base 的特权上下文里 checkout PR 头，不能用来跑候选代码。

## 合并之后

审阅高危同步 PR 时重点看 `.github/`、资金/支付/钱包、鉴权和数据库 schema 变更；涉及资金或迁移时先在隔离数据库演练。合入 `main` 后从对应提交的 Cloudflare Check Run 核实 Build ID 与 Version ID。
