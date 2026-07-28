package boresolver

import (
	"fmt"
	"strings"
	"time"
)

// FXConversionConfig specifies the target reporting currency and temporal date context
type FXConversionConfig struct {
	TargetCurrency string
	EffectiveDate  time.Time
	FXTableName    string // defaults to "public.fx_rates"
}

// FXResolver inspects semantic terms and injects currency translation joins/expressions
type FXResolver struct {
	Config FXConversionConfig
}

// NewFXResolver creates an FXResolver given a target currency and optional effective date
func NewFXResolver(targetCurrency string, effDate time.Time) *FXResolver {
	fxTable := "public.fx_rates"
	if targetCurrency == "" {
		targetCurrency = "USD"
	}
	return &FXResolver{
		Config: FXConversionConfig{
			TargetCurrency: targetCurrency,
			EffectiveDate:  effDate,
			FXTableName:    fxTable,
		},
	}
}

// InjectFXConversionJoin injects a temporal FX rate join into a query generation context if the term has a foreign currency property
func (f *FXResolver) InjectFXConversionJoin(ctx *GenerationContext, dialect Dialect, termCurrency string, valueExpr string) (string, string) {
	if termCurrency == "" || strings.EqualFold(termCurrency, f.Config.TargetCurrency) {
		return valueExpr, ""
	}

	fxAlias := fmt.Sprintf("fx_%s", strings.ToLower(termCurrency))
	if ctx.Aliases == nil {
		ctx.Aliases = make(map[string]string)
	}

	// Check if join already registered
	if existingAlias, ok := ctx.Aliases[fxAlias]; ok {
		fxAlias = existingAlias
	} else {
		ctx.Aliases[fxAlias] = fxAlias

		if dialect == nil {
			dialect = PostgresDialect{}
		}

		ctx.ParamCounter++
		fromCurrToken := paramToken(dialect, ctx.ParamCounter)
		ctx.Args = append(ctx.Args, termCurrency)

		ctx.ParamCounter++
		toCurrToken := paramToken(dialect, ctx.ParamCounter)
		ctx.Args = append(ctx.Args, f.Config.TargetCurrency)

		joinCond := fmt.Sprintf("%s.from_currency = %s AND %s.to_currency = %s", fxAlias, fromCurrToken, fxAlias, toCurrToken)

		ctx.Joins = append(ctx.Joins, JoinStep{
			FromTable: "t0",
			ToTable:   f.Config.FXTableName,
			Alias:     fxAlias,
			Type:      "LEFT",
			Condition: joinCond,
		})
	}

	convertedExpr := fmt.Sprintf("(%s * COALESCE(%s.rate, 1.0))", valueExpr, fxAlias)
	return convertedExpr, fxAlias
}
