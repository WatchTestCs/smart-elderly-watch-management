package db

import "context"

// RowQuerier is the read-only query capability shared by databases and transactions.
type RowQuerier interface {
	QueryRow(context.Context, string, ...any) Row
}

type rowsQuerier interface {
	Query(context.Context, string, ...any) (Rows, error)
}

// QueryEach visits rows in query order and closes the result before returning.
func QueryEach(ctx context.Context, q rowsQuerier, visit func(Rows) error, query string, args ...any) error {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := visit(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// QueryList scans all rows and closes the result before returning to the caller.
func QueryList[T any](ctx context.Context, q rowsQuerier, scan func(Rows) (T, error), query string, args ...any) ([]T, error) {
	items := []T{}
	err := QueryEach(ctx, q, func(rows Rows) error {
		item, err := scan(rows)
		if err == nil {
			items = append(items, item)
		}
		return err
	}, query, args...)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func ScanValue[T any](row Rows) (T, error) {
	var value T
	err := row.Scan(&value)
	return value, err
}
