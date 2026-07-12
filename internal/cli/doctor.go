package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var outputForm string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check config, binaries, API reachability, workloads, and RBAC hints",
		Long: `Run operator preflight checks without mutating the cluster.

Checks:
  - config YAML loads
  - kubectl/helm on PATH when run.execution is shell or auto
  - Kubernetes API handshake
  - pipeline deployment/statefulset/pvc/exec refs exist
  - SelfSubjectAccessReview hints for scale/delete verbs

Exit codes:
  0  all checks passed (warnings allowed)
  1  one or more errors
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("doctor config: %w", err)
			}
			applyCLIRunOverrides(cfg)

			ctx, cancel := context.WithTimeout(cmd.Context(), 45*time.Second)
			defer cancel()

			rep := doctor.Run(ctx, cfg, doctor.Options{})
			if err := renderDoctor(cmd, rep, strings.ToLower(strings.TrimSpace(outputForm))); err != nil {
				return err
			}
			if !rep.OK {
				return fmt.Errorf("doctor: checks failed")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&outputForm, "output", "text", "Output format: text or json")
	return cmd
}

func renderDoctor(cmd *cobra.Command, rep doctor.Report, format string) error {
	out := cmd.OutOrStdout()
	if format == "json" {
		b, err := json.MarshalIndent(rep, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, string(b))
		return err
	}

	verdict := "OK"
	if !rep.OK {
		verdict = "FAIL"
	}
	if _, err := fmt.Fprintf(out, "Doctor: %s\n", verdict); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "  findings: %d\n", len(rep.Findings)); err != nil {
		return err
	}
	for _, f := range rep.Findings {
		if _, err := fmt.Fprintf(out, "  - [%s] %s: %s\n", doctorSeverityLabel(f.Severity), f.Check, f.Message); err != nil {
			return err
		}
	}
	return nil
}

func doctorSeverityLabel(s doctor.Severity) string {
	switch s {
	case doctor.SeverityError:
		return "ERROR"
	case doctor.SeverityWarn:
		return "WARN "
	case doctor.SeverityOK:
		return "OK   "
	default:
		return "????"
	}
}
