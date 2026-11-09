package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
)

func listLatestMeasurements[T any](h *Handler, w http.ResponseWriter, r *http.Request, query, key string, scan func(dbstore.Rows) (T, error)) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	page, pageSize, ok := strictPage(w, m)
	if !ok {
		return
	}
	name := str(m, "name")
	timeFirst := str(m, "timeFirst")
	timeLast := str(m, "timeLast")
	sql := "SELECT * FROM (" + query + ") t WHERE 1=1"
	var args []any
	if strings.TrimSpace(name) != "" {
		sql += " AND t.name LIKE $" + strconv.Itoa(len(args)+1)
		args = append(args, "%"+name+"%")
	}
	if strings.TrimSpace(timeFirst) != "" && strings.TrimSpace(timeLast) != "" {
		sql += " AND t.createTime BETWEEN $" + strconv.Itoa(len(args)+1) + " AND $" + strconv.Itoa(len(args)+2)
		args = append(args, timeFirst, timeLast)
	}
	sql += " ORDER BY t.createTime DESC, t.oldId"
	total, ok := h.pagedTotal(w, r, sql, args)
	if !ok {
		return
	}
	limitSQL, fullArgs := pageArgs(args, page, pageSize)
	items, ok := queryRows(h, w, r.Context(), scan, sql+limitSQL, fullArgs...)
	if !ok {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		key: items, "total": total, "page": page, "pageSize": pageSize,
	})
}

func queryRows[T any](h *Handler, w http.ResponseWriter, ctx context.Context, scan func(dbstore.Rows) (T, error), sql string, args ...any) ([]T, bool) {
	items, err := dbstore.QueryList(ctx, h.DB, scan, sql, args...)
	return items, !internalError(w, err)
}
