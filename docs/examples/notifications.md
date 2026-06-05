# Notifications (notify)

How to configure outbound alerts for pipeline **start**, **success**, and **error** events, and how to **test channels without running a pipeline**.

kzero sends HTTP POSTs in **`run.mode: live`** only (`down`, `up`, `reset`). **`dry-run`** does not fire pipeline notifications. Use **`kzero notify test`** to verify wiring at any time.

## Quick test (no cluster mutations)

1. Enable at least one channel in your profile (or via `KZERO_*` env vars below).
2. Run:

```bash
kzero notify test --config /path/to/kzero.yaml
```

3. Expect exit **0** and stdout: `notify test: sent event "notify.test" to enabled channel(s)`.

Preview real event formatting:

```bash
kzero notify test -c /path/to/kzero.yaml --event pipeline.start
kzero notify test -c /path/to/kzero.yaml --event pipeline.success
kzero notify test -c /path/to/kzero.yaml --event pipeline.error
```

`pipeline.error` in test mode includes **sample** `failed_step` and `error` fields so you can check Slack/Teams layout before a real failure.

## YAML schema

```yaml
notify:
  on_error: true          # default true when any channel is enabled
  slack:
    enabled: false
    webhook_url: ""
  discord:
    enabled: false
    webhook_url: ""
  teams:
    enabled: false
    webhook_url: ""
  pagerduty:
    enabled: false
    routing_key: ""       # Events API v2 integration key
  webhook:
    enabled: false
    url: ""
    headers: {}           # optional extra HTTP headers
```

Annotated reference: [configs/kzero.sample.yml](../../configs/kzero.sample.yml).

## Events

| Event | When (live pipelines) |
|-------|------------------------|
| `pipeline.start` | After the **`Kubernetes target:`** block, before the first hook or step |
| `pipeline.success` | After successful `post-down` / `post-up` / full `reset` |
| `pipeline.error` | On fail-fast, **before** `hooks.on-error` (includes step ref when available) |
| `notify.test` | Only via **`kzero notify test`** (not sent by pipelines) |

Set **`notify.on_error: false`** to suppress **`pipeline.error`** while keeping start/success.

## Channel examples

### Generic webhook (JSON body)

Best for custom integrations, n8n, or internal receivers. The **`webhook`** channel sends the full structured payload:

```json
{
  "event": "pipeline.success",
  "command": "reset",
  "mode": "live",
  "client_id": "ops-team-a",
  "cluster_name": "example-cluster",
  "started_at": "2026-06-05T12:00:00Z",
  "duration": "18m32s"
}
```

```yaml
notify:
  webhook:
    enabled: true
    url: "https://hooks.example.com/kzero"
    headers:
      Authorization: "Bearer ${TOKEN}"   # expand in your secret manager / env before deploy
```

### Slack incoming webhook

```yaml
notify:
  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/services/T…/B…/…"
```

Slack receives a short **text** line (not the full JSON). Use **`webhook`** if you need the structured payload in Slack via a middleware.

### Discord webhook

```yaml
notify:
  discord:
    enabled: true
    webhook_url: "https://discord.com/api/webhooks/…"
```

### Microsoft Teams workflow / connector URL

```yaml
notify:
  teams:
    enabled: true
    webhook_url: "https://….webhook.office.com/…"
```

### PagerDuty Events API v2

```yaml
notify:
  pagerduty:
    enabled: true
    routing_key: "your-integration-key"
```

Errors trigger with **error** severity; start/success use **info**.

## Environment overrides (`KZERO_*`)

Viper prefix **`KZERO_`** with dots and dashes mapped to underscores (same as other config keys):

| Env var | YAML key |
|---------|----------|
| `KZERO_NOTIFY_ON_ERROR` | `notify.on_error` |
| `KZERO_NOTIFY_SLACK_ENABLED` | `notify.slack.enabled` |
| `KZERO_NOTIFY_SLACK_WEBHOOK_URL` | `notify.slack.webhook_url` |
| `KZERO_NOTIFY_DISCORD_ENABLED` | `notify.discord.enabled` |
| `KZERO_NOTIFY_DISCORD_WEBHOOK_URL` | `notify.discord.webhook_url` |
| `KZERO_NOTIFY_TEAMS_ENABLED` | `notify.teams.enabled` |
| `KZERO_NOTIFY_TEAMS_WEBHOOK_URL` | `notify.teams.webhook_url` |
| `KZERO_NOTIFY_PAGERDUTY_ENABLED` | `notify.pagerduty.enabled` |
| `KZERO_NOTIFY_PAGERDUTY_ROUTING_KEY` | `notify.pagerduty.routing_key` |
| `KZERO_NOTIFY_WEBHOOK_ENABLED` | `notify.webhook.enabled` |
| `KZERO_NOTIFY_WEBHOOK_URL` | `notify.webhook.url` |

Example: test Slack without editing YAML on disk:

```bash
export KZERO_NOTIFY_SLACK_ENABLED=true
export KZERO_NOTIFY_SLACK_WEBHOOK_URL="https://hooks.slack.com/services/…"
kzero notify test --config ./kzero.yaml
```

**Never commit webhook URLs or routing keys.** Use env vars, CI secrets, or a secret manager.

## Live pipeline workflow

Recommended order for a new profile:

```bash
# 1. Plan only
kzero analyze --config ./kzero.yaml

# 2. Verify notify channels (no API mutations)
kzero notify test --config ./kzero.yaml
kzero notify test -c ./kzero.yaml --event pipeline.error

# 3. Dry-run pipeline (no notify, no mutations)
kzero down --config ./kzero.yaml   # run.mode: dry-run

# 4. Live run (notify fires on start / success / error)
export KZERO_RUN_MODE=live
kzero reset --config ./kzero.yaml
```

See also [automation-and-pipelines.md](automation-and-pipelines.md) for CI/cron patterns.

## Operator audit fields in payloads

When configured, notifications include **`client_id`** from **`client.id`** and cluster metadata from **`cluster.name`**. The **`Kubernetes target:`** block also prints **`os_user`** / **`os_uid`** (hooks receive **`KZERO_OS_USER`** / **`KZERO_OS_UID`**). See [SPECIFICATIONS.md](../SPECIFICATIONS.md).

## Troubleshooting

| Symptom | Check |
|---------|--------|
| No message on `down` / `up` | **`run.mode`** must be **`live`**; **`dry-run`** skips pipeline notify |
| `notify test: no notify channel enabled` | At least one `*.enabled: true` or matching `KZERO_NOTIFY_*_ENABLED` |
| HTTP 4xx from webhook | URL, auth headers, and firewall egress |
| Duplicate error alerts | **`pipeline.error`** fires before **`on-error`** hook; hook may send its own alert |
| Secrets in logs | kzero redacts webhook URLs in notify **error messages**; keep URLs out of committed YAML |

Contract details: [SPECIFICATIONS.md](../SPECIFICATIONS.md) → **`notify`** and **`kzero notify test`**.
