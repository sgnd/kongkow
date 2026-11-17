package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/sealdb"
	"github.com/sgnd/kongkow/internal/store"
)

func audit(st *store.State, command string, ok bool, extra map[string]any) {
	if st != nil && !st.Settings.Audit {
		return
	}
	if g.dryRun {
		return
	}
	rec := map[string]any{
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		"command": command,
		"ok":      ok,
		"actor":   actor(),
	}
	if isAgent() {
		rec["agent"] = true
	}
	for k, v := range extra {
		rec[k] = v
	}
	_ = writeAudit(rec)
}

func writeAudit(rec map[string]any) error {
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	d, err := store.DB(true)
	if err != nil {
		return err
	}
	return d.Update(context.Background(), func(tx *sealdb.Tx) error {
		return tx.AppendAudit(b)
	})
}

type auditEntry struct {
	Seq  int64          `json:"seq"`
	Time time.Time      `json:"time"`
	Rec  map[string]any `json:"record"`
}

type auditResult struct {
	Entries []auditEntry `json:"entries"`
}

func (r auditResult) Text() string {
	if len(r.Entries) == 0 {
		return "No audit entries yet.\n"
	}
	var b strings.Builder
	for _, e := range r.Entries {
		cmd, _ := e.Rec["command"].(string)
		ok, _ := e.Rec["ok"].(bool)
		mark := "ok"
		if !ok {
			mark = "fail"
		}
		fmt.Fprintf(&b, "%d  %s  %s  %s\n", e.Seq, e.Time.Local().Format("2006-01-02 15:04:05"), mark, cmd)
	}
	return b.String()
}

func auditCmd() *cobra.Command {
	var limit int
	c := &cobra.Command{
		Use:   "audit",
		Short: "Show the local audit log",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if limit < 0 {
				return out.Usage("--limit cannot be negative")
			}
			entries, err := readAudit(limit)
			if err != nil {
				return err
			}
			out.P.Print("audit", auditResult{Entries: entries}, nil)
			return nil
		},
	}
	c.Flags().IntVar(&limit, "limit", 20, "how many newest entries to show (0 = all)")
	return c
}

func readAudit(limit int) ([]auditEntry, error) {
	d, err := store.DB(false)
	if errors.Is(err, store.ErrNoStore) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var last int64
	err = d.View(context.Background(), func(tx *sealdb.Tx) error {
		var err error
		last, err = tx.AuditLast()
		return err
	})
	if err != nil {
		return nil, err
	}
	if last == 0 {
		return nil, nil
	}
	after := int64(0)
	if limit > 0 && int64(limit) < last {
		after = last - int64(limit)
	}
	var rows []sealdb.AuditRow
	err = d.View(context.Background(), func(tx *sealdb.Tx) error {
		var err error
		rows, err = tx.Audit(after, int(last-after)+1)
		return err
	})
	if err != nil {
		return nil, err
	}
	out := make([]auditEntry, 0, len(rows))
	for _, r := range rows {
		var m map[string]any
		if err := json.Unmarshal(r.Data, &m); err != nil {
			m = map[string]any{"raw": string(r.Data)}
		}
		out = append(out, auditEntry{Seq: r.Seq, Time: r.Time, Rec: m})
	}
	// newest last in store; show newest first
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
