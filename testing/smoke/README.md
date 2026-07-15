# CI smoke tests

Fast binary smoke without kind or a live cluster:

- `make build`
- `kzero analyze` on [kzero-smoke.yaml](kzero-smoke.yaml)
- `kzero down` in dry-run mode
- `kzero --print-sample-config`

Run locally:

```bash
bash testing/smoke/smoke.sh
```

Product kind CI gate: [../kind/README.md](../kind/README.md). Full multi-workload lab: [kzero-selfhosted](https://github.com/hrodrig/kzero-selfhosted/tree/main/testing/kind).
