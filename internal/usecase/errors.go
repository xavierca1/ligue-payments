package usecase


type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string {
	return e.Message
}


func IsDomainError(err error) bool {
	_, ok := err.(*DomainError)
	return ok
}


type TechnicalError struct {
	Code    string
	Message string
}

func (e *TechnicalError) Error() string {
	return e.Message
}


func IsTechnicalError(err error) bool {
	_, ok := err.(*TechnicalError)
	return ok
}
