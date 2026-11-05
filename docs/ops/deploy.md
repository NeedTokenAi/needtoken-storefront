# 生产部署（NeedToken storefront）

目标状态：只由 GitHub Actions 工作流 `deploy-cloudrun` 部署 Cloud Run 服务 `needtoken-storefront`。凭证用 Workload Identity Federation，不用长期 JSON 密钥。

## 先关掉 Cloud Build，避免部署两次

项目 `suichong` 里已有 Cloud Build 触发器 `needtoken-storefront-asia-southeast1-NeedTokenAi`。它会在 `main` 上构建，并立刻把流量切到 100%。已知一次：build `58a4277f` 把 SHA `d3e3903` 部署成修订版本 `needtoken-storefront-00006-dl9`。

在启用本工作流之前，必须禁用这个触发器，或删掉它的 Cloud Run 部署 / 切流量步骤，保证只有一条流水线会改生产流量。两条一起开着会各部署一次并抢流量。

触发器多半在区域 `asia-southeast1`（Cloud Run 持续部署）。列表为空时再查 `global`。控制台里直接停用也可以；命令因触发器代际可能略有差别。

```bash
gcloud builds triggers list --project=suichong --region=asia-southeast1
gcloud builds triggers update needtoken-storefront-asia-southeast1-NeedTokenAi \
  --project=suichong --region=asia-southeast1 --disabled
```

本文件进 `main` 之后，下一次 push 到 `main` 且 `ci` 成功就会尝试走 Actions。请先关 Cloud Build 再合并。变量还没配好时，Actions 会在联系 GCP 之前失败，不会新建服务；Cloud Build 若仍开着，仍会自己部署。

## 什么时候会部署

工作流：`.github/workflows/deploy-cloudrun.yml`。

1. `ci` 在 **push 到 `main`** 上成功结束后，由 `workflow_run` 拉起。会再核对：仓库是 `NeedTokenAi/needtoken-storefront`、事件是 `push`、分支是 `main`、结论是 `success`、使用该次运行的 `head_sha`。不看 `workflow_run` 的一句话结论就算数，会把该次 `ci` 的每个 job 拉出来。`Verify installer`、`Verify API`、`Verify release config`、`Verify fullstack build` 都必须是 `success`。若能读到 `main` 的 ruleset 要求的 status checks，那些也必须成功。
2. 维护者在 **`main` 上** `workflow_dispatch`，填写完整 `git_sha`。该 SHA 必须仍在 `main` 历史上，并且已有一次 push 到 `main` 的成功 `ci`。默认 SHA 必须等于当前 `main`。只有 `allow_behind_main=true` 才部署更旧的提交。回滚优先切流量，不要靠重部署旧镜像。

不会部署的情况：pull request、`dev`、fork、`ci` 失败、手动 `workflow_dispatch` 跑 `ci`（事件不是 `push`）。`ci` 没有 `id-token: write`，拿不到生产凭证。

`main` 又往前走了：这次自动部署直接跳过，不构建、不切流量。部署 job 排队期间若 `main` 已变，构建可以做完，但切流量前会再查一次，过期就不把流量给新修订版本。正在跑的部署不会被取消，避免流量切到一半。还没开始、且记录的 SHA 已是当前 `main` 祖先的自动部署会被取消。同一时间只有一个 deploy job（concurrency `deploy-cloudrun-production`）。

## Actions Variables

在仓库 `NeedTokenAi/needtoken-storefront` 设置：

| 变量 | 值 |
| --- | --- |
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | `projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL/providers/PROVIDER` |
| `GCP_SERVICE_ACCOUNT` | 部署用服务账号邮箱 |
| `GCP_PROJECT_ID` | `suichong` |
| `GCP_REGION` | `asia-southeast1` |
| `CLOUD_RUN_SERVICE` | `needtoken-storefront` |
| `ARTIFACT_REGISTRY_IMAGE` | `asia-southeast1-docker.pkg.dev/suichong/cloud-run-source-deploy/needtoken-storefront/needtoken-storefront` |

`ARTIFACT_REGISTRY_IMAGE` 不要带 tag 或 digest。镜像标签是完整 git SHA，Cloud Run 上的是 push 当时记下的 `@sha256:...`。

## 还要在 GCP 里做的事

下面的 pool / provider / 服务账号名字可以改，但 attribute condition 不要放宽。`workflow_run` 的 OIDC `ref` 是 `refs/heads/main`（工作流文件所在的默认分支），`sha` 是当时的默认分支 SHA，**不是**要部署的那次提交。不要用 `assertion.sha` 做条件。部署的提交由工作流自己 checkout 并核对。

