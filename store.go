package media

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"sync"

	"github.com/togo-framework/togo"
)

// Store persists media metadata. The plugin uses a SQL-backed store when a DB
// is reachable, and an in-memory store otherwise.
type Store interface {
	Save(ctx context.Context, m *Media) error
	List(ctx context.Context, modelType, modelID, collection string) ([]*Media, error)
	Get(ctx context.Context, id string) (*Media, error)
	Delete(ctx context.Context, id string) error
}

// ---- in-memory store ----

type memStore struct {
	mu sync.RWMutex
	m  map[string]*Media
}

func newMemStore() *memStore { return &memStore{m: map[string]*Media{}} }

func (s *memStore) Save(_ context.Context, m *Media) error {
	s.mu.Lock()
	cp := *m
	s.m[m.ID] = &cp
	s.mu.Unlock()
	return nil
}

func (s *memStore) List(_ context.Context, mt, mid, coll string) ([]*Media, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Media
	for _, m := range s.m {
		if m.ModelType == mt && m.ModelID == mid && (coll == "" || m.Collection == coll) {
			cp := *m
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *memStore) Get(_ context.Context, id string) (*Media, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.m[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *m
	return &cp, nil
}

func (s *memStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	delete(s.m, id)
	s.mu.Unlock()
	return nil
}

// ---- SQL store ----

type sqlStore struct {
	db *sql.DB
	ph func(n int) string // dialect placeholder
}

func newSQLStore(k *togo.Kernel) (*sqlStore, error) {
	db, err := k.SQL(context.Background())
	if err != nil {
		return nil, err
	}
	ddl := `CREATE TABLE IF NOT EXISTS media (
		id text PRIMARY KEY,
		model_type text NOT NULL DEFAULT '',
		model_id text NOT NULL DEFAULT '',
		collection text NOT NULL DEFAULT '',
		disk text NOT NULL DEFAULT '',
		path text NOT NULL,
		name text NOT NULL DEFAULT '',
		mime text NOT NULL DEFAULT '',
		size bigint NOT NULL DEFAULT 0,
		conversions text NOT NULL DEFAULT '',
		custom text NOT NULL DEFAULT '',
		created_at text NOT NULL)`
	if _, err := db.ExecContext(context.Background(), ddl); err != nil {
		return nil, err
	}
	return &sqlStore{db: db, ph: k.Dialect().Placeholder}, nil
}

func (s *sqlStore) Save(ctx context.Context, m *Media) error {
	conv, _ := json.Marshal(m.Conversions)
	custom, _ := json.Marshal(m.Custom)
	q := `INSERT INTO media (id, model_type, model_id, collection, disk, path, name, mime, size, conversions, custom, created_at) VALUES (` +
		s.ph(1) + `,` + s.ph(2) + `,` + s.ph(3) + `,` + s.ph(4) + `,` + s.ph(5) + `,` + s.ph(6) + `,` + s.ph(7) + `,` + s.ph(8) + `,` + s.ph(9) + `,` + s.ph(10) + `,` + s.ph(11) + `,` + s.ph(12) + `)`
	_, err := s.db.ExecContext(ctx, q, m.ID, m.ModelType, m.ModelID, m.Collection, m.Disk, m.Path, m.Name, m.Mime, m.Size, string(conv), string(custom), m.CreatedAt.Format(rfc3339))
	return err
}

func (s *sqlStore) List(ctx context.Context, mt, mid, coll string) ([]*Media, error) {
	q := `SELECT id, model_type, model_id, collection, disk, path, name, mime, size, conversions, custom, created_at FROM media WHERE model_type=` + s.ph(1) + ` AND model_id=` + s.ph(2)
	args := []any{mt, mid}
	if coll != "" {
		q += ` AND collection=` + s.ph(3)
		args = append(args, coll)
	}
	q += ` ORDER BY created_at`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Media
	for rows.Next() {
		m, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *sqlStore) Get(ctx context.Context, id string) (*Media, error) {
	q := `SELECT id, model_type, model_id, collection, disk, path, name, mime, size, conversions, custom, created_at FROM media WHERE id=` + s.ph(1)
	return scan(s.db.QueryRowContext(ctx, q, id))
}

func (s *sqlStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM media WHERE id=`+s.ph(1), id)
	return err
}

const rfc3339 = "2006-01-02T15:04:05.999999999Z07:00"

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scan(sc scanner) (*Media, error) {
	var (
		m              Media
		conv, custom   string
		created        string
	)
	if err := sc.Scan(&m.ID, &m.ModelType, &m.ModelID, &m.Collection, &m.Disk, &m.Path, &m.Name, &m.Mime, &m.Size, &conv, &custom, &created); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(conv), &m.Conversions)
	_ = json.Unmarshal([]byte(custom), &m.Custom)
	m.CreatedAt, _ = parseTime(created)
	return &m, nil
}
