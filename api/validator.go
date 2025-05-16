package api

import (
	"github.com/LamThanhNguyen/future-bank/util"
	"github.com/LamThanhNguyen/future-bank/val"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
			return val.ValidateUsername(fl.Field().String()) == nil
		})

		v.RegisterValidation("fullname", func(fl validator.FieldLevel) bool {
			return val.ValidateFullname(fl.Field().String()) == nil
		})

		v.RegisterValidation("currency", func(fl validator.FieldLevel) bool {
			return util.IsSupportedCurrency(fl.Field().String())
		})
	}
}
