# Reference hook scripts

Example **`pre`** / **`post`** shell hooks for operator-maintained configs moved to **[kzero-selfhosted](https://github.com/hrodrig/kzero-selfhosted)**:

| Script | Purpose |
|--------|---------|
| [wait-deployment-scale-down.sh](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/examples/hooks/wait-deployment-scale-down.sh) | `post` hook after scale-to-zero |
| [wait-helm-release-ready.sh](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/examples/hooks/wait-helm-release-ready.sh) | `post` hook after Helm install |
| [wait-master-ready.sh](https://github.com/hrodrig/kzero-selfhosted/blob/main/run/examples/hooks/wait-master-ready.sh) | `pre` hook: slave waits for master |

Walkthroughs that use these hooks remain here: [pipeline-order-and-integrity.md](../pipeline-order-and-integrity.md), [waiting-between-pipeline-steps.md](../waiting-between-pipeline-steps.md).
