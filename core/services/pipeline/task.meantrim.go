package pipeline

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/multierr"

	"github.com/smartcontractkit/chainlink/v2/core/logger"
)

// remove lower and higher
//  Return types:

// *decimal.Decimal
type MeanTrimTask struct {
	BaseTask      `mapstructure:",squash"`
	Values        string `json:"values"`
	AllowedFaults string `json:"allowedFaults"`
	Precision     string `json:"precision"`
}

var (
	_ Task = (*MeanTrimTask)(nil)

	ErrBadLength = errors.New("unsuitable length of data for task")
)

func (t *MeanTrimTask) Type() TaskType {
	return TaskTypeMeanTrim
}

func (t *MeanTrimTask) Run(ctx context.Context, lggr logger.Logger, vars Vars, inputs []Result) (result Result, runInfo RunInfo) {
	var (
		maybeAllowedFaults MaybeUint64Param
		maybePrecision     MaybeInt32Param
		valuesAndErrs      SliceParam
		decimalValues      DecimalSliceParam
		allowedFaults      int
		faults             int
	)
	err := multierr.Combine(
		errors.Wrap(ResolveParam(&maybeAllowedFaults, From(t.AllowedFaults)), "allowedFaults"),
		errors.Wrap(ResolveParam(&maybePrecision, From(VarExpr(t.Precision, vars), t.Precision)), "precision"),
		errors.Wrap(ResolveParam(&valuesAndErrs, From(VarExpr(t.Values, vars), JSONWithVarExprs(t.Values, vars, true), Inputs(inputs))), "values"),
	)
	if err != nil {
		return Result{Error: err}, runInfo
	}

	if allowed, isSet := maybeAllowedFaults.Uint64(); isSet {
		allowedFaults = int(allowed)
	} else {
		allowedFaults = len(valuesAndErrs) - 1
	}

	values, faults := valuesAndErrs.FilterErrors()
	if faults > allowedFaults {
		return Result{Error: errors.Wrapf(ErrTooManyErrors, "Number of faulty inputs %v to mean task > number allowed faults %v", faults, allowedFaults)}, runInfo
	} else if len(values) == 0 {
		return Result{Error: errors.Wrap(ErrWrongInputCardinality, "values")}, runInfo
	}

	err = decimalValues.UnmarshalPipelineParam(values)
	if err != nil {
		return Result{Error: errors.Wrapf(ErrBadInput, "values: %v", err)}, runInfo
	}

	if decimalValues == nil || len(decimalValues) < 3 {
		return Result{Error: errors.Wrapf(ErrBadLength, "values: %v", err)}, runInfo
	}
	sort.Slice(decimalValues, func(i, j int) bool {
		return decimalValues[i].Cmp(decimalValues[j]) <= 0
	})

	decimalValues = decimalValues[1 : len(decimalValues)-1]
	total := decimal.NewFromInt(0)
	for _, val := range decimalValues {
		total = total.Add(val)
	}
	numValues := decimal.NewFromInt(int64(len(decimalValues)))

	if precision, isSet := maybePrecision.Int32(); isSet {
		return Result{Value: total.DivRound(numValues, precision)}, runInfo
	}
	// Note that decimal library defaults to rounding to 16 precision
	//https://github.com/shopspring/decimal/blob/2568a29459476f824f35433dfbef158d6ad8618c/decimal.go#L44
	return Result{Value: total.Div(numValues)}, runInfo
}
