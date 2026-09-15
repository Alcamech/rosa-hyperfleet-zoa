# End-to-End Testing

Practical guide for ZOA's e2e coverage: what runs where, how to run it yourself against a real
environment (dev-account ephemeral, standing integration, or CI), and how container images flow
from a PR into that environment.

## Test Tiers

| Tier            | Lives in                                    | Runs from                                                                                                                          | Scope                                                                                                                                                                                                                                                                                                                                                                  | Image under test                                                                                                                                            |
| --------------- | ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Deep**        | `rosa-hyperfleet-zoa/test/e2e/` (all specs) | This repo's own `on-demand-e2e` (PR) + `nightly-ephemeral` (daily)                                                                | Every registered Trusted Action, both scopes (`kube-api`/`aws-api`), read+write+dry-run, sync+async, RC _and_ MC, real (non-dry-run) `delete_pod`/`rollout_restart` against `kube-system/coredns`, cooldown enforcement (per action+params), all CLI commands (`get`, `runs`, `audit`, `output`, `logs`, `download`), generic param validation, e2e conformance checks | on-demand-e2e: **this PR's** CI-built image (via Quay). nightly: the `zoa_lambda_image_tag`/`zoa_runner_image_tag` pinned in `rosa-hyperfleet@main`'s config |
| **Light/smoke** | Same suite, `Label("smoke")` subset only    | `rosa-hyperfleet`'s `nightly-ephemeral`/`on-demand-e2e`, and `rosa-hyperfleet-api`'s `on-demand-e2e`. On nightlies, full deep suite | 6 specs (~2min): `zoa version`, `zoa actions` discovery, one kube-api read (`get_resource --resource nodes`), one aws-api read (`list_eks_clusters`), one write TA `--dry-run` (`rollout_restart`), one async execution with `--wait`                                                                                                                                  | Whatever `zoa_lambda_image_tag`/`zoa_runner_image_tag` is already pinned/deployed — **no PR-image override in these repos**                                 |

ZOA owns all of its own test logic, including what counts as "smoke," in one place
(`test/e2e/`). Consumer repos (`rosa-hyperfleet`, `rosa-hyperfleet-api`) select a labeled subset
of _this_ suite rather than re-implementing target discovery or CLI invocation a second time.

## Developer Workflow: Testing Code Changes

When you change ZOA code (API, Trusted Actions, executor, etc.) and want to validate against a
live ephemeral environment, you must **build and deploy your own images first**. The e2e test
suite only exercises the `zoa` CLI — the Lambda running on the ephemeral must already contain
your code changes.

