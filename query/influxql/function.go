package influxql

import (
	"fmt"

	"github.com/influxdata/influxql"
	"github.com/influxdata/platform/query"
	"github.com/influxdata/platform/query/execute"
	"github.com/influxdata/platform/query/functions"
)

// createFunctionCursor creates a new cursor that calls a function on one of the columns
// and returns the result.
func createFunctionCursor(t *transpilerState, call *influxql.Call, in cursor) (cursor, error) {
	cur := &functionCursor{
		call:   call,
		parent: in,
	}
	switch call.Name {
	case "mean":
		value, ok := in.Value(call.Args[0])
		if !ok {
			return nil, fmt.Errorf("undefined variable: %s", call.Args[0])
		}
		cur.id = t.op("mean", &functions.MeanOpSpec{
			AggregateConfig: execute.AggregateConfig{
				Columns: []string{value},
				TimeSrc: execute.DefaultStartColLabel,
				TimeDst: execute.DefaultTimeColLabel,
			},
		}, in.ID())
		cur.value = value
		cur.exclude = map[influxql.Expr]struct{}{call.Args[0]: {}}
	case "max":
		value, ok := in.Value(call.Args[0])
		if !ok {
			return nil, fmt.Errorf("undefined variable: %s", call.Args[0])
		}
		cur.id = t.op("max", &functions.MaxOpSpec{
			SelectorConfig: execute.SelectorConfig{
				Column: value,
			},
		}, in.ID())
		cur.value = value
		cur.exclude = map[influxql.Expr]struct{}{call.Args[0]: {}}
	default:
		return nil, fmt.Errorf("unimplemented function: %q", call.Name)
	}
	return cur, nil
}

type functionCursor struct {
	id      query.OperationID
	call    *influxql.Call
	value   string
	exclude map[influxql.Expr]struct{}
	parent  cursor
}

func (c *functionCursor) ID() query.OperationID {
	return c.id
}

func (c *functionCursor) Keys() []influxql.Expr {
	keys := []influxql.Expr{c.call}
	if a := c.parent.Keys(); len(a) > 0 {
		for _, e := range a {
			if _, ok := c.exclude[e]; ok {
				continue
			}
			keys = append(keys, e)
		}
	}
	return keys
}

func (c *functionCursor) Value(expr influxql.Expr) (string, bool) {
	if expr == c.call {
		return c.value, true
	} else if _, ok := c.exclude[expr]; ok {
		return "", false
	}
	return c.parent.Value(expr)
}
