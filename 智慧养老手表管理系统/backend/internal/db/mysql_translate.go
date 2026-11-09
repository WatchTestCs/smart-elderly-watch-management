package db

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// This covers the application's SQL vocabulary, not arbitrary PostgreSQL SQL.
// Keep parameters bound (including repeated/out-of-order $n) and never interpolate values.
type queryPlan struct {
	sql          string
	positions    []int
	parameters   int
	requiredArgs int
	// direct means placeholders are exactly $1,$2,... in occurrence order.
	direct bool
}

// SQL structure only; request arguments never enter the shared cache.
var queryPlans = newQueryPlanCache()

func mysqlQuery(query string, args []any) (string, []any, error) {
	plan, err := queryPlans.load(query)
	if err != nil {
		return "", nil, err
	}
	if len(args) != plan.requiredArgs {
		return "", nil, fmt.Errorf("SQL argument count mismatch: query requires %d arguments, got %d", plan.requiredArgs, len(args))
	}
	if plan.direct {
		n := plan.parameters
		// Executors consume these arguments synchronously and never mutate
		// them. Limit capacity so appending cannot overwrite caller arguments.
		// No request values are retained in the shared SQL structure cache.
		return plan.sql, args[:n:n], nil
	}
	bound := make([]any, len(plan.positions))
	for i, position := range plan.positions {
		if position >= len(args) {
			return "", nil, errors.New("SQL parameter out of range")
		}
		bound[i] = args[position]
	}
	return plan.sql, bound, nil
}

func compileMySQLQuery(query string) (queryPlan, error) {
	// REGEXP 与 now() 交给分词器处理：它们必须只在引号之外生效。
	// 用 ReplaceAll 直接改整串会把字符串字面量里的内容一起改掉。
	out := make([]byte, 0, len(query)+32)
	var positions []int
	requiredArgs := 0
	for i := 0; i < len(query); {
		end, err := protectedSQLTokenEnd(query, i)
		if err != nil {
			return queryPlan{}, err
		}
		if end != i {
			out = append(out, query[i:end]...)
			i = end
			continue
		}
		// PostgreSQL regex operators are domain-owned. Reject a leaked operator
		// instead of silently producing MySQL with different semantics.
		if query[i] == '~' {
			return queryPlan{}, errors.New("PostgreSQL regex operator must be selected at the call site")
		}
		// now()：只在标识符边界处替换，避免动到 snow() 这类名字；
		// 同时因为已经进入分词循环，字符串字面量里的 now() 不会被改写。
		if query[i] == 'n' && strings.HasPrefix(query[i:], "now()") && (i == 0 || !identifierByte(query[i-1])) {
			out = append(out, "CURRENT_TIMESTAMP(6)"...)
			i += len("now()")
			continue
		}
		if query[i] == '$' && i+1 < len(query) && query[i+1] >= '0' && query[i+1] <= '9' {
			j := i + 1
			for j < len(query) && query[j] >= '0' && query[j] <= '9' {
				j++
			}
			n, err := strconv.Atoi(query[i+1 : j])
			if err != nil || n < 1 {
				return queryPlan{}, errors.New("SQL parameter out of range")
			}
			out = append(out, '?')
			positions = append(positions, n-1)
			if n > requiredArgs {
				requiredArgs = n
			}
			i = j
			continue
		}
		out = append(out, query[i])
		i++
	}
	direct := true
	for i, position := range positions {
		if position != i {
			direct = false
			break
		}
	}
	return queryPlan{sql: string(out), positions: positions, parameters: len(positions), requiredArgs: requiredArgs, direct: direct}, nil
}

// identifierByte 判断 c 是否可以作为标识符的一部分，用于避免替换到
// snow()、now_timestamp 这类以 now() 结尾/开头的名字里。
func identifierByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