> **Important:** Automatic image injection (CI builds your PR's images and deploys them) only
> happens in CI via the `on-demand-e2e` Prow job. For local development, you manage the image
> lifecycle manually.

### Step-by-step

```bash
# 1. Build + push both images to Quay (single command)
cd rosa-hyperfleet-zoa
make images-push
# → pushes quay.io/rrp-dev-ci/zoa-lambda:<commit> and zoa-runner:<commit>

# 2. Note the commit tag (printed during push, or run:)
git rev-parse --short HEAD   # e.g. fc40612

# 3. Configure the new image tag in rosa-hyperfleet
cd ../rosa-hyperfleet
# Edit config/defaults.yaml:
#   zoa_lambda_image_tag: "fc40612"
#   zoa_runner_image_tag: "fc40612"
uv run scripts/render.py     # regenerate deploy/ files

# 4. Resync the ephemeral to deploy the new Lambda
make ephemeral-resync ID=<your-env-id>

# 5. Run e2e tests (picks up your code changes via the new Lambda)
make ephemeral-zoa-e2e ID=<your-env-id> \
  ZOA_REF=my-feature-branch \
  ZOA_REPO=https://github.com/my-fork/rosa-hyperfleet-zoa.git
```

If you are only changing **test code** (files in `test/e2e/`) and not API/TA code, skip steps
1–4 — the existing Lambda is fine, you just need the tests to use your branch (via `ZOA_REF`).

### When is image management automatic?

| Scenario | Image management |
| --- | --- |
| **CI `on-demand-e2e`** (ZOA PR) | Automatic — ci-operator builds images, pushes to Quay, and overrides config before deploy |
| **CI `nightly-ephemeral`** | None — tests whatever tags are already pinned in `rosa-hyperfleet@main` config |
| **Local dev testing** | **Manual** — you build, push, configure, and resync (steps above) |

## Running the Tests

The suite drives the built `zoa` CLI against a real, already-provisioned RC and/or MC ZOA Lambda
API. It never needs a specific environment shape beyond "give me a URL" — the same suite runs
unmodified against a dev-account ephemeral env, a CI ephemeral env, or (in principle) a standing
integration environment.

### From `rosa-hyperfleet` (Recommended)

You need an ephemeral environment already provisioned (`make ephemeral-provision` in that repo —
see [`rosa-hyperfleet/docs/development-environment.md`](https://github.com/openshift-online/rosa-hyperfleet/blob/main/docs/development-environment.md)).

```bash
cd rosa-hyperfleet

# Deep suite — clones rosa-hyperfleet-zoa@main inside the test container.
# AWS credentials (rrp-rc, rrp-mc profiles) are set up automatically.
make ephemeral-zoa-e2e \
  ID=<your-env-id> \
  ZOA_REF=my-feature-branch \
  ZOA_REPO=https://github.com/my-fork/rosa-hyperfleet-zoa.git

# Smoke only (~2min)
make ephemeral-zoa-e2e-smoke ID=<your-env-id>

# Verbose output
GINKGO_FLAGS=-ginkgo.v make ephemeral-zoa-e2e ID=<your-env-id>
```

`ZOA_REF` defaults to `main`, `ZOA_REPO` defaults to the upstream repo. To test uncommitted
changes, push them to a branch first — there is no local-checkout-mount option (same convention
as `E2E_REF`/`E2E_REPO` for API tests and `CLI_REF`/`CLI_REPO` for CLI tests).

This is the recommended path because:

- AWS credentials are handled automatically (the container maps your host profiles to
  `rrp-rc`/`rrp-mc` inside the container).
- ZOA Lambda URLs are resolved from Terraform outputs.
- No manual env var setup needed.

### From This Repo Directly

If you are iterating on `test/e2e/*.go` itself and want fast feedback without container overhead,
you can run the suite directly against an existing ephemeral environment. You need two things:
the Lambda Function URLs and AWS credentials for both the RC and MC accounts.

**Step 1 — Get the Lambda URLs.** From the `rosa-hyperfleet` checkout:

```bash
make ephemeral-list ID=<your-env-id>
# Look for the ZOA RC and MC API URLs in the output
```

**Step 2 — Run both RC and MC together.** Your host `~/.aws/config` has profiles named
`rrp-regional-dev` and `rrp-management-dev`, but the test suite defaults to `rrp-rc` and `rrp-mc`
(the names used inside CI containers). Override with `ZOA_RC_AWS_PROFILE` and `ZOA_MC_AWS_PROFILE`:

```bash
cd rosa-hyperfleet-zoa

export ZOA_RC_API_URL="https://yyyy.lambda-url.us-east-1.on.aws/"
export ZOA_MC_API_URL="https://zzzz.lambda-url.us-east-1.on.aws/"
export ZOA_RC_AWS_PROFILE="rrp-regional-dev"
export ZOA_MC_AWS_PROFILE="rrp-management-dev"

make test-e2e              # full deep suite
make test-e2e-smoke        # smoke only
GINKGO_FLAGS=-ginkgo.v make test-e2e   # verbose output
```

RC and MC run in **parallel** automatically (separate `go test` processes, one per target). Each
process runs its specs sequentially. If you only set one URL, only that target runs.

Each target carries a `DeploymentTarget` field (`"rc"` or `"mc"`) derived from which URL env var was set. Specs
use this to pick the correct `--gather` value and expected tarball layout without hard-coding RC vs
MC — so the same test file is safe under parallel execution (each process only sees one target).

### `must_gather` platform dumps

`ta_mustgather_test.go` exercises async platform collection on the endpoint under test:

| Target | CLI | Tarball marker |
| ------ | --- | -------------- |
| RC | `--gather rc` | `rc/namespaces/platform-api/` |
| MC | `--gather mc` | `mc/namespaces/kube-applier/` |

The happy-path spec runs `must_gather --wait` (up to 20m), downloads `output.tar.gz` via
`zoa download`, and asserts the deployment-specific namespace path exists while the opposite
deployment tree (`mc/` or `rc/`) does not. HCP gather (`gather=hcp`) is covered separately later
via `E2E_HCP_CLUSTER_ID` — not part of the default deep suite yet.

Run only must_gather specs:

```bash
GINKGO_FLAGS='-ginkgo.focus=must_gather' make test-e2e
```

**Step 3 (optional) — Run only RC or only MC.**

```bash
# RC only
export ZOA_RC_API_URL="https://yyyy.lambda-url.us-east-1.on.aws/"
export ZOA_RC_AWS_PROFILE="rrp-regional-dev"
make test-e2e

# MC only
export ZOA_MC_API_URL="https://zzzz.lambda-url.us-east-1.on.aws/"
export ZOA_MC_AWS_PROFILE="rrp-management-dev"
make test-e2e
```

## AWS Credentials

RC and MC are **separate AWS accounts** with separate IAM principals. The test suite handles this
by setting `AWS_PROFILE` on each subprocess call (not globally), so both targets can run in the
same session without conflicting.

### Profile Names

The test suite defaults to `rrp-rc` and `rrp-mc` (the names used inside CI containers and
`make ephemeral-zoa-e2e`). On your host, ephemeral dev accounts use different names:

| Test suite default | Your host profile      | Purpose                               |
| ------------------ | ---------------------- | ------------------------------------- |
| `rrp-rc`           | `rrp-regional-dev`     | RC account — invoke RC ZOA Lambda     |
| `rrp-mc`           | `rrp-management-dev`   | MC account — invoke MC ZOA Lambda     |

Override with `ZOA_RC_AWS_PROFILE` / `ZOA_MC_AWS_PROFILE` when running from your host.
Inside CI containers or `make ephemeral-zoa-e2e`, the defaults (`rrp-rc`/`rrp-mc`) are correct —
no override needed.

### How the Test Suite Uses Profiles

The test suite (`test/e2e/helpers_test.go`) sets `AWS_PROFILE` per-subprocess, filtering out any
inherited `AWS_PROFILE`/`ZOA_API_URL` to avoid duplicate keys:

```go
func runZoa(tgt target, args ...string) (string, error) {
    cmd := exec.Command(zoaBin, args...)
    cmd.Env = append(filterEnv(os.Environ(), "AWS_PROFILE", "ZOA_API_URL"),
        "ZOA_API_URL="+tgt.APIURL,
        "AWS_PROFILE="+tgt.AWSProfile,  // rrp-rc or rrp-mc
    )
    out, err := cmd.CombinedOutput()
    return string(out), err
}
```

This ensures:

- RC specs use `rrp-rc` (can invoke RC Lambda)
- MC specs use `rrp-mc` (can invoke MC Lambda)
- No race conditions even with parallel spec execution

### Credential Setup per Scenario

| Scenario                                                   | Who manages credentials                                                                                                                           |
| ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `make ephemeral-zoa-e2e ID=...` (from `rosa-hyperfleet`)   | Automatic — `ephemeral-env.sh` maps your host profiles → `rrp-rc`/`rrp-mc` inside the container                                                  |
| `make test-e2e` (from this repo directly)                  | Manual — set `ZOA_RC_AWS_PROFILE=rrp-regional-dev` and `ZOA_MC_AWS_PROFILE=rrp-management-dev`                                                   |
| CI (`nightly-ephemeral`, `on-demand-e2e`)                  | Automatic — Prow mounts a Vault secret with `rrp-rc`/`rrp-mc` profiles pre-configured                                                            |
| Smoke via `rosa-hyperfleet`/`rosa-hyperfleet-api` CI jobs  | Automatic — same Prow secret; `rosa-hyperfleet/ci/e2e-tests.sh` sets profiles, then calls `make test-e2e-smoke`                                   |

## CI: How It Runs

### ZOA's Own Jobs (`openshift/release`)

Config:
[`ci-operator/config/openshift-online/rosa-hyperfleet-zoa/`](https://github.com/openshift/release/blob/master/ci-operator/config/openshift-online/rosa-hyperfleet-zoa/openshift-online-rosa-hyperfleet-zoa-main.yaml)
in `openshift/release`.

**`on-demand-e2e`** (presubmit, `/test on-demand-e2e` on a ZOA PR, `always_run: false`):

ci-operator builds both images (`Containerfile` → `zoa-lambda`, `Containerfile.runner` → `zoa-runner`),
the `rosa-hyperfleet-image-push` step mirrors them to `quay.io/rrp-dev-ci/zoa-{lambda,runner}` with
a `ci-<PR>-<BUILD_ID>` tag, and the provision step overrides `zoa_lambda_image_tag`,
`zoa_runner_image_tag`, `zoa_lambda_source_image`, and `zoa_runner_source_image` in
`config/defaults.yaml` before deploying. The e2e step auto-detects the PR's ZOA repo/branch so
the deep suite runs against the PR's own code and images.

**`nightly-ephemeral`** (periodic): provisions with whatever image tags are pinned in
`rosa-hyperfleet@main`'s config (Konflux-built). Catches infrastructure/platform regressions.

### Consumer Repo Jobs (`rosa-hyperfleet` / `rosa-hyperfleet-api`)

Their `nightly-ephemeral`, `on-demand-e2e`, and `rosa-regionality-compatibility-e2e` jobs call
`rosa-hyperfleet/ci/e2e-tests.sh`, which clones `rosa-hyperfleet-zoa@main` and runs:

- PRs (`JOB_TYPE=presubmit`): `make test-e2e-smoke` (~2min, 6 specs per target)
- Nightlies (`JOB_TYPE=periodic`): `make test-e2e` (full deep suite)

These jobs always test against whatever image tags are pinned in `rosa-hyperfleet` — they never
inject a custom ZOA image.

## Image Management

### How Images Are Published

| System      | When                       | Registry                                                        | Tag format                           | Expiry           |
| ----------- | -------------------------- | --------------------------------------------------------------- | ------------------------------------ | ---------------- |
| Konflux     | Every push to `main`       | `quay.io/redhat-user-workloads/rosa-tenant/zoa-{lambda,runner}` | Full commit SHA (e.g. `8af92e65...`) | Never            |
| ci-operator | ZOA PR `on-demand-e2e` job | `quay.io/rrp-dev-ci/zoa-{lambda,runner}`                        | `ci-<PR>-<BUILD_ID>`                 | No expiry policy |

Konflux builds are the **production** images (with attestation/SAST scans). The `config/defaults.yaml`
pin in `rosa-hyperfleet` always points at a Konflux-built tag.

ci-operator builds are used **only** for pre-merge CI testing.

### How `rosa-hyperfleet` Consumes Image Tags

Four config keys in `rosa-hyperfleet`'s `config/defaults.yaml` control which ZOA images get
deployed: `zoa_lambda_image_tag`, `zoa_runner_image_tag`, `zoa_lambda_source_image`, and
`zoa_runner_source_image`. These flow through `render.py` into Terraform inputs. Lambda images
are mirrored to ECR at deploy time; runner images are pulled directly from Quay by Kubernetes.

See [`rosa-hyperfleet/docs/adding-component-pre-merge.md`](https://github.com/openshift-online/rosa-hyperfleet/blob/main/docs/adding-component-pre-merge.md#multi-image--terraform-deployed-components-zoa)
for the full image override mechanism.

## Troubleshooting

| Symptom                                                                       | Cause                                                            | Fix                                                                                                                         |
| ----------------------------------------------------------------------------- | ---------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| `no target configured` test failure                                           | Neither `ZOA_RC_API_URL` nor `ZOA_MC_API_URL` is set             | Export at least one URL before running `make test-e2e`                                                                      |
| MC specs don't run, only RC                                                   | `ZOA_MC_API_URL` not set — env may not have an MC provisioned    | Set `ZOA_MC_API_URL`; if using `make ephemeral-zoa-e2e`, MC should always be available                                      |
| `InvalidSignatureException` or `AccessDeniedException` on MC calls            | Using the RC profile to call the MC Lambda (different AWS account) | Set `ZOA_MC_AWS_PROFILE=rrp-management-dev` (or your MC profile); verify with `AWS_PROFILE=<profile> aws sts get-caller-identity` |
| `InvalidSignatureException` or `AccessDeniedException` on RC calls            | Wrong profile for the RC account                                 | Set `ZOA_RC_AWS_PROFILE=rrp-regional-dev` (or your RC profile); verify with `AWS_PROFILE=<profile> aws sts get-caller-identity`   |
| `make ephemeral-zoa-e2e` fails to clone with a branch-not-found error         | `ZOA_REF` points at a branch that hasn't been pushed             | Push your branch first — this clones from the remote, not your local working tree                                           |
