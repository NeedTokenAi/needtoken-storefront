# 上游同步（dujiao-next）

工作流：`.github/workflows/sync-upstream.yml`。每天 `0 0 * * *`（UTC 00:00，北京时间 08:00）以及手动 `workflow_dispatch`。只在 `refs/heads/main` 上的工作流文件会跑。没有 `id-token`，不使用任何 GCP 变量或生产 secret。

同步任务只做 fetch、merge、开 PR / issue。不编译，不执行合并进来的代码。

## 通道

仓库变量 `UPSTREAM_CHANNEL`，不设置时是 `stable-tags`。

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
- 冲突：`git merge --abort`，不推送。开一条 issue，标题含上游 SHA，正文有我们的 main SHA、上游 SHA、冲突文件，并写明要 Codex 解决。不要把带冲突标记的分支 force-push 到 `main`。同一条 issue 不重复开。冲突还在时，之后的定时运行会失败并指向它。
- 干净：只推这个分支。PR 说明里要求用 **Create a merge commit**，不要 Squash、不要 Rebase。Squash 会丢掉两个父提交，下次无法判断上游提交是否已经在 main 里。

远程如果已经有同名分支且包含当前 main 和该上游提交，就只补 PR，不 force-push。对不上就失败并停下。

## 让 CI 跑起来

用默认 `GITHUB_TOKEN` 开的 PR **不会**触发其他工作流，所以 `ci` 不会自动出现。

优先做法：配置 Actions secret `SYNC_GITHUB_TOKEN`。

- 细粒度 PAT 或 GitHub App installation token，仅 `NeedTokenAi/needtoken-storefront`。
- 权限只给 Contents（读写）和 Pull requests（读写）。不要给 Administration，不要给其他仓库，不要给 GCP / WIF。
- 工作流用这个 token checkout、推分支、开 PR。`pull_request` 会以这个身份触发 `ci`。

没有这个 secret 时，工作流仍用 `GITHUB_TOKEN` 开 PR，然后对头分支执行 `workflow_dispatch` 跑 `.github/workflows/ci.yml`。`ci` 因此多了一个 `workflow_dispatch`。这次运行没有生产凭证。`deploy-cloudrun` 只认 **push 到 main** 的成功 `ci`，手动 dispatch `ci` 不会部署。

不要把同步改成 `pull_request_target`。那个事件会在 base 的特权上下文里 checkout PR 头，不能用来跑候选代码。

建议把 `SYNC_GITHUB_TOKEN` 放到仅允许 `main` 部署的 GitHub Environment，并限制谁可以手动跑工作流。能改非 main 分支并手动跑工作流的人，已经有仓库写权限；token 仍然不要做成全权限 PAT。

## 合并之后

审查同步 PR 时看 `.github/` 有没有被上游改掉。有资金、支付、钱包或 schema 变更时，先在隔离数据库演练。生产部署见 [deploy.md](deploy.md)。新修订版本启动就会对现有生产库做 `AutoMigrate`，和这次同步工作流无关，但合并进 `main` 且 `ci` 通过后，部署工作流会把那个提交送上 Cloud Run。
