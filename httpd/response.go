package httpd

const (
	// OK for successful reponse
	OK = 1000
	// ParamsError for params error reponse
	ParamsError = 1001
	// KeyNotFound for key/value no found response
	KeyNotFound = 1002
	// KeyDuplicate for key/value duplicate response
	KeyDuplicate = 1003
	// InernalError for internal error
	InernalError = 1004
)

// Status for response
type Status struct {
	Code    int
	Message string
}

// StatusParamsError for params error status
var StatusParamsError = Status{
	Code:    ParamsError,
	Message: "params error",
}

// StatusKeyNotFound for key/value no found status
var StatusKeyNotFound = Status{
	Code:    KeyNotFound,
	Message: "key not found",
}

// StatusKeyDuplicate for key/value duplicate status
var StatusKeyDuplicate = Status{
	Code:    KeyDuplicate,
	Message: "key duplicate",
}

// StatusInternalError for internal error
var StatusInternalError = Status{
	Code:    InernalError,
	Message: "internal error",
}

// StatusOK for successful status
var StatusOK = Status{
	Code:    OK,
	Message: "",
}
