// Package guardianrepo owns guardian profile persistence.
package guardianrepo

import (
	"context"
	"database/sql"
	"errors"

	"suining-care/internal/db"
)

// Guardian is the typed profile returned by the guardian management API.
type Guardian struct {
	ID      int64   `json:"customerId"`
	Name    *string `json:"customerName"`
	Age     *int64  `json:"customerAge"`
	Sex     *string `json:"customerSex"`
	Phone   *string `json:"customerPhone"`
	Address *string `json:"customerAddress"`
}

type Fields struct {
	Name    string
	Age     int
	Sex     string
	Phone   string
	Address string
}

// List reads a page and its count together, including when the page is empty.
func List(ctx context.Context, store db.RowsQuerier, name string, page, pageSize int) ([]Guardian, int64, error) {
	const filter = ` FROM customer WHERE (?='' OR customer_name LIKE CONCAT('%',?,'%'))`
	const countSQL = `SELECT COUNT(*) AS total` + filter
	items := []Guardian{}
	var total int64
	err := db.QueryEach(ctx, store, func(rows *sql.Rows) error {
		var id *int64
		var item Guardian
		if err := rows.Scan(&id, &item.Name, &item.Age, &item.Sex, &item.Phone, &item.Address, &total); err != nil {
			return err
		}
		if id != nil {
			item.ID = *id
			items = append(items, item)
		}
		return nil
	}, `SELECT page.*, totals.total FROM (`+countSQL+`) AS totals LEFT JOIN (
		 SELECT customer_id,customer_name,customer_age,customer_sex,customer_phone,customer_address`+filter+`
		 ORDER BY customer_id LIMIT ? OFFSET ?) AS page
		 ON TRUE ORDER BY page.customer_id`, name, name, name, name, pageSize, (page-1)*pageSize)
	return items, total, err
}

func Create(ctx context.Context, store *sql.Tx, fields Fields) (int64, error) {
	result, err := store.ExecContext(ctx,
		`INSERT INTO customer(customer_name,customer_age,customer_sex,customer_phone,customer_address)
		 VALUES(?,?,?,?,?)`,
		fields.Name, fields.Age, fields.Sex, fields.Phone, fields.Address)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func Update(ctx context.Context, store *sql.Tx, id int64, fields Fields) (int64, error) {
	result, err := store.ExecContext(ctx,
		`UPDATE customer SET customer_name=?,customer_age=?,customer_sex=?,
		 customer_phone=?,customer_address=? WHERE customer_id=?`,
		fields.Name, fields.Age, fields.Sex, fields.Phone, fields.Address, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

var ErrGuardianNotFound = errors.New("guardian not found")
var ErrGuardianAssigned = errors.New("guardian has assigned residents")

// Delete locks the guardian before checking resident assignments and removing it.
func Delete(ctx context.Context, tx *sql.Tx, id int64) error {

	var found int64
	if err := tx.QueryRowContext(ctx, "SELECT customer_id FROM customer WHERE customer_id=? FOR UPDATE", id).Scan(&found); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGuardianNotFound
		}
		return err
	}
	var assigned bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM oldperson WHERE customer_id=?)", id).Scan(&assigned); err != nil {
		return err
	}
	if assigned {
		return ErrGuardianAssigned
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM customer WHERE customer_id=?", id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrGuardianNotFound
	}
	return nil
}
