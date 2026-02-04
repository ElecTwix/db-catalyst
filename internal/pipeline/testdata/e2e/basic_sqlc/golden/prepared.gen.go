package basic

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
	queries               *Queries
	db                    PrepareDB
	closed                bool
	stmtCreateUser        *sql.Stmt
	stmtGetUser           *sql.Stmt
	stmtListPostsByAuthor *sql.Stmt
}

func (p *PreparedQueries) Raw() *Queries {
	return p.queries
}

func Prepare(ctx context.Context, db PrepareDB, cfg PreparedConfig) (*PreparedQueries, error) {
	pq := &PreparedQueries{
		queries: New(db),
		db:      db,
	}
	prepared := make([]*sql.Stmt, 0, 3)
	stmt, err := db.PrepareContext(ctx, queryCreateUser)
	if err != nil {
		for _, preparedStmt := range prepared {
			preparedStmt.Close()
		}
		return nil, err
	}
	prepared = append(prepared, stmt)
	pq.stmtCreateUser = stmt
	stmt, err := db.PrepareContext(ctx, queryGetUser)
	if err != nil {
		for _, preparedStmt := range prepared {
			preparedStmt.Close()
		}
		return nil, err
	}
	prepared = append(prepared, stmt)
	pq.stmtGetUser = stmt
	stmt, err := db.PrepareContext(ctx, queryListPostsByAuthor)
	if err != nil {
		for _, preparedStmt := range prepared {
			preparedStmt.Close()
		}
		return nil, err
	}
	prepared = append(prepared, stmt)
	pq.stmtListPostsByAuthor = stmt
	return pq, nil
}

func (p *PreparedQueries) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	var err error
	if p.stmtCreateUser != nil {
		if closeErr := p.stmtCreateUser.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		p.stmtCreateUser = nil
	}
	if p.stmtGetUser != nil {
		if closeErr := p.stmtGetUser.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		p.stmtGetUser = nil
	}
	if p.stmtListPostsByAuthor != nil {
		if closeErr := p.stmtListPostsByAuthor.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		p.stmtListPostsByAuthor = nil
	}
	return err
}

func (p *PreparedQueries) CreateUser(ctx context.Context, username string, email string) (CreateUserRow, error) {
	stmt := p.stmtCreateUser
	rows, err := stmt.QueryContext(ctx, username, email)
	if err != nil {
		return CreateUserRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return CreateUserRow{}, err
		}
		return CreateUserRow{}, sql.ErrNoRows
	}
	item, err := scanCreateUserRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}

func (p *PreparedQueries) GetUser(ctx context.Context, id int32) (GetUserRow, error) {
	stmt := p.stmtGetUser
	rows, err := stmt.QueryContext(ctx, id)
	if err != nil {
		return GetUserRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return GetUserRow{}, err
		}
		return GetUserRow{}, sql.ErrNoRows
	}
	item, err := scanGetUserRow(rows)
	if err != nil {
		return item, err
	}
	if err := rows.Err(); err != nil {
		return item, err
	}
	return item, nil
}

func (p *PreparedQueries) ListPostsByAuthor(ctx context.Context, authorId int32) ([]ListPostsByAuthorRow, error) {
	stmt := p.stmtListPostsByAuthor
	rows, err := stmt.QueryContext(ctx, authorId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ListPostsByAuthorRow, 0)
	for rows.Next() {
		item, err := scanListPostsByAuthorRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
