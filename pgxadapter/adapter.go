package pgxadapter

import (
	"context"
	"fmt"

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
`

type Adapter struct {
	pool *pgxpool.Pool
}

// New returns an adapter that reuses the caller-supplied pool.
// It executes the DDL once; drop that call if you prefer migrations.
func New(ctx context.Context, pool *pgxpool.Pool) (*Adapter, error) {
	if _, err := pool.Exec(ctx, ddl); err != nil {
		return nil, fmt.Errorf("create casbin_rule: %w", err)
	}
	return &Adapter{pool: pool}, nil
}

/* ------------ persist.Adapter interface ------------------ */

func (a *Adapter) LoadPolicy(_ model.Model) error {
	// Casbin uses a second interface, persist.BatchAdapter, for bulk loading;
	// we'll implement that instead for speed.
	return fmt.Errorf("LoadPolicy not supported, use BatchAdapter")
}

func (a *Adapter) SavePolicy(_ model.Model) error {
	// Casbin calls this to *replace* the whole table.
	// Because we enable AutoSave and handle Add/Remove directly,
	// we can return an error to force Casbin to fall back.
	return fmt.Errorf("SavePolicy: not implemented; enable AutoSave instead")
}

func (a *Adapter) AddPolicy(sec, ptype string, rule []string) error {
	ctx := context.Background()
	cols := make([]interface{}, 0, 7)
	cols = append(cols, ptype)
	for len(rule) < 6 { // pad to 6 vals
		rule = append(rule, "")
	}
	for _, v := range rule[:6] {
		cols = append(cols, v)
	}
	_, err := a.pool.Exec(ctx,
		`INSERT INTO casbin_rule (ptype,v0,v1,v2,v3,v4,v5)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`, cols...)
	return err
}

func (a *Adapter) RemovePolicy(sec, ptype string, rule []string) error {
	ctx := context.Background()
	sql := `DELETE FROM casbin_rule
	        WHERE ptype=$1 AND v0=$2 AND v1=$3 AND v2=$4 AND v3=$5 AND v4=$6 AND v5=$7`
	args := make([]interface{}, 0, 7)
	args = append(args, ptype)
	for len(rule) < 6 {
		rule = append(rule, "")
	}
	for _, v := range rule[:6] {
		args = append(args, v)
	}
	_, err := a.pool.Exec(ctx, sql, args...)
	return err
}

func (a *Adapter) RemoveFilteredPolicy(sec, ptype string, fieldIndex int, fieldValues ...string) error {
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
	sql := `DELETE FROM casbin_rule WHERE ` + join(conds, " AND ")
	_, err := a.pool.Exec(ctx, sql, args...)
	return err
}

/* ------------ persist.BatchAdapter interface ------------- */

func (a *Adapter) LoadPolicyBatch(m model.Model) error {
	rows, err := a.pool.Query(context.Background(),
		`SELECT ptype,v0,v1,v2,v3,v4,v5 FROM casbin_rule`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var rec [7]string
	for rows.Next() {
		if err := rows.Scan(&rec[0], &rec[1], &rec[2], &rec[3], &rec[4], &rec[5], &rec[6]); err != nil {
			return err
		}
		persist.LoadPolicyLine(join(rec[:], ", "), m)
	}
	return rows.Err()
}

/* ------------------- helpers ------------------- */

func join(strs []string, sep string) string {
	out := ""
	for i, s := range strs {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}
