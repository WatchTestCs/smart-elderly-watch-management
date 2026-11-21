package blob

import (
	"context"
	"database/sql"
	"errors"

	mysql "github.com/go-sql-driver/mysql"
)

// MySQL stores immutable encrypted objects alongside their business catalog.
// It is the only production blob backend; Local is retained for offline imports.
type MySQL struct{ db *sql.DB }

func NewMySQL(database *sql.DB) (*MySQL, error) {
	if database == nil {
		return nil, errors.New("MySQL image storage requires a database connection")
	}
	return &MySQL{db: database}, nil
}

func (s *MySQL) Put(ctx context.Context, key string, payload []byte) error {
	if !ValidKey(key) {
		return ErrNotFound
	}
	if len(payload) > MaxStoredBytes {
		return ErrIntegrity
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO image_payload(name,payload) VALUES(?,?)", key, payload)
	var duplicate *mysql.MySQLError
	if errors.As(err, &duplicate) && duplicate.Number == 1062 {
		return ErrExists
	}
	return err
}

type payloadReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func getPayload(ctx context.Context, reader payloadReader, key string) ([]byte, error) {
	if !ValidKey(key) {
		return nil, ErrNotFound
	}
	var data []byte
	err := reader.QueryRowContext(ctx, "SELECT payload FROM image_payload WHERE name=?", key).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(data) > MaxStoredBytes {
		return nil, ErrIntegrity
	}
	return data, nil
}

func (s *MySQL) Get(ctx context.Context, key string) ([]byte, error) {
	return getPayload(ctx, s.db, key)
}

// GetTx avoids acquiring another pooled connection while a business
// transaction is already holding the image reference lock.
func (s *MySQL) GetTx(ctx context.Context, tx *sql.Tx, key string) ([]byte, error) {
	return getPayload(ctx, tx, key)
}

func (s *MySQL) Delete(ctx context.Context, key string) error {
	if !ValidKey(key) {
		return ErrNotFound
	}
	_, err := s.db.ExecContext(ctx, "DELETE FROM image_payload WHERE name=?", key)
	return err
}

// Walk paginates without keeping a database connection while the callback
// makes its own MySQL queries, including transactional GC/catalog operations.
func (s *MySQL) Walk(ctx context.Context, visit func(Entry) error) error {
	var after string
	for {
		rows, err := s.db.QueryContext(ctx, "SELECT name,OCTET_LENGTH(payload),created_at FROM image_payload WHERE name>? ORDER BY name LIMIT 128", after)
		if err != nil {
			return err
		}
		var batch []Entry
		for rows.Next() {
			var item Entry
			if err = rows.Scan(&item.Key, &item.Size, &item.Modified); err != nil {
				break
			}
			batch = append(batch, item)
		}
		if err == nil {
			err = rows.Err()
		}
		if closeErr := rows.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
		for _, item := range batch {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := visit(item); err != nil {
				return err
			}
			after = item.Key
		}
		if len(batch) < 128 {
			return nil
		}
	}
}

// GetTx authenticates payloads using the business transaction's connection.
// This prevents a full pool from deadlocking the reference transaction.
func (s *Encrypted) GetTx(ctx context.Context, tx *sql.Tx, key string) ([]byte, error) {
	mysqlStore, ok := s.Store.(*MySQL)
	if !ok {
		return s.Get(ctx, key)
	}
	data, err := mysqlStore.GetTx(ctx, tx, key)
	if err != nil {
		return nil, err
	}
	return s.decode(key, data)
}

// putMySQLCatalog publishes ciphertext and integrity metadata atomically.
// A duplicate key never overwrites an immutable image or its catalog.
func (s *Encrypted) putMySQLCatalog(ctx context.Context, database *sql.DB, key string, data []byte) error {
	if !ValidKey(key) {
		return ErrNotFound
	}
	if len(data) > MaxPlainBytes {
		return ErrIntegrity
	}
	ciphertext, err := s.Box.Seal(string(data), "image:"+key)
	if err != nil {
		return err
	}
	if len(ciphertext) > MaxStoredBytes {
		return ErrIntegrity
	}
	tx, err := database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "INSERT INTO image_payload(name,payload) VALUES(?,?)", key, []byte(ciphertext)); err == nil {
		_, err = tx.ExecContext(ctx, "INSERT INTO image_blob(name,content_sha256,size_bytes,state) VALUES(?,?,?,'ready')", key, Digest(data), len(data))
	}
	if err != nil {
		var duplicate *mysql.MySQLError
		if errors.As(err, &duplicate) && duplicate.Number == 1062 {
			return ErrExists
		}
		return err
	}
	return tx.Commit()
}
