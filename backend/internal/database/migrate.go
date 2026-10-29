package database

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Migrator applies SQL migrations from a directory. Migrations are embedded
// when built with go:embed; fallback to filesystem is provided for CLI use.
type Migrator struct {
	conn    *pgx.Conn
	files   []MigrationFile
	table   string
}

type MigrationFile struct {
	Version int64
	Name    string
	SQL     string
}

func NewMigrator(ctx context.Context, dsn string, files []MigrationFile) (*Migrator, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect for migrations: %w", err)
	}
	m := &Migrator{conn: conn, files: files, table: "schema_migrations"}
	return m, nil
}

func (m *Migrator) Close() { m.conn.Close(context.Background()) }

func (m *Migrator) ensureTable(ctx context.Context) error {
	_, err := m.conn.Exec(ctx, fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (version BIGINT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())", m.table))
	return err
}

func (m *Migrator) applied(ctx context.Context) (map[int64]bool, error) {
	rows, err := m.conn.Query(ctx, fmt.Sprintf("SELECT version FROM %s", m.table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]bool{}
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func (m *Migrator) Up(ctx context.Context) error {
	if err := m.ensureTable(ctx); err != nil {
		return err
	}
	applied, err := m.applied(ctx)
	if err != nil {
		return err
	}
	ordered := make([]MigrationFile, len(m.files))
	copy(ordered, m.files)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Version < ordered[j].Version })

	for _, f := range ordered {
		if applied[f.Version] {
			continue
		}
		if err := m.apply(ctx, f); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrator) Down(ctx context.Context) error {
	if err := m.ensureTable(ctx); err != nil {
		return err
	}
	applied, err := m.applied(ctx)
	if err != nil {
		return err
	}
	ordered := make([]MigrationFile, len(m.files))
	copy(ordered, m.files)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Version < ordered[j].Version })
	// Roll back the most recently applied migration by removing its version marker.
	for i := len(ordered) - 1; i >= 0; i-- {
		f := ordered[i]
		if !applied[f.Version] {
			continue
		}
		if _, err := m.conn.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE version = $1", m.table), f.Version); err != nil {
			return err
		}
		fmt.Printf("rolled back migration marker %d_%s\n", f.Version, f.Name)
		return nil
	}
	fmt.Println("no migrations to roll back")
	return nil
}

func (m *Migrator) apply(ctx context.Context, f MigrationFile) error {
	tx, err := m.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, f.SQL); err != nil {
		return fmt.Errorf("migration %d_%s: %w", f.Version, f.Name, err)
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("INSERT INTO %s (version) VALUES ($1)", m.table), f.Version); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Printf("applied migration %d_%s\n", f.Version, f.Name)
	return nil
}

func (m *Migrator) Status(ctx context.Context) error {
	if err := m.ensureTable(ctx); err != nil {
		return err
	}
	applied, err := m.applied(ctx)
	if err != nil {
		return err
	}
	ordered := make([]MigrationFile, len(m.files))
	copy(ordered, m.files)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Version < ordered[j].Version })
	for _, f := range ordered {
		status := "not applied"
		if applied[f.Version] {
			status = "applied"
		}
		fmt.Printf("%d %-30s %s\n", f.Version, f.Name, status)
	}
	return nil
}

// LoadMigrationFiles reads *.sql files from a directory using a naming convention
// of NN_name.sql where NN is a numeric version.
func LoadMigrationFiles(dir string) ([]MigrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []MigrationFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(e.Name(), "_", 2)
		var version int64
		fmt.Sscanf(parts[0], "%d", &version)
		name := strings.TrimSuffix(e.Name(), ".sql")
		if len(parts) > 1 {
			name = strings.TrimSuffix(parts[1], ".sql")
		}
		b, err := os.ReadFile(dir + "/" + e.Name())
		if err != nil {
			return nil, err
		}
		files = append(files, MigrationFile{Version: version, Name: name, SQL: string(b)})
	}
	return files, nil
}
