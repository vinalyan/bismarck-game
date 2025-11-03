package utils

import (
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
	// Register custom validators if needed
}

// ValidateRequest validates a struct using validator tags and writes validation errors
// in a format compatible with WriteValidationError
// Returns true if validation passes, false if validation fails (and writes error response)
func ValidateRequest(w http.ResponseWriter, req interface{}) bool {
	if err := validate.Struct(req); err != nil {
		validationErrors := make(map[string]string)
		var errorMessage string

		if validationErr, ok := err.(validator.ValidationErrors); ok {
			for _, fieldError := range validationErr {
				fieldName := strings.ToLower(fieldError.Field())
				fieldMsg := getValidationErrorMessage(fieldError)
				validationErrors[fieldName] = fieldMsg
				// Use first error as main error message
				if errorMessage == "" {
					errorMessage = fieldMsg
				}
			}
		} else {
			// Fallback for non-validation errors
			errorMessage = "Invalid request format"
			validationErrors["body"] = errorMessage
		}

		WriteValidationError(w, errorMessage, validationErrors)
		return false
	}

	return true
}

// capitalizeFirstLetter capitalizes the first letter of a string
func capitalizeFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

// getValidationErrorMessage returns a user-friendly error message for a validation error
func getValidationErrorMessage(fieldError validator.FieldError) string {
	// Use the original field name to preserve capitalization (e.g., "Username", "Email")
	fieldName := fieldError.Field()

	switch fieldError.Tag() {
	case "required":
		return fieldName + " is required"
	case "min":
		// Special handling for password min length
		if fieldError.Field() == "Password" || fieldError.Field() == "NewPassword" {
			return fieldName + " is too short"
		}
		return fieldName + " must be at least " + fieldError.Param() + " characters"
	case "max":
		return fieldName + " must be at most " + fieldError.Param() + " characters"
	case "email":
		return "Email is required"
	case "oneof":
		return fieldName + " must be one of: " + strings.ReplaceAll(fieldError.Param(), " ", ", ")
	default:
		return "Invalid " + fieldName
	}
}

