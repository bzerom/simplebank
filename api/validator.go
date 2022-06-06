package api

import (
	"github.com/bgebrechristos/simplebank/util"
	"github.com/go-playground/validator/v10"
)

var validCurrency validator.Func = func(fl validator.FieldLevel) bool {
	// It's a reflection value so we need to call Interface() to get its value
	// as an empty interface
	if currency, ok := fl.Field().Interface().(string); ok {
		// check if currency is supported
		return util.IsSupportedCurrency(currency)
	}

	return false
}
