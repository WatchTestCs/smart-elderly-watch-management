package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// MySQL checks duplicate CHECK names before duplicate columns for an atomic
// multi-action ALTER. Do not globally ignore error 3822: verify the complete
// known expansion before skipping it after a crash or a metadata-only rollback.
func ensureUserPhoneExpansion(ctx context.Context, conn *sql.Conn, statement string) error {
	present, err := userPhoneExpansionPresent(ctx, conn)
	if err != nil || present {
		return err
	}
	if _, err := conn.ExecContext(ctx, statement); err != nil {
		return err
	}
	present, err = userPhoneExpansionPresent(ctx, conn)
	if err != nil {
		return err
	}
	if !present {
		return errors.New("administrator phone expansion did not establish its required schema")
	}
	return nil
}

func userPhoneExpansionPresent(ctx context.Context, conn *sql.Conn) (present bool, err error) {
	rows, err := conn.QueryContext(ctx, `SELECT COLUMN_NAME,COLUMN_TYPE,IS_NULLABLE,COLUMN_DEFAULT,COLLATION_NAME,EXTRA
 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='users'
 AND COLUMN_NAME IN ('user_phone_cipher','user_phone_lookup','user_phone_protected')`)
	if err != nil {
		return false, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			present, err = false, closeErr
		}
	}()
	count := 0
	for rows.Next() {
		var name, kind, nullable, extra string
		var initial, collation *string
		if err := rows.Scan(&name, &kind, &nullable, &initial, &collation, &extra); err != nil {
			return false, err
		}
		valid := extra == ""
		switch name {
		case "user_phone_cipher":
			valid = valid && kind == "text" && nullable == "YES" && initial == nil && collation != nil && *collation == "utf8mb4_bin"
		case "user_phone_lookup":
			valid = valid && kind == "char(64)" && nullable == "YES" && initial == nil && collation != nil && *collation == "ascii_bin"
		case "user_phone_protected":
			valid = valid && kind == "tinyint(1)" && nullable == "NO" && initial != nil && *initial == "0" && collation == nil
		default:
			valid = false
		}
		if !valid {
			return false, errors.New("administrator phone expansion has incompatible columns; refusing to skip or repair it")
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	if err := rows.Close(); err != nil {
		return false, err
	}
	if count == 0 {
		return false, nil
	}
	if count != 3 {
		return false, errors.New("administrator phone expansion is incomplete; refusing to skip or repair it")
	}
	var clause, enforced string
	if err := conn.QueryRowContext(ctx, `SELECT cc.CHECK_CLAUSE,tc.ENFORCED FROM information_schema.TABLE_CONSTRAINTS tc
 JOIN information_schema.CHECK_CONSTRAINTS cc ON cc.CONSTRAINT_SCHEMA=tc.CONSTRAINT_SCHEMA AND cc.CONSTRAINT_NAME=tc.CONSTRAINT_NAME
 WHERE tc.TABLE_SCHEMA=DATABASE() AND tc.TABLE_NAME='users' AND tc.CONSTRAINT_TYPE='CHECK' AND tc.CONSTRAINT_NAME='ck_users_phone_storage'`).Scan(&clause, &enforced); err != nil {
		return false, errors.New("administrator phone storage constraint is missing or unreadable")
	}
	const expected = "(((`user_phone_protected` = false) and (`user_phone_cipher` is null) and (`user_phone_lookup` is null)) or " +
		"((`user_phone_protected` = true) and (`user_phone` is null) and (((`user_phone_cipher` is null) and (`user_phone_lookup` is null)) or " +
		"((`user_phone_cipher` is not null) and (`user_phone_lookup` is not null) and (char_length(`user_phone_lookup`) = 64)))))"
	normalized, err := normalizePhoneCheck(clause)
	if err != nil {
		return false, err
	}
	want, err := normalizePhoneCheck(expected)
	if err != nil {
		return false, err
	}
	if enforced != "YES" || normalized != want {
		return false, errors.New("administrator phone storage constraint differs or is not enforced")
	}
	indexes, err := conn.QueryContext(ctx, `SELECT INDEX_NAME,NON_UNIQUE,SEQ_IN_INDEX,COLUMN_NAME,SUB_PART,IS_VISIBLE,EXPRESSION
 FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='users'
 AND INDEX_NAME IN ('uk_users_phone_lookup','ix_users_phone_protected')`)
	if err != nil {
		return false, err
	}
	defer func() {
		if closeErr := indexes.Close(); closeErr != nil && err == nil {
			present, err = false, closeErr
		}
	}()
	count = 0
	for indexes.Next() {
		var name, column, visible string
		var nonUnique, sequence int
		var prefix *int64
		var expression *string
		if err := indexes.Scan(&name, &nonUnique, &sequence, &column, &prefix, &visible, &expression); err != nil {
			return false, err
		}
		valid := prefix == nil && expression == nil && sequence == 1 && visible == "YES"
		if name == "uk_users_phone_lookup" {
			valid = valid && nonUnique == 0 && column == "user_phone_lookup"
		} else {
			valid = valid && nonUnique == 1 && column == "user_phone_protected"
		}
		if !valid {
			return false, errors.New("administrator phone expansion indexes differ")
		}
		count++
	}
	if err := indexes.Err(); err != nil {
		return false, err
	}
	if count != 2 {
		return false, errors.New("administrator phone expansion indexes are incomplete")
	}
	return true, nil
}

// Ignore code formatting only. Keep every parenthesis/operator and quoted byte;
// removing parentheses or whitespace inside identifiers could hide a weaker CHECK.
func normalizePhoneCheck(clause string) (string, error) {
	return rewriteSQLCodeSegments(clause, func(code string) (string, error) {
		return strings.Map(func(r rune) rune {
			if r < 128 && sqlCommentSpace(byte(r)) {
				return -1
			}
			if r >= 'A' && r <= 'Z' {
				return r + ('a' - 'A')
			}
			return r
		}, code), nil
	})
}
