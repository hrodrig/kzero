package verify

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hrodrig/kzero/internal/log"
)

// Print writes the report to w using text or JSON format.
func Print(w io.Writer, format log.Format, r Report) error {
	if format == log.FormatJSON {
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return fmt.Errorf("verify: encode report: %w", err)
		}
		_, err = fmt.Fprintln(w, string(data))
		return err
	}
	return printText(w, r)
}

func printText(w io.Writer, r Report) error {
	if err := printTextHeader(w, r); err != nil {
		return err
	}
	for _, c := range r.Checks {
		if err := printTextCheck(w, c); err != nil {
			return err
		}
	}
	return nil
}

func printTextHeader(w io.Writer, r Report) error {
	if _, err := fmt.Fprintln(w, "Verify:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  outcome: %s\n", r.Outcome); err != nil {
		return err
	}
	if r.Cluster != "" {
		if _, err := fmt.Fprintf(w, "  cluster: %s\n", r.Cluster); err != nil {
			return err
		}
	}
	if r.ClientID != "" {
		if _, err := fmt.Fprintf(w, "  client_id: %s\n", r.ClientID); err != nil {
			return err
		}
	}
	return nil
}

func printTextCheck(w io.Writer, c CheckResult) error {
	status := "FAIL"
	if c.OK {
		status = "OK"
	}
	if _, err := fmt.Fprintf(w, "  %s  %s\n", status, c.Name); err != nil {
		return err
	}
	for _, item := range c.Items {
		if err := printTextItem(w, c.Name, item); err != nil {
			return err
		}
	}
	return nil
}

func printTextItem(w io.Writer, checkName string, item Item) error {
	lineStatus := "FAIL"
	if item.OK {
		lineStatus = "OK"
	}
	ref := item.Ref
	if ref == "" {
		ref = checkName
	}
	if item.Detail != "" {
		_, err := fmt.Fprintf(w, "    %s  %s (%s)\n", lineStatus, ref, item.Detail)
		return err
	}
	_, err := fmt.Fprintf(w, "    %s  %s\n", lineStatus, ref)
	return err
}