```bash
PROJECT=suichong
REGION=asia-southeast1
POOL=github-actions
PROVIDER=needtoken-storefront
SA=needtoken-storefront-deploy
REPO=NeedTokenAi/needtoken-storefront

gcloud iam workload-identity-pools create "$POOL" \
  --project="$PROJECT" --location=global \
  --display-name="GitHub Actions"

gcloud iam workload-identity-pools providers create-oidc "$PROVIDER" \
  --project="$PROJECT" --location=global \
  --workload-identity-pool="$POOL" \
  --display-name="needtoken-storefront" \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.ref=assertion.ref,attribute.job_workflow_ref=assertion.job_workflow_ref" \
  --attribute-condition="assertion.repository=='${REPO}' && assertion.ref=='refs/heads/main' && assertion.job_workflow_ref=='${REPO}/.github/workflows/deploy-cloudrun.yml@refs/heads/main'"

PROJECT_NUMBER=$(gcloud projects describe "$PROJECT" --format='value(projectNumber)')
gcloud iam service-accounts create "$SA" \
  --project="$PROJECT" \
  --display-name="NeedToken storefront Cloud Run deploy"

gcloud iam service-accounts add-iam-policy-binding \
  "${SA}@${PROJECT}.iam.gserviceaccount.com" \
  --project="$PROJECT" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL}/attribute.repository/${REPO}"

gcloud artifacts repositories add-iam-policy-binding cloud-run-source-deploy \
  --project="$PROJECT" --location="$REGION" \
  --member="serviceAccount:${SA}@${PROJECT}.iam.gserviceaccount.com" \
  --role="roles/artifactregistry.writer"

gcloud run services add-iam-policy-binding needtoken-storefront \
  --project="$PROJECT" --region="$REGION" \
  --member="serviceAccount:${SA}@${PROJECT}.iam.gserviceaccount.com" \
  --role="roles/run.developer"
```

部署修订版本还需要 `roles/iam.serviceAccountUser`（`iam.serviceAccounts.actAs`），对象是 **Cloud Run 服务现在正在用的运行时服务账号**，不是上面这个部署账号。先查再绑：

```bash
gcloud run services describe needtoken-storefront \
  --project=suichong --region=asia-southeast1 \
  --format='value(spec.template.spec.serviceAccountName)'
```

不要把这套 WIF 绑到 `ci.yml`、pull request 或其他分支。`ci` 即使以后加上 `id-token: write`，provider 上的 `job_workflow_ref` 也会拒绝它。

## 部署时实际做了什么

- checkout 门禁里定下来的那一个 SHA，并核对 `HEAD`。
- `docker buildx` 构建现有 `Dockerfile`：admin `build:fullstack`、user `build`、Go `-tags release,fullstack`，平台 `linux/amd64`。
- 只传入 Dockerfile 里已经声明的 `APP_VERSION`（值是 git SHA）。**不传 `VITE_NEEDTOKEN_*`。** Dockerfile 的前端阶段没有这些 ARG/ENV，只在工作流里加 `--build-arg` 不会进包。运行时 Cloud Run 环境变量也改不了已经编译的前端。默认控制台和 API 基址在 `frontend/user/src/templates/needtoken/config.ts`，是 `https://api.needtoken.ai`。以后要覆盖，先改 Dockerfile，再改工作流。
- 镜像标签是完整 SHA。Cloud Run 使用 push 输出的 digest：`gcloud run services update --image=...@sha256:... --no-traffic`。不重建服务，不改 env、secret、资源、扩缩。
- 给新修订版本打临时 tag `smoke`（不占流量百分比），请求它的 URL：
  - `GET /health` 的 `status` 为 `ok`
  - 首页 HTML 引用构建出的 `/assets/*.js`，而不是开发用的 `/src/main.ts`
  - `GET /api/v1/public/config` 的 `status_code` 为 `0`，且 `data.storefront_template` 为 `needtoken`
- 通过后 `gcloud run services update-traffic --to-revisions=NEW=100`，再删掉 `smoke` tag。
- 不调用 `admin reset-password` / `reset-2fa`，不重建数据库，不导入商品，不打开销售开关。
- Job summary 和 artifact `deploy-record-<run_id>` 里有 source SHA、digest、修订版本、流量百分比和冒烟结果。

冒烟失败或中途 `main` 已变：新修订版本留在 0% 流量，生产仍指向原来的修订版本。

服务不存在时工作流失败，不会执行 `gcloud run deploy` 去创建。

## 数据库和回滚

新修订版本一启动，进程就会连 **当前服务配置里的那个库** 并执行 `AutoMigrate`。`--no-traffic` 只是用户流量先不到这个修订版本，容器仍会起来，迁移仍会跑。流量切回旧修订版本 **不会** 把 schema 退回去；迁移开始之后，旧二进制可能已经不兼容。

资金、支付、钱包、迁移相关的变更，先在隔离数据库演练，再让这份镜像进生产。

只回滚流量：

```bash
gcloud run revisions list \
  --service=needtoken-storefront --region=asia-southeast1 --project=suichong

gcloud run services update-traffic needtoken-storefront \
  --region=asia-southeast1 --project=suichong \
  --to-revisions=PREVIOUS_REVISION=100
```
