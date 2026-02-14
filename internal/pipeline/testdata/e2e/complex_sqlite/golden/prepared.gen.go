package complex

import (
	"context"
	"database/sql"
)

type PrepareDB interface {
	DBTX
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

type PreparedConfig struct{}

type PreparedQueries struct {
	queries             *Queries
	db                  PrepareDB
	closed              bool
	stmtDeleteTag       *sql.Stmt
	stmtGetItemWithTags *sql.Stmt
}

func (p *PreparedQueries) Raw() *Queries {
	return p.queries
}

func Prepare(ctx context.Context, db PrepareDB, cfg PreparedConfig) (*PreparedQueries, error) {
	pq := &PreparedQueries{
		queries: New(db),
		db:      db,
	}
	prepared := make([]*sql.Stmt, 0, 2)
	var stmt *sql.Stmt
	var err error
	stmt, err = db.PrepareContext(ctx, queryDeleteTag)
	if err != nil {
		for _, preparedStmt := range prepared {
			preparedStmt.Close()
		}
		return nil, err
	}
	prepared = append(prepared, stmt)
	pq.stmtDeleteTag = stmt
	stmt, err = db.PrepareContext(ctx, queryGetItemWithTags)
	if err != nil {
		for _, preparedStmt := range prepared {
			preparedStmt.Close()
		}
		return nil, err
	}
	prepared = append(prepared, stmt)
	pq.stmtGetItemWithTags = stmt
	return pq, nil
}

func (p *PreparedQueries) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	var err error
	if p.stmtDeleteTag != nil {
		if closeErr := p.stmtDeleteTag.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		p.stmtDeleteTag = nil
	}
	if p.stmtGetItemWithTags != nil {
		if closeErr := p.stmtGetItemWithTags.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		p.stmtGetItemWithTags = nil
	}
	return err
}

func (p *PreparedQueries) DeleteTag(ctx context.Context, tag string) error {
	stmt := p.stmtDeleteTag
	_, err := stmt.ExecContext(ctx, tag)
	return err
}

func (p *PreparedQueries) GetItemWithTags(ctx context.Context, id *int32) (GetItemWithTagsRow, error) {
	stmt := p.stmtGetItemWithTags
	rows, err := stmt.QueryContext(ctx, id)
	if err != nil {
		return GetItemWithTagsRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetItemWithTagsRow{}, err
		}
		return GetItemWithTagsRow{}, sql.ErrNoRows
	}
	item, err := scanGetItemWithTagsRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}
