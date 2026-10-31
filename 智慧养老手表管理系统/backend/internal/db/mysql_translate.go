package db

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var intervals = regexp.MustCompile(`(?i)^INTERVAL\s+'(\d+)\s+(seconds?|minutes?|hours?|days?)'`)
var hardcodedTimeOffset = regexp.MustCompile(`(?i)^now\(\)\s*\+\s*\(\s*\$(\d+)\s*\*\s*interval\s+'1 second'\s*\)`)

var conflictUpdate = regexp.MustCompile(`(?i)ON\s+CONFLICT\s*\([^)]*\)\s+DO\s+UPDATE\s+SET`)
var conflictNothing = regexp.MustCompile(`(?i)ON\s+CONFLICT(?:\s*\([^)]*\))?\s+DO\s+NOTHING`)
var excludedColumn = regexp.MustCompile(`(?i)excluded\.([a-z][a-z0-9_]*)`)
var firstInsertColumn = regexp.MustCompile(`(?is)^\s*INSERT\s+INTO\s+(?:\x60[^\x60]+\x60|"[^"]+"|[a-z_][a-z0-9_]*)(?:\s*\.\s*(?:\x60[^\x60]+\x60|"[^"]+"|[a-z_][a-z0-9_]*))?\s*\(\s*(\x60[^\x60]+\x60|"[^"]+"|[a-z_][a-z0-9_]*)`)
var lockLimit = regexp.MustCompile(`(?i)FOR UPDATE SKIP LOCKED\s+LIMIT\s+(\d+)`)
var dateFormat = regexp.MustCompile(`(?i)^TO_CHAR\(([a-z_.]+),'(YYYY-MM-DD(?: HH24:MI:SS(?:\.MS)?)?)'\)`)
var varcharCast = regexp.MustCompile(`(?i)\bAS varchar\b`)

