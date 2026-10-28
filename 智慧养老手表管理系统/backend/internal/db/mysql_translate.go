package db

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var intervals = regexp.MustCompile(`(?i)interval\s+'(\d+)\s+(seconds?|minutes?|hours?|days?)'`)
var conflictUpdate = regexp.MustCompile(`(?i)ON\s+CONFLICT\s*\([^)]*\)\s+DO\s+UPDATE\s+SET`)
var conflictNothing = regexp.MustCompile(`(?i)ON\s+CONFLICT(?:\s*\([^)]*\))?\s+DO\s+NOTHING`)
var excludedColumn = regexp.MustCompile(`(?i)excluded\.([a-z_]+)`)
var lockLimit = regexp.MustCompile(`(?i)FOR UPDATE SKIP LOCKED\s+LIMIT\s+(\d+)`)
var textTable = regexp.MustCompile("(?i)\\b(FROM|JOIN|UPDATE|INTO)\\s+text\\b")
var dateFormat = regexp.MustCompile(`(?i)TO_CHAR\(([a-z_.]+),'(YYYY-MM-DD(?: HH24:MI:SS(?:\.MS)?)?)'\)`)
var boolOr = regexp.MustCompile(`(?i)bool_or\(`)
var varcharCast = regexp.MustCompile(`(?i)\bAS varchar\b`)

// This covers the application's SQL vocabulary, not arbitrary PostgreSQL SQL.
// Keep parameters bound (including repeated/out-of-order $n) and never interpolate values.
type queryPlan struct {
	sql        string
	positions  []int
	parameters int
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
	if plan.direct {
		n := plan.parameters
		if n > len(args) {
			return "", nil, errors.New("SQL parameter out of range")
		}
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
	query = dateFormat.ReplaceAllStringFunc(query, func(s string) string {
		m := dateFormat.FindStringSubmatch(s)
		format := "%Y-%m-%d"
		if strings.Contains(strings.ToUpper(m[2]), "HH24") {
			format += " %H:%i:%s"
		}
		if strings.HasSuffix(strings.ToUpper(m[2]), ".MS") {
			return "LEFT(DATE_FORMAT(" + m[1] + ",'" + format + ".%f'),23)"
		}
		return "DATE_FORMAT(" + m[1] + ",'" + format + "')"
	})
	query = varcharCast.ReplaceAllString(query, "AS CHAR")
	const latestPosition = "GREATEST(COALESCE(l.be_time, o.be_time), COALESCE(a.be_time, o.be_time))"
	query = strings.ReplaceAll(query, latestPosition, "COALESCE("+latestPosition+",COALESCE(l.be_time,o.be_time),COALESCE(a.be_time,o.be_time))")
	query = intervals.ReplaceAllStringFunc(query, func(s string) string {
		m := intervals.FindStringSubmatch(s)
		return "INTERVAL " + m[1] + " " + strings.TrimSuffix(strings.ToUpper(m[2]), "S")
	})
	query = strings.ReplaceAll(query, "now()+($3 * INTERVAL 1 SECOND)", "TIMESTAMPADD(SECOND,$3,now())")
	query = boolOr.ReplaceAllString(query, "max(")
	query = lockLimit.ReplaceAllString(query, "LIMIT $1 FOR UPDATE SKIP LOCKED")
	query = conflictUpdate.ReplaceAllString(query, "ON DUPLICATE KEY UPDATE")
	query = excludedColumn.ReplaceAllString(query, "VALUES($1)")
	if conflictNothing.MatchString(query) {
		key := "id"
		if strings.Contains(query, "INSERT INTO watch_info") {
			key = "device_id"
		}
		query = conflictNothing.ReplaceAllString(query, "ON DUPLICATE KEY UPDATE "+key+"="+key)
	}
	query = textTable.ReplaceAllString(query, "$1 `text`")
	// REGEXP 与 now() 交给分词器处理：它们必须只在引号之外生效。
	// 用 ReplaceAll 直接改整串会把字符串字面量里的内容一起改掉。
	out := make([]byte, 0, len(query)+32)
	var positions []int
	for i := 0; i < len(query); {
		if query[i] == '\'' || query[i] == '"' || query[i] == '`' {
			quote := query[i]
			out = append(out, quote)
			i++
			for i < len(query) {
				c := query[i]
				out = append(out, c)
				i++
				if c == '\\' && i < len(query) {
					out = append(out, query[i])
					i++
					continue
				}
				if c == quote {
					if i < len(query) && query[i] == quote {
						out = append(out, query[i])
						i++
						continue
					}
					break
				}
			}
			continue
		}
		// PostgreSQL 的正则匹配操作符。MySQL 用 REGEXP，间距不敏感；
		// 唯一的静默错误来源是 x~'...'（无空格）——旧实现只替换 " ~ "，
		// 这种写法会被 MySQL 当成按位取反，不报错但结果是错的。
		if query[i] == '~' {
			if i+1 < len(query) && (query[i+1] == '~' || query[i+1] == '*') {
				return queryPlan{}, errors.New("不支持的 PostgreSQL 正则操作符（~~ 或 ~*），请改写为 REGEXP")
			}
			out = trimRightSpace(out)
			if len(out) > 0 {
				out = append(out, ' ')
			}
			out = append(out, "REGEXP"...)
			i++
			for i < len(query) && query[i] == ' ' {
				i++
			}
			if i < len(query) {
				out = append(out, ' ')
			}
			continue
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
	return queryPlan{sql: string(out), positions: positions, parameters: len(positions), direct: direct}, nil
}

// trimRightSpace 去掉尾部空格，让 REGEXP 的输出保持单一空格分隔。
func trimRightSpace(b []byte) []byte {
	for len(b) > 0 && b[len(b)-1] == ' ' {
		b = b[:len(b)-1]
	}
	return b
}

// identifierByte 判断 c 是否可以作为标识符的一部分，用于避免替换到
// snow()、now_timestamp 这类以 now() 结尾/开头的名字里。
func identifierByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
