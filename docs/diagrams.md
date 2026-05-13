# kzero diagrams (Mermaid)

Visual complements to [SPECIFICATIONS.md](SPECIFICATIONS.md). All diagrams describe the **v1** CLI and engine contract.

## 1. CLI and engine (high level)

`kzero` is a **CLI**: operators invoke subcommands; the **engine** loads YAML, validates, and runs phases. **Live** mode executes external tools (`kubectl`, Helm helper scripts, shell hooks); **dry-run** records the plan without mutating the cluster.

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
    LV[live: kubectl / scripts / hooks]
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
  R --> D[down]
  R --> U[up]
  R --> X[reset]
  R --> V[version]
  A --> P[validate + plan output]
  D --> PD[phase: pipelines.down + hooks]
  U --> PU[phase: pipelines.up + hooks]
  X --> RU[down then up]
  RU --> PD
  RU --> PU
  V --> I[build metadata]
```

## 3. Phase hooks and one pipeline step (`down` or `up`)

Global phase hooks wrap the whole pipeline list. Each list item may run **per-step** `pre` → **main action** → `post` (see SPECIFICATIONS §3).

```mermaid
sequenceDiagram
  participant H as hooks.pre-down / pre-up
  participant S as Step 0..N
  participant T as hooks.post-down / post-up
  H->>S: run if configured
  loop Each pipeline step
    S->>S: optional step pre script
    S->>S: main action scale / rollout / release / custom
    S->>S: optional step post script only if main ok
  end
  S->>T: run only if phase succeeded fail-fast
```

## 4. `reset` = full `down` then full `up`

If **down** fails, **up** must not run (SPECIFICATIONS §4 / §5).

```mermaid
flowchart TD
  Start([kzero reset]) --> Down[Execute down sequence]
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
  A -->|error| E[hooks.on-error]
  E --> F[Exit non-zero]
```

## 6. Configuration file (conceptual)

The process reads a single YAML file (`--config` / default `kzero.yaml`). Viper supplies env overrides where implemented.

```mermaid
flowchart LR
  Y[kzero.yaml] --> L[Load + validate]
  L --> G[Resolved Config]
  G --> E[Engine phases]
```

---

To edit diagrams: change the fenced `mermaid` blocks in this file; keep prose in **English** per repository conventions.
