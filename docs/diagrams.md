# kzero diagrams (Mermaid)

Visual complements to [SPECIFICATIONS.md](SPECIFICATIONS.md). All diagrams describe the **v1** CLI and engine contract.

## 1. CLI and engine (high level)

`kzero` is a **CLI**: operators invoke subcommands; the **engine** loads YAML, validates, and runs phases.

**Live** mode mutates the cluster (or validates via server-side dry-run on the native path). **Dry-run** records the plan without persisting changes.

Workload and data steps use **`run.execution`**: **`shell`** (kubectl subprocess), **`native`** (client-go + Helm SDK + API delete/exec), or **`auto`** (native with shell fallback). Phase hooks and **`custom:`** steps always use **`/bin/sh`**.

```mermaid
flowchart LR
  subgraph Operator
    U[Operator]
  end
  subgraph CLI
    C[kzero CLI]
  end
  subgraph Engine
    E[Engine]
  end
  subgraph Outcomes
    D{{run.mode}}
    DR[dry-run: plan / log only]
    LV[live: native API / kubectl / hooks]
  end
  U --> C
  C --> E
  E --> D
  D -->|dry-run| DR
  D -->|live| LV
```

## 2. Subcommands

```mermaid
flowchart TB
  R[kzero]
  R --> A[analyze]
  R --> T[target]
  R --> N[notify test]
  R --> Pr[probe]
  R --> Vf[verify]
  R --> D[down]
  R --> U[up]
  R --> X[reset]
  R --> V[version]
  A --> P[validate + plan output]
  T --> KT[Kubernetes target audit]
  N --> NT[POST to enabled notify channels]
  Pr --> IP[infra_probe mini-pipeline]
  Vf --> VR[readiness report]
  D --> PD[phase: pipelines.down + hooks]
  U --> PU[phase: pipelines.up + hooks]
  X --> RU[down then up]
  RU --> PD
  RU --> PU
  V --> I[build metadata]
```

## 3. Phase hooks and one pipeline step (`down` or `up`)

Global phase hooks wrap the whole pipeline list. Each list item may run **per-step** `pre` → **main action** → `post` (see SPECIFICATIONS §3).

Main actions include **scale** (`deployment` / `statefulset`), **Helm release** (shell script or **Helm SDK**), **`pvc` delete**, **`exec` in pod**, and **`custom:`** scripts.

```mermaid
sequenceDiagram
  participant H as hooks.pre-down / pre-up
  participant S as Step 0..N
  participant T as hooks.post-down / post-up
  H->>S: run if configured
  loop Each pipeline step
    S->>S: optional step pre script
    S->>S: main action scale / release / pvc / exec / custom
    S->>S: optional step post script only if main ok
  end
  S->>T: run only if phase succeeded fail-fast
```

## 4. `reset` = full `down` then full `up`

If **down** fails, **up** must not run (SPECIFICATIONS §4 / §5). In **live** mode, an optional **`infra_probe`** gate may run before **down**; **`notify`** may fire on start / success / error.

```mermaid
flowchart TD
  Start([kzero reset]) --> Probe{infra_probe enabled?}
  Probe -->|live + before reset| Gate[Run probe up → checks → down]
  Probe -->|no| Down
  Gate -->|ok| Down[Execute down sequence]
  Gate -->|fail + fail_fast| Fail
  Down -->|success| Up[Execute up sequence]
  Down -->|failure| Fail([Exit non-zero up skipped])
  Up --> Done([Exit 0])
```

## 5. Fail-fast and `on-error` (simplified)

Any failing hook or step aborts the current phase. `hooks.on-error` runs per the failure policy; phase `post-*` hooks do not run after a failed phase.

```mermaid
flowchart TD
  A[Run next hook or step] -->|ok| B{More work in phase?}
  B -->|yes| A
  B -->|no| C[Run phase post-hook if configured]
  A -->|error| N[notify pipeline.error if enabled]
  N --> E[hooks.on-error]
  E --> F[Exit non-zero]
```

## 6. Configuration file (conceptual)

The process reads a single YAML file (`--config` / default `kzero.yaml`). Viper supplies **`KZERO_*`** env overrides for `run.*`, `notify.*`, `helm.workspace`, and related keys.

```mermaid
flowchart LR
  Y[kzero.yaml] --> L[Load + validate]
  E[KZERO_* env] --> L
  L --> G[Resolved Config]
  G --> Eng[Engine phases]
```

## 7. Workload execution backend (`run.execution`)

```mermaid
flowchart TD
  Step[deployment / statefulset / release step] --> Mode{run.execution}
  Mode -->|shell| Sh[kubectl scale / helm uninstall / release.sh]
  Mode -->|native| Nat[client-go scale + Helm SDK]
  Mode -->|auto| Try[Try native]
  Try -->|ok| Nat
  Try -->|client init fail| Sh
  PVC[pvc step] --> API[API delete claim]
  Exec[exec step] --> RC[remotecommand in pod]
```

---

To edit diagrams: change the fenced `mermaid` blocks in this file; keep prose in **English** per repository conventions.
