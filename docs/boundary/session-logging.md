# ZOA Boundary — Session logging (CloudWatch)

Boundary sessions produce **two audit layers**:

| Layer                       | Store                                  | Use for                                                                                       |
| --------------------------- | -------------------------------------- | --------------------------------------------------------------------------------------------- |
| **ECS Exec PTY transcript** | CloudWatch Logs (`ssm-sessions` group) | Shell forensics: commands typed, `zoa run` stdout in the session, Claude TUI output           |
| **ZOA application audit**   | DynamoDB **audit** + **executions**    | TA dispatch proof, parameters, operator attribution via [identity bridge](identity-bridge.md) |

`zoa run` is recorded in DynamoDB **even if** Exec logging fails. Use **`zoa runs`** / **`zoa audit`** for TA detail; use CloudWatch for interactive shell evidence.

## Log groups (per target cluster)

Both groups use the **same KMS CMK** as other ZOA resources (shared key on RC; per-cluster key on MC when Terraform does not pass `kms_key_arn`). **Retention** is at least **365 days** (`max(365, boundary_log_retention_days)` in `terraform/modules/zoa-lambda/boundary-ecs.tf`).

| Purpose               | Log group                                     | Stream naming                           | Contents                                                                                                                                                               |
| --------------------- | --------------------------------------------- | --------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Container stdout**  | `/ecs/<cluster_id>/zoa-boundary`              | `container/zoa-boundary/<task-id>`      | Entrypoint only: banner, tool checks, “ready for connections”.                                                                                                         |
| **ECS Exec sessions** | `/ecs/<cluster_id>/zoa-boundary/ssm-sessions` | `ecs-execute-command-<exec-session-id>` | One stream **per Exec join** (each `zoa session join` or **`session start --exec-script`** / **`join --exec-script`**); shared log group across all boundary tasks on that cluster |

`<cluster_id>` is the Seaplane cluster id (`regional_id` on RC, `management_id` on MC).

**Do not** search session transcripts in the **container** group. **Do not** use the exec group to debug task startup — use the container group.

