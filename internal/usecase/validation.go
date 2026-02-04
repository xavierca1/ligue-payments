package usecase

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func ValidateCreateCustomerInput(input CreateCustomerInput) []ValidationError {
	var errors []ValidationError

	if strings.TrimSpace(input.Name) == "" {
		errors = append(errors, ValidationError{"name", "is required"})
	} else if len(input.Name) < 3 {
		errors = append(errors, ValidationError{"name", "must have at least 3 characters"})
	} else if len(input.Name) > 200 {
		errors = append(errors, ValidationError{"name", "must not exceed 200 characters"})
	}

	if strings.TrimSpace(input.Email) == "" {
		errors = append(errors, ValidationError{"email", "is required"})
	} else if _, err := mail.ParseAddress(input.Email); err != nil {
		errors = append(errors, ValidationError{"email", "is invalid"})
	}

	if input.CPF == "" {
		errors = append(errors, ValidationError{"cpf", "is required"})
	} else if !isValidCPF(input.CPF) {
		errors = append(errors, ValidationError{"cpf", "is invalid"})
	}

	if strings.TrimSpace(input.Phone) == "" {
		errors = append(errors, ValidationError{"phone", "is required"})
	} else if !isValidPhoneNumber(input.Phone) {
		errors = append(errors, ValidationError{"phone", "must be a valid phone number"})
	}

	if strings.TrimSpace(input.BirthDate) == "" {
		errors = append(errors, ValidationError{"birth_date", "is required"})
	} else if !isValidDate(input.BirthDate) {
		errors = append(errors, ValidationError{"birth_date", "must be a valid date (YYYY-MM-DD)"})
	} else if isMinor(input.BirthDate) {
		errors = append(errors, ValidationError{"birth_date", "customer must be at least 18 years old"})
	}

	if strings.TrimSpace(input.PlanID) == "" {
		errors = append(errors, ValidationError{"plan_id", "is required"})
	}

	if strings.TrimSpace(input.Street) == "" {
		errors = append(errors, ValidationError{"street", "is required"})
	}
	if strings.TrimSpace(input.Number) == "" {
		errors = append(errors, ValidationError{"number", "is required"})
	}
	if strings.TrimSpace(input.District) == "" {
		errors = append(errors, ValidationError{"district", "is required"})
	}
	if strings.TrimSpace(input.City) == "" {
		errors = append(errors, ValidationError{"city", "is required"})
	}
	if strings.TrimSpace(input.State) == "" {
		errors = append(errors, ValidationError{"state", "is required"})
	}
	if !isValidZipCode(input.ZipCode) {
		errors = append(errors, ValidationError{"zip_code", "must be a valid zip code (XXXXX-XXX)"})
	}

	if input.PaymentMethod == "" {
		errors = append(errors, ValidationError{"payment_method", "is required"})
	} else if input.PaymentMethod != "PIX" && input.PaymentMethod != "CREDIT_CARD" {
		errors = append(errors, ValidationError{"payment_method", "must be PIX or CREDIT_CARD"})
	}

	if input.PaymentMethod == "CREDIT_CARD" {
		if input.CardHolder == "" {
			errors = append(errors, ValidationError{"card_holder", "is required for CREDIT_CARD payment"})
		}
		if input.CardNumber == "" {
			errors = append(errors, ValidationError{"card_number", "is required for CREDIT_CARD payment"})
		} else if !isValidCardNumber(input.CardNumber) {
			errors = append(errors, ValidationError{"card_number", "is invalid"})
		}
		if input.CardMonth == "" {
			errors = append(errors, ValidationError{"card_month", "is required for CREDIT_CARD payment"})
		} else if !isValidMonth(input.CardMonth) {
			errors = append(errors, ValidationError{"card_month", "must be 01-12"})
		}
		if input.CardYear == "" {
			errors = append(errors, ValidationError{"card_year", "is required for CREDIT_CARD payment"})
		} else if !isValidYear(input.CardYear) {
			errors = append(errors, ValidationError{"card_year", "must be a 2 or 4 digit year"})
		}
		if input.CardCVV == "" {
			errors = append(errors, ValidationError{"card_cvv", "is required for CREDIT_CARD payment"})
		} else if !isValidCVV(input.CardCVV) {
			errors = append(errors, ValidationError{"card_cvv", "must be 3 or 4 digits"})
		}
	}

	if !input.TermsAccepted {
		errors = append(errors, ValidationError{"terms_accepted", "must be accepted"})
	}
	if strings.TrimSpace(input.TermsAcceptedAt) == "" {
		errors = append(errors, ValidationError{"terms_accepted_at", "is required when terms are accepted"})
	} else if !isValidDate(input.TermsAcceptedAt) {
		errors = append(errors, ValidationError{"terms_accepted_at", "must be a valid ISO8601 datetime"})
	}

	return errors
}

func isValidCPF(cpf string) bool {

	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(cpf, "")

	if len(cleaned) != 11 {
		return false
	}

	firstDigit := string(cleaned[0])
	allEqual := true
	for i := 1; i < len(cleaned); i++ {
		if string(cleaned[i]) != firstDigit {
			allEqual = false
			break
		}
	}
	if allEqual {
		return false
	}

	return true
}

func isValidPhoneNumber(phone string) bool {
	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")

	return len(cleaned) >= 10 && len(cleaned) <= 11
}

func isValidDate(dateStr string) bool {

	if _, err := time.Parse("2006-01-02", dateStr); err == nil {
		return true
	}

	if _, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return true
	}

	if _, err := time.Parse(time.RFC3339Nano, dateStr); err == nil {
		return true
	}
	return false
}

func isMinor(birthDate string) bool {
	t, err := time.Parse("2006-01-02", birthDate)
	if err != nil {
		return true // Por segurança, rejeita datas inválidas
	}
	age := time.Now().Year() - t.Year()
	if time.Now().YearDay() < t.YearDay() {
		age--
	}
	return age < 18
}

func isValidGender(gender string) bool {
	g := strings.ToUpper(strings.TrimSpace(gender))
	return g == "1" || g == "2" || g == "3" || g == "M" || g == "F" || g == "OTHER"
}

func isValidZipCode(zipcode string) bool {
	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(zipcode, "")
	return len(cleaned) == 8
}

func isValidCardNumber(cardNumber string) bool {
	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(cardNumber, "")
	if len(cleaned) < 13 || len(cleaned) > 19 {
		return false
	}
	return luhnCheck(cleaned)
}

func luhnCheck(num string) bool {
	sum := 0
	isEven := false

	for i := len(num) - 1; i >= 0; i-- {
		digit := int(num[i] - '0')

		if isEven {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		isEven = !isEven
	}

	return sum%10 == 0
}

func isValidMonth(month string) bool {
	return regexp.MustCompile(`^(0[1-9]|1[0-2])$`).MatchString(month)
}

func isValidYear(year string) bool {
	if !regexp.MustCompile(`^\d{2}(\d{2})?$`).MatchString(year) {
		return false
	}

	fullYear := year
	if len(year) == 2 {
		fullYear = "20" + year
	}

	yearInt := 0
	fmt.Sscanf(fullYear, "%d", &yearInt)
	return yearInt >= time.Now().Year()
}

func isValidCVV(cvv string) bool {
	return regexp.MustCompile(`^\d{3,4}$`).MatchString(cvv)
}
