// Copyright 2011 The Go Authors. All rights reserved.
// Conversion portions derive from database/sql/convert.go (Go 1.26.10).
// The Go BSD license is retained in third_party/go-sql-conversion/LICENSE.
package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"unicode"
	"unicode/utf8"
)

// This private carrier is accepted only by our pinned connection wrapper.
// Individual values still follow the same driver/statement conversion routes.
type mysqlBatchArgs struct{ values []any }

func (c *statementConn) CheckNamedValue(nv *driver.NamedValue) error {
	if batch, ok := nv.Value.(mysqlBatchArgs); ok {
		if nv.Name != "" || len(batch.values) < 64 || len(batch.values) > 4096 {
			return fmt.Errorf("invalid internal MySQL batch transport")
		}
		return nil
	}
	return c.mysqlWireConn.CheckNamedValue(nv)
}
func (c *statementConn) expandBatchArgs(args []driver.NamedValue) ([]driver.NamedValue, error) {
	if len(args) != 1 {
		return args, nil
	}
	batch, ok := args[0].Value.(mysqlBatchArgs)
	if !ok {
		return args, nil
	}
	return c.convertBatchValues(batch.values, nil)
}

// Registered before conversion so panic paths also release all references.
// batchScratch retains the full original input length even after removals.
// Prior larger uses cleared their tail before shrinking. Clearing this length
// therefore leaves the whole capacity empty without rewriting already-zero slots.
func (c *statementConn) clearBatchArgs() { clear(c.batchScratch) }

// Keep fallback inside this driver call: database/sql still holds its physical
// connection lock and transaction close lock. The first direct ErrSkip has not
// executed SQL. Preparing again and reconverting originals preserves Valuers,
// statement checkers, NumInput, cancellation and the standard retry boundary.
func (c *statementConn) execBatchDirect(ctx context.Context, q string, args []driver.NamedValue, originals []any) (driver.Result, error) {
	result, err := c.mysqlWireConn.ExecContext(ctx, q, args)
	if err != driver.ErrSkip || originals == nil {
		return result, err
	}
	stmt, err := c.PrepareContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	c.clearBatchArgs()
	converted, err := c.convertBatchValues(originals, stmt)
	if err != nil {
		return nil, err
	}
	exec, ok := stmt.(driver.StmtExecContext)
	if !ok {
		return nil, fmt.Errorf("MySQL driver lacks context statement interfaces")
	}
	return exec.ExecContext(ctx, converted)
}

var batchValuerReflectType = reflect.TypeFor[driver.Valuer]()

type batchCCChecker struct {
	cci  driver.ColumnConverter
	want int
}

func (c batchCCChecker) CheckNamedValue(nv *driver.NamedValue) error {
	if c.cci == nil {
		return driver.ErrSkip
	}
	// The column converter shouldn't be called on any index
	// it isn't expecting. The final error will be thrown
	// in the argument converter loop.
	index := nv.Ordinal - 1
	if c.want >= 0 && c.want <= index {
		return nil
	}

	// First, see if the value itself knows how to convert
	// itself to a driver type. For example, a NullString
	// struct changing into a string or nil.
	if vr, ok := nv.Value.(driver.Valuer); ok {
		sv, err := batchCallValuerValue(vr)
		if err != nil {
			return err
		}
		if !driver.IsValue(sv) {
			return fmt.Errorf("non-subset type %T returned from Value", sv)
		}
		nv.Value = sv
	}

	// Second, ask the column to sanity check itself. For
	// example, drivers might use this to make sure that
	// an int64 values being inserted into a 16-bit
	// integer field is in range (before getting
	// truncated), or that a nil can't go into a NOT NULL
	// column before going across the network to get the
	// same error.
	var err error
	arg := nv.Value
	nv.Value, err = c.cci.ColumnConverter(index).ConvertValue(arg)
	if err != nil {
		return err
	}
	if !driver.IsValue(nv.Value) {
		return fmt.Errorf("driver ColumnConverter error converted %T to unsupported type %T", arg, nv.Value)
	}
	return nil
}

func batchDefaultCheckNamedValue(nv *driver.NamedValue) (err error) {
	nv.Value, err = driver.DefaultParameterConverter.ConvertValue(nv.Value)
	return err
}

