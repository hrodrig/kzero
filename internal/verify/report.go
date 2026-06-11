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
	if err := log.WriteLine(w, log.LevelInfo, "Verify:"); err != nil {
		return err
	}
	if err := log.WriteLine(w, log.LevelInfo, "  outcome: "+r.Outcome); err != nil {
		return err
	}
	if r.Cluster != "" {
		if err := log.WriteLine(w, log.LevelInfo, "  cluster: "+r.Cluster); err != nil {
			return err
		}
	}
	if r.ClientID != "" {
		if err := log.WriteLine(w, log.LevelInfo, "  client_id: "+r.ClientID); err != nil {
			return err
		}
	}
	return nil
}

func printTextCheck(w io.Writer, c CheckResult) error {
	status := "FAIL"
	level := log.LevelWarn
	if c.OK {
		status = "OK"
		level = log.LevelInfo
	}
	if err := log.WriteLine(w, level, fmt.Sprintf("  %s  %s", status, c.Name)); err != nil {
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
	level := log.LevelWarn
	if item.OK {
		lineStatus = "OK"
		level = log.LevelInfo
	}
	ref := item.Ref
	if ref == "" {
		ref = checkName
	}
	if item.Detail != "" {
		return log.WriteLine(w, level, fmt.Sprintf("    %s  %s (%s)", lineStatus, ref, item.Detail))
	}
	return log.WriteLine(w, level, fmt.Sprintf("    %s  %s", lineStatus, ref))
}