// This covers the application's SQL vocabulary, not arbitrary PostgreSQL SQL.
// Keep parameters bound (including repeated/out-of-order $n) and never interpolate values.
type queryPlan struct {
	sql           string
	positions     []int
	parameters    int
	requiredArgs  int
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
	var err error
	query, err = rewriteDialectExpressions(query)
	if err != nil {
		return queryPlan{}, err
	}
	query, err = rewriteSQLCodeSegments(query, func(code string) (string, error) {
		code = varcharCast.ReplaceAllString(code, "AS CHAR")
		code = lockLimit.ReplaceAllString(code, "LIMIT $1 FOR UPDATE SKIP LOCKED")
		code = conflictUpdate.ReplaceAllString(code, "ON DUPLICATE KEY UPDATE")
		code = excludedColumn.ReplaceAllString(code, "VALUES($1)")
		if conflictNothing.MatchString(code) {
			match := firstInsertColumn.FindStringSubmatch(query)
			if len(match) != 2 {
				return "", errors.New("ON CONFLICT DO NOTHING requires an explicit INSERT column list for MySQL translation")
			}
			column := strings.Trim(match[1], "`\"")
			if column == "" {
				return "", errors.New("cannot determine MySQL no-op conflict column")
			}
			code = conflictNothing.ReplaceAllString(code, "ON DUPLICATE KEY UPDATE `"+column+"`=`"+column+"`")
		}
		return code, nil
	})
	if err != nil {
		return queryPlan{}, err
	}
	// REGEXP 与 now() 交给分词器处理：它们必须只在引号之外生效。
	// 用 ReplaceAll 直接改整串会把字符串字面量里的内容一起改掉。
	out := make([]byte, 0, len(query)+32)
	var positions []int
	requiredArgs := 0
	for i := 0; i < len(query); {
		// SQL comments must not bind parameters or rewrite operators.
		if strings.HasPrefix(query[i:], "--") || query[i] == '#' {
			end := strings.IndexByte(query[i:], '\n')
			if end < 0 {
				out = append(out, query[i:]...)
				break
			}
			out = append(out, query[i:i+end+1]...)
			i += end + 1
			continue
		}
		if strings.HasPrefix(query[i:], "/*") {
			end := strings.Index(query[i+2:], "*/")
			if end < 0 {
				return queryPlan{}, errors.New("unterminated SQL block comment")
			}
			end += i + 4
			out = append(out, query[i:end]...)
			i = end
			continue
		}
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

func rewriteSQLCodeSegments(query string, rewrite func(string) (string, error)) (string, error) {
	var out strings.Builder
	codeStart := 0
	for i := 0; i < len(query); {
		if strings.HasPrefix(query[i:], "--") || query[i] == '#' {
			code, err := rewrite(query[codeStart:i])
			if err != nil {
				return "", err
			}
			out.WriteString(code)
			end := strings.IndexByte(query[i:], '\n')
			if end < 0 {
				out.WriteString(query[i:])
				return out.String(), nil
			}
			end += i + 1
			out.WriteString(query[i:end])
			i, codeStart = end, end
			continue
		}
		if strings.HasPrefix(query[i:], "/*") {
			code, err := rewrite(query[codeStart:i])
			if err != nil {
				return "", err
			}
			out.WriteString(code)
			end := strings.Index(query[i+2:], "*/")
			if end < 0 {
				return "", errors.New("unterminated SQL block comment")
			}
			end += i + 4
			out.WriteString(query[i:end])
			i, codeStart = end, end
			continue
		}
		if query[i] == '\'' || query[i] == '"' || query[i] == '`' {
			code, err := rewrite(query[codeStart:i])
			if err != nil {
				return "", err
			}
			out.WriteString(code)
			quote := query[i]
			start := i
			i++
			closed := false
			for i < len(query) {
				c := query[i]
				i++
				if c == '\\' && i < len(query) {
					i++
					continue
				}
				if c == quote {
					if i < len(query) && query[i] == quote {
						i++
						continue
					}
					closed = true
					break
				}
			}
			if !closed {
				return "", errors.New("unterminated SQL quoted value")
			}
			out.WriteString(query[start:i])
			codeStart = i
			continue
		}
		i++
	}
	code, err := rewrite(query[codeStart:])
	if err != nil {
		return "", err
	}
	out.WriteString(code)
	return out.String(), nil
}

func rewriteDialectExpressions(query string) (string, error) {
	var out strings.Builder
	for i := 0; i < len(query); {
		if strings.HasPrefix(query[i:], "--") || query[i] == '#' {
			end := strings.IndexByte(query[i:], '\n')
			if end < 0 {
				out.WriteString(query[i:])
				break
			}
			end += i + 1
			out.WriteString(query[i:end])
			i = end
			continue
		}
		if strings.HasPrefix(query[i:], "/*") {
			end := strings.Index(query[i+2:], "*/")
			if end < 0 {
				return "", errors.New("unterminated SQL block comment")
			}
			end += i + 4
			out.WriteString(query[i:end])
			i = end
			continue
		}
		if query[i] == '\'' || query[i] == '"' || query[i] == '`' {
			quote, start := query[i], i
			i++
			closed := false
			for i < len(query) {
				c := query[i]
				i++
				if c == '\\' && i < len(query) {
					i++
					continue
				}
				if c == quote {
					if i < len(query) && query[i] == quote {
						i++
						continue
					}
					closed = true
					break
				}
			}
			if !closed {
				return "", errors.New("unterminated SQL quoted value")
			}
			out.WriteString(query[start:i])
			continue
		}
		if i == 0 || !identifierByte(query[i-1]) {
			if match := hardcodedTimeOffset.FindStringSubmatch(query[i:]); match != nil {
				out.WriteString("TIMESTAMPADD(SECOND,$" + match[1] + ",now())")
				i += len(match[0])
				continue
			}
			if match := dateFormat.FindStringSubmatch(query[i:]); match != nil {
				format := "%Y-%m-%d"
				if strings.Contains(strings.ToUpper(match[2]), "HH24") {
					format += " %H:%i:%s"
				}
				if strings.HasSuffix(strings.ToUpper(match[2]), ".MS") {
					out.WriteString("LEFT(DATE_FORMAT(" + match[1] + ",'" + format + ".%f'),23)")
				} else {
					out.WriteString("DATE_FORMAT(" + match[1] + ",'" + format + "')")
				}
				i += len(match[0])
				continue
			}
			if match := intervals.FindStringSubmatch(query[i:]); match != nil {
				out.WriteString("INTERVAL " + match[1] + " " + strings.TrimSuffix(strings.ToUpper(match[2]), "S"))
				i += len(match[0])
				continue
			}
		}
		out.WriteByte(query[i])
		i++
	}
	return out.String(), nil
}

func containsSQLCodePattern(query string, pattern *regexp.Regexp) (bool, error) {
	found := false
	_, err := rewriteSQLCodeSegments(query, func(code string) (string, error) {
		if pattern.MatchString(code) {
			found = true
		}
		return code, nil
	})
	return found, err
}
