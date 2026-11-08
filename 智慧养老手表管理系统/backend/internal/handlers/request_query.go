package handlers

import (
	"net/http"
	"strconv"

	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
)

// pagedTotal counts rows for a paginated query.
func (h *Handler) pagedTotal(w http.ResponseWriter, r *http.Request, sql string, args []any) (int64, bool) {
	var total int64
	if internalError(w, h.DB.QueryRow(r.Context(), "SELECT COUNT(*) FROM ("+sql+") AS paged_total", args...).Scan(&total)) {
		return 0, false
	}
	return total, true
}

// pageArgs 拼 LIMIT/OFFSET 占位符与完整参数。
func pageArgs(args []any, page, pageSize int) (string, []any) {
	limit := "$" + strconv.Itoa(len(args)+1)
	offset := "$" + strconv.Itoa(len(args)+2)
	full := make([]any, 0, len(args)+2)
	full = append(full, args...)
	full = append(full, pageSize, (page-1)*pageSize)
	return " LIMIT " + limit + " OFFSET " + offset, full
}

// rowsOK 检查 rows.Err()，有错写 500。
func rowsOK(w http.ResponseWriter, rows dbstore.Rows) bool {
	if err := rows.Err(); err != nil {
		rows.Close()
		response.Error500(w, ref())
		return false
	}
	return true
}
