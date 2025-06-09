package api

import (
	"github.com/LamThanhNguyen/banking-system/util"
	"github.com/LamThanhNguyen/banking-system/val"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
			return val.ValidateUsername(fl.Field().String()) == nil
		}); err != nil {
			panic(err)
		}

		if err := v.RegisterValidation("fullname", func(fl validator.FieldLevel) bool {
			return val.ValidateFullname(fl.Field().String()) == nil
		}); err != nil {
			panic(err)
		}

		if err := v.RegisterValidation("currency", func(fl validator.FieldLevel) bool {
			return util.IsSupportedCurrency(fl.Field().String())
		}); err != nil {
			panic(err)
		}

		if err := v.RegisterValidation("email_id", func(fl validator.FieldLevel) bool {
			return val.ValidateEmailId(fl.Field().Int()) == nil
		}); err != nil {
			panic(err)
		}

		if err := v.RegisterValidation("secret_code", func(fl validator.FieldLevel) bool {
			return val.ValidateSecretCode(fl.Field().String()) == nil
		}); err != nil {
			panic(err)
		}
	}
}