Terraform: [rosa-seaplane `terraform/modules/zoa-lambda/`](https://github.com/openshift-online/rosa-seaplane/tree/main/terraform/modules/zoa-lambda) (`boundary-ecs.tf`, `boundary-task.tf`).

FedRAMP AU-09: dedicated exec log group, KMS encryption, `cloudWatchEncryptionEnabled = true` on the ECS cluster `executeCommandConfiguration`.

## What Exec PTY logging captures

- Interactive shell input and output (including `zoa run` printed in the session).
- **`session … --exec-script`**: same PTY capture as interactive Exec (MOTD, prompts, plugin banners, plus the automation script). Exec-script stdout on the laptop is this full transcript, not a minimal script-only log.
- **Claude Code** TUI: partial terminal capture; model payloads are **not** in Exec logs. Optional [Bedrock model invocation logging](https://docs.aws.amazon.com/bedrock/latest/userguide/model-invocation-logging.html) (metadata only) is separate account-level configuration in Seaplane Bedrock modules.

## Platform requirements (why the image matters)

ECS Exec logging to CloudWatch requires:

1. Cluster **execute command** logging override → `.../ssm-sessions` with the shared CMK.
2. Task role permissions on the **exec log group** and KMS.
3. **`EnableExecuteCommand: true`** on `RunTask` (`pkg/awsecs/client.go`).
4. Boundary image includes **`script`** and **`cat`** (`util-linux` in `Containerfile.boundary`); entrypoint fails closed if missing.

**Interactive user:** ECS Exec attaches as **root**; Access returns `exec_command` (default `runuser -u sre -- /bin/bash -l`) on join. **Exec credentials** are vended by Access (`*-zoa-access-exec-scoped` role); the laptop does not use broad deployment-admin roles for Exec.

## Task stop (`stopTimeout`)

Fargate **`stopTimeout` is 30 seconds** on the boundary task definition. There is no persistent home volume — only time for SIGTERM and log agent flush after terminate or reaper `StopTask`.

## Find the Exec log stream (not the ZOA `session_id`)

CloudWatch Insights filters on **`@logStream`**, which is named after the **ECS Exec session id** — **not** the ZOA boundary **`session_id`** (DynamoDB UUID on the session row / shell prompt).

| Identifier                           | Example                                          | Use in CloudWatch?                                                               |
| ------------------------------------ | ------------------------------------------------ | -------------------------------------------------------------------------------- |
| ZOA **`session_id`**                 | `bcacb5bb-041b-47e7-b839-6ad619f6a91b`           | **No** — identifies the boundary session in DynamoDB and `zoa session terminate` |
| **Exec join** (`exec_session_ids[]`) | `ecs-execute-command-jtopofu479y5el3kgiob9ubti4` | **Yes** — value in JSON is the full **`@logStream`** name (prefix included)      |

**One ZOA boundary session, many Exec joins:** every successful **`zoa session join`** (including reconnects after disconnect) creates a **new** SSM/ECS Exec session id. Access appends each id to **`exec_session_ids`** on the session row when join registration succeeds (`POST .../sessions/exec-attached/{session-id}`). Each id is a **separate** log stream in the **same** `/ecs/<cluster_id>/zoa-boundary/ssm-sessions` log group.

**From the laptop (audit view):**

```bash
zoa session history <deployment> -o json
```

Pick the boundary session row, then an entry from **`exec_session_ids`** for the join you care about (most recent is usually last in the array). Table output shows a truncated id or `N joins` — use **JSON** for the full exec id.

Also note **`target_cluster`** on that row (e.g. `eph-3cfd072c-regional`) so you open the correct log group and Grafana datasource (RC vs MC account).

If **`exec_session_ids`** is empty but you know a join happened, the CLI may have failed exec-attached registration; the reaper can still use SSM/CW directly — operators may need AWS console/SSM session history for that join id.

## Viewing session logs in Grafana Explorer

Operators use **Grafana** with the **CloudWatch Logs** datasource for the target AWS account — not the AWS CLI on a laptop (and not from inside the boundary container for fleet-wide review).

Same pattern as [Lambda logs in observability](../observability.md#viewing-logs-in-grafana-explorer):

1. Open **Grafana → Explore**.
2. Select the **CloudWatch Logs** datasource for the account that runs the boundary task:
   - **RC boundary** → regional cluster account datasource.
   - **MC boundary** → management cluster account datasource.
3. **Search log groups** and open:

   ```
   /ecs/<cluster_id>/zoa-boundary/ssm-sessions
   ```

4. Run **CloudWatch Logs Insights** with an **exact** `@logStream` match. Copy a string from **`exec_session_ids`** in `zoa session history -o json` (stored as the full stream name):

   ```
   fields @timestamp, @message
   | filter @logStream = "ecs-execute-command-jtopofu479y5el3kgiob9ubti4"
   | sort @timestamp asc
   | limit 10000
   ```

5. Example — recent activity across **all** Exec streams on the cluster (narrow the time range in Explore first):

   ```
   fields @timestamp, @logStream, @message
   | filter @logStream like /ecs-execute-command-/
   | sort @timestamp desc
   | limit 100
   ```

6. **Startup-only** debugging — container group:

   ```
   /ecs/<cluster_id>/zoa-boundary
   ```

   Streams: `container/zoa-boundary/<ecs-task-id>`. The ECS **task id** (not session UUID) appears in the session row as `task_id` when the task is active.

**Latency:** After disconnecting Exec, PTY events often appear within **1–2 minutes**.

## DynamoDB `exec_session_ids`

After `ecs:ExecuteCommand`, the laptop CLI calls Access **`POST /api/v0/sessions/exec-attached/{zoa-session-id}`** to append the **Exec session id** to the boundary session row. That id is what operators use for CloudWatch forensics. Failure prints a warning but still opens the shell. Idle reaper logic does **not** depend on this field — see [session reaper](session-reaper.md).