func batchDescribeNamedValue(nv *driver.NamedValue) string {
	if len(nv.Name) == 0 {
		return fmt.Sprintf("$%d", nv.Ordinal)
	}
	return fmt.Sprintf("with name %q", nv.Name)
}

func batchValidateNamedValueName(name string) error {
	if len(name) == 0 {
		return nil
	}
	r, _ := utf8.DecodeRuneInString(name)
	if unicode.IsLetter(r) {
		return nil
	}
	return fmt.Errorf("name %q does not begin with a letter", name)
}

func batchCallValuerValue(vr driver.Valuer) (v driver.Value, err error) {
	if rv := reflect.ValueOf(vr); rv.Kind() == reflect.Pointer &&
		rv.IsNil() &&
		rv.Type().Elem().Implements(batchValuerReflectType) {
		return nil, nil
	}
	return vr.Value()
}

func (c *statementConn) convertBatchValues(args []any, si driver.Stmt) ([]driver.NamedValue, error) {
	if len(args) < 64 || len(args) > 4096 {
		return nil, fmt.Errorf("invalid internal MySQL batch transport")
	}
	if cap(c.batchScratch) < len(args) {
		c.batchScratch = make([]driver.NamedValue, len(args))
	} else {
		c.batchScratch = c.batchScratch[:len(args)]
	}
	nvargs := c.batchScratch
	var ci driver.Conn = c.mysqlWireConn

	// -1 means the driver doesn't know how to count the number of
	// placeholders, so we won't sanity check input here and instead let the
	// driver deal with errors.
	want := -1

	var cc batchCCChecker
	if si != nil {
		want = si.NumInput()
		cc.want = want
	}

	// Check all types of interfaces from the start.
	// Drivers may opt to use the NamedValueChecker for special
	// argument types, then return driver.ErrSkip to pass it along
	// to the column converter.
	nvc, ok := si.(driver.NamedValueChecker)
	if !ok {
		nvc, _ = ci.(driver.NamedValueChecker)
	}
	cci, ok := si.(driver.ColumnConverter)
	if ok {
		cc.cci = cci
	}

	// Loop through all the arguments, checking each one.
	// If no error is returned simply increment the index
	// and continue. However, if driver.ErrRemoveArgument
	// is returned the argument is not included in the query
	// argument list.
	var err error
	var n int
	for _, arg := range args {
		nv := &nvargs[n]
		if np, ok := arg.(sql.NamedArg); ok {
			if err = batchValidateNamedValueName(np.Name); err != nil {
				return nil, err
			}
			arg = np.Value
			nv.Name = np.Name
		}
		nv.Ordinal = n + 1
		nv.Value = arg

		// Checking sequence has four routes:
		// A: 1. Default
		// B: 1. NamedValueChecker 2. Column Converter 3. Default
		// C: 1. NamedValueChecker 3. Default
		// D: 1. Column Converter 2. Default
		//
		// The only time a Column Converter is called is first
		// or after NamedValueConverter. If first it is handled before
		// the nextCheck label. Thus for repeats tries only when the
		// NamedValueConverter is selected should the Column Converter
		// be used in the retry.
		checker := batchDefaultCheckNamedValue
		nextCC := false
		switch {
		case nvc != nil:
			nextCC = cci != nil
			checker = nvc.CheckNamedValue
		case cci != nil:
			checker = cc.CheckNamedValue
		}

	nextCheck:
		err = checker(nv)
		switch err {
		case nil:
			n++
			continue
		case driver.ErrRemoveArgument:
			nvargs = nvargs[:len(nvargs)-1]
			continue
		case driver.ErrSkip:
			if nextCC {
				nextCC = false
				checker = cc.CheckNamedValue
			} else {
				checker = batchDefaultCheckNamedValue
			}
			goto nextCheck
		default:
			return nil, fmt.Errorf("sql: converting argument %s type: %w", batchDescribeNamedValue(nv), err)
		}
	}

	// Check the length of arguments after conversion to allow for omitted
	// arguments.
	if want != -1 && len(nvargs) != want {
		return nil, fmt.Errorf("sql: expected %d arguments, got %d", want, len(nvargs))
	}

	return nvargs, nil
}
