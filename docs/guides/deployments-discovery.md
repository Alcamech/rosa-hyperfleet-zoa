# ZOA deployment discovery and direct Access override

On your **laptop**, ZOA finds the right **Access Lambda** for a Seaplane environment using **Central account SSM**. You do **not** set URLs or regions for normal work.

| Doc                                                                            | Use when                                                            |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------- |
| [Operator workflow](operator-workflow.md)                                       | Full session path (start, join, `zoa run`, terminate)               |
| [Boundary architecture — Discovery](../boundary/architecture.md#discovery-ssm) | **Which account** holds which SSM path (diagram + MC targets in RC) |

This page covers **CLI behavior only**: default reads, regional failover, and DR overrides.

## First day — what to run

After Central AWS login (`rh-aws-saml-login` or your team’s flow):

```bash
zoa deployments              # names you can use below (e.g. us-east-1)
zoa targets us-east-1        # RC + MC target ids (e.g. regional, mc01)
zoa session start us-east-1 regional --reason ROSAENG-1234
```

| Term           | Meaning                                                                         |
| -------------- | ------------------------------------------------------------------------------- |
| **Deployment** | Environment name from `zoa deployments` (e.g. `us-east-1`).                     |
| **Target**     | Cluster id from the `TARGET` column of `zoa targets` (e.g. `regional`, `mc01`). |

**Targets** are **not** in Central SSM. The CLI calls **Access**, which reads **`/zoa/targets/<deployment>/…`** in the **RC** account (MC targets are stored there too — see [architecture](../boundary/architecture.md#discovery-ssm)).

You do **not** need `AWS_REGION` or `ZOA_*` env vars for this discovery path.

## Default behavior (no extra flags)

This is what almost everyone uses every day.

1. ZOA reads `/zoa/deployments/<deployment>` in the Central account using your current AWS credentials.
2. **Automatic failover:** SSM is tried in **`us-east-1` first** (default). If that request **returns an error**, the CLI retries the **same path** in **`eu-west-1`**. No flag is required; failover is silent if the second region works.
3. From that JSON it gets the **Access URL** and **invoker role**, assumes the invoker role, and calls Access (`targets`, `session`, …).

Platform Terraform keeps the **same** deployment record in both Central regions so either copy should match.

AWS Systems Manager Parameter Store has a published monthly SLA of **99.9%**; in practice **`us-east-1`** has often seen **>99.99%** steady-state availability, but multi-hour outages still happen during broad regional or identity (IAM/KMS) incidents. That is why Terraform replicates parameters to **`eu-west-1`** and the CLI retries there on error—overrides below should be rare.

**Choose a region yourself:** pass **`--deployments-region <aws-region>`** (for example `eu-west-1`) on `zoa deployments`, `zoa targets`, or `zoa session`. That **disables** automatic failover and uses **only** the region you name.

**You only need direct Access URL flags if runbooks say deployment SSM is down in every region.**

## Optional overrides

Use these only when the default path (automatic `us-east-1` → `eu-west-1` failover) is not enough.

| Situation                                         | Flags                                                       | Commands                                                                     | Effect                                                                                                                                                                                                                                                                                                               |
| ------------------------------------------------- | ----------------------------------------------------------- | ---------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Pin one Central SSM region**                    | `--deployments-region <aws-region>`                         | `deployments`, `targets`, every `session` subcommand that names a deployment | Reads `/zoa/deployments/*` **only** in that AWS region. **Turns off** automatic failover. Use when runbooks point you at the replica (e.g. `eu-west-1`) while primary SSM is failing.                                                                                                                                |
| **Central deployment SSM unavailable everywhere** | `--access-url` **and** `--invoker-role-arn` (both required) | `targets` and `session` only — **not** `deployments`                         | Skips **Central deployment SSM autodiscovery**. You still pass the **deployment name** (e.g. `us-east-1`) so Access knows which environment to use. You must already have the Access URL and invoker ARN (saved from an earlier `zoa deployments`, IC/runbook, or a future **private ZOA deployments** page for DR). |

`zoa deployments` always needs SSM — it cannot bypass autodiscovery with `--access-url`. When SSM is down everywhere, use URL and ARN from your team or DR documentation on `zoa targets` / `zoa session` instead.

## Examples

Deployment **`us-east-1`**, targets **`regional`** / **`mc01`**. Session id is illustrative.

### Every day (default)

```bash
zoa deployments
zoa targets us-east-1
zoa session start us-east-1 regional --reason ROSAENG-1234
zoa session list us-east-1
zoa session join us-east-1/1989937d-0889-485d-b64e-facee9e0150d
```

### Specify Central SSM region (`--deployments-region`)

Use this when you want a particular region (for example the replica) instead of default failover.

```bash
zoa deployments --deployments-region eu-west-1
zoa targets us-east-1 --deployments-region eu-west-1
zoa session start us-east-1 regional --reason ROSAENG-1234 --deployments-region eu-west-1
zoa session list us-east-1 --deployments-region eu-west-1
zoa session join us-east-1/1989937d-0889-485d-b64e-facee9e0150d --deployments-region eu-west-1
```

### Direct Access URL (Central deployment SSM unavailable)

You need the **Access Function URL** and **invoker role ARN** before running these commands (they are normally returned by `zoa deployments`). During a Central SSM outage, IC may publish them on a runbook or a planned **private ZOA deployments** page (DR-only mirror of deployment endpoints).

```bash
zoa targets us-east-1 \
  --access-url 'https://<id>.lambda-url.us-east-1.on.aws/' \
  --invoker-role-arn 'arn:aws:iam::123456789012:role/us-east-1-zoa-access-invoker'

zoa session start us-east-1 regional --reason ROSAENG-1234 \
  --access-url 'https://<id>.lambda-url.us-east-1.on.aws/' \
  --invoker-role-arn 'arn:aws:iam::123456789012:role/us-east-1-zoa-access-invoker'

zoa session join us-east-1/1989937d-0889-485d-b64e-facee9e0150d \
  --access-url 'https://<id>.lambda-url.us-east-1.on.aws/' \
  --invoker-role-arn 'arn:aws:iam::123456789012:role/us-east-1-zoa-access-invoker'
```

Other session commands (`terminate`, `history`, filters on `list`) accept the same global flags where they take a deployment name.
