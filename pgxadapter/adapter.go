package pgxadapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ddl = `
CREATE TABLE IF NOT EXISTS casbin_rule (
	id     SERIAL PRIMARY KEY,
	ptype  TEXT NOT NULL,
	v0     TEXT,
	v1     TEXT,
	v2     TEXT,
	v3     TEXT,
	v4     TEXT,
	v5     TEXT
);
-- optional but recommended for faster deletes / lookups
CREATE INDEX IF NOT EXISTS idx_casbin_rule
ON casbin_rule (ptype, v0, v1, v2, v3);
`

// compile-time assertions: if we later forget a method, 'go vet' will complain
var (
	_ persist.Adapter      = (*Adapter)(nil)
	_ persist.BatchAdapter = (*Adapter)(nil)
)

type Adapter struct {
	pool *pgxpool.Pool
}

// New creates an Adapter that reuses the caller-supplied pgx pool.
// It also runs the DDL once; remove the Exec if you prefer migrations.
func New(ctx context.Context, pool *pgxpool.Pool) (*Adapter, error) {
	if _, err := pool.Exec(ctx, ddl); err != nil {
		return nil, fmt.Errorf("create casbin_rule: %w", err)
	}
	return &Adapter{pool: pool}, nil
}

/* ------------ persist.Adapter interface ------------------ */

// LoadPolicy just delegates to the batch loader.
func (a *Adapter) LoadPolicy(m model.Model) error {
	return a.LoadPolicyBatch(m)
}

// SavePolicy would replace the entire table; not needed because we
// rely on AutoSave and the Add/Remove helpers.
func (a *Adapter) SavePolicy(_ model.Model) error {
	return fmt.Errorf("SavePolicy not implemented; use enforcer.EnableAutoSave(true)")
}

func (a *Adapter) AddPolicy(_ string, ptype string, rule []string) error {
	ctx := context.Background()

	// pad rule to 6 fields
	for len(rule) < 6 {
		rule = append(rule, "")
	}

	args := []interface{}{ptype}
	for _, v := range rule[:6] {
		args = append(args, v)
	}

	_, err := a.pool.Exec(ctx, `
		INSERT INTO casbin_rule (ptype,v0,v1,v2,v3,v4,v5)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT DO NOTHING
	`, args...)
	return err
}

func (a *Adapter) RemovePolicy(_ string, ptype string, rule []string) error {
	ctx := context.Background()

	for len(rule) < 6 {
		rule = append(rule, "")
	}

	args := []interface{}{ptype}
	for _, v := range rule[:6] {
		args = append(args, v)
	}

	_, err := a.pool.Exec(ctx, `
		DELETE FROM casbin_rule
		WHERE ptype=$1 AND v0=$2 AND v1=$3 AND v2=$4 AND v3=$5 AND v4=$6 AND v5=$7
	`, args...)
	return err
}

// RemoveFilteredPolicy deletes rules that match the supplied filter.
func (a *Adapter) RemoveFilteredPolicy(_ string, ptype string, fieldIndex int, fieldValues ...string) error {
	if fieldIndex < 0 || fieldIndex > 5 {
		return fmt.Errorf("fieldIndex must be 0..5")
	}

	ctx := context.Background()
	conds := []string{"ptype=$1"}
	args := []interface{}{ptype}

	for i, fv := range fieldValues {
		if fv == "" {
			continue
		}
		col := fmt.Sprintf("v%d", fieldIndex+i)
		conds = append(conds, fmt.Sprintf("%s=$%d", col, len(args)+1))
		args = append(args, fv)
	}

	sql := `DELETE FROM casbin_rule WHERE ` + strings.Join(conds, " AND ")
	_, err := a.pool.Exec(ctx, sql, args...)
	return err
}

/* ------------ persist.BatchAdapter interface ------------- */

func (a *Adapter) AddPolicies(sec, ptype string, rules [][]string) error {
	for _, rule := range rules {
		if err := a.AddPolicy(sec, ptype, rule); err != nil {
			return err
		}
	}
	return nil
}

func (a *Adapter) RemovePolicies(sec, ptype string, rules [][]string) error {
	for _, rule := range rules {
		if err := a.RemovePolicy(sec, ptype, rule); err != nil {
			return err
		}
	}
	return nil
}

func (a *Adapter) LoadPolicyBatch(m model.Model) error {
	rows, err := a.pool.Query(context.Background(),
		`SELECT ptype,v0,v1,v2,v3,v4,v5 FROM casbin_rule`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var rec [7]string
	for rows.Next() {
		if err := rows.Scan(&rec[0], &rec[1], &rec[2], &rec[3],
			&rec[4], &rec[5], &rec[6]); err != nil {
			return err
		}
		persist.LoadPolicyLine(strings.Join(rec[:], ", "), m)
	}
	return rows.Err()
}
