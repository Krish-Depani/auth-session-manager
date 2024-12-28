package validators

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
}

type ValidationError struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
	Value string `json:"value"`
}

type ValidationResponse struct {
	Errors []ValidationError `json:"errors"`
}

func Validate(data interface{}) []ValidationError {
	var validationErrors []ValidationError

	err := validate.Struct(data)
	if err != nil {
		if errors, ok := err.(validator.ValidationErrors); ok {
			for _, e := range errors {
				validationErrors = append(validationErrors, ValidationError{
					Field: e.Field(),
					Tag:   e.Tag(),
					Value: e.Param(),
				})
			}
		}
	}

	return validationErrors
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email" binding:"required,email"`
	Username string `json:"username" validate:"required,min=3,max=50" binding:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=8" binding:"required,min=8"`
	FullName string `json:"full_name" validate:"required" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email" binding:"required,email"`
	Password string `json:"password" validate:"required" binding:"required"`
}

func ValidateRegisterRequest(c *gin.Context) (*RegisterRequest, bool) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request payload",
		})
		return nil, false
	}

	if errs := Validate(req); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, ValidationResponse{
			Errors: errs,
		})
		return nil, false
	}

	return &req, true
}

func ValidateLoginRequest(c *gin.Context) (*LoginRequest, bool) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request payload",
		})
		return nil, false
	}

	if errs := Validate(req); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, ValidationResponse{
			Errors: errs,
		})
		return nil, false
	}

	return &req, true
}
