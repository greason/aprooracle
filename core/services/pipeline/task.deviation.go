package pipeline

import (
	"context"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/smartcontractkit/chainlink/v2/core/logger"
	"go.uber.org/multierr"
)

// Return types:
//
//	bool
type DeviationTask struct {
	BaseTask   `mapstructure:",squash"`
	CurAnswer  string `json:"curAnswer"`
	NextAnswer string `json:"nextAnswer"`
	Rel        string `json:"rel"` // threshold: 1 * 100
	Abs        string `json:"abs"` // absoluteThreshold
}

var _ Task = (*DeviationTask)(nil)

func (t *DeviationTask) Type() TaskType {
	return TaskTypeDeviation
}

func (t *DeviationTask) Run(_ context.Context, lggr logger.Logger, vars Vars, inputs []Result) (result Result, runInfo RunInfo) {
	_, err := CheckInputs(inputs, 0, 1, 0)
	if err != nil {
		return Result{Error: errors.Wrap(err, "task inputs")}, runInfo
	}

	var (
		current DecimalParam
		next    DecimalParam
		rel     DecimalParam
		abs     DecimalParam
	)
	err = multierr.Combine(
		errors.Wrap(ResolveParam(&current, From(VarExpr(t.CurAnswer, vars), NonemptyString(t.CurAnswer))), "curAnswer"),
		errors.Wrap(ResolveParam(&next, From(VarExpr(t.NextAnswer, vars), NonemptyString(t.NextAnswer))), "nextAnswer"),
		errors.Wrap(ResolveParam(&rel, From(VarExpr(t.Rel, vars), t.Rel)), "rel"),
		errors.Wrap(ResolveParam(&abs, From(VarExpr(t.Abs, vars), t.Abs)), "abs"),
	)
	if err != nil {
		return Result{Error: err}, runInfo
	}

	if rel.Decimal().IsZero() && abs.Decimal().IsZero() {
		lggr.Debugw("Deviation thresholds both zero; short-circuiting deviation checker to true, regardless of feed values")
		return Result{Value: true}, runInfo
	}
	diff := current.Decimal().Sub(next.Decimal()).Abs()
	if !diff.GreaterThan(abs.Decimal()) {
		return Result{Error: errors.New("Absolute deviation threshold not met")}, runInfo
	}

	if current.Decimal().IsZero() {
		if next.Decimal().IsZero() {
			return Result{Error: errors.New("Relative deviation is undefined; can't satisfy threshold")}, runInfo
		}
		return Result{Value: true}, runInfo
	}

	// 100*|new-old|/|old|: Deviation (relative to curAnswer) as a percentage
	percentage := diff.Div(current.Decimal()).Mul(decimal.NewFromInt(100))
	if percentage.LessThan(rel.Decimal()) {
		return Result{Error: errors.New("Relative deviation threshold not met")}, runInfo
	}

	return Result{Value: true}, runInfo
}
