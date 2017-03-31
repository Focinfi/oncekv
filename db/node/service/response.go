package service

const (
	// OK for successful response
	OK = 1000
	// ParamsError for params error response
	ParamsError = 1001
	// KeyNotFound for key/value no found response
	KeyNotFound = 1002
	// KeyDuplicate for key/value duplicate response
	KeyDuplicate = 1003
	// InternalError for internal error
	InternalError = 1004
	// NotLeaderError for not leader error
	NotLeaderError = 1005
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

// StatusNotLeaderError for params error status
var StatusNotLeaderError = Status{
	Code:    NotLeaderError,
	Message: "i am not the leader",
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
	Code:    InternalError,
	Message: "internal error",
}

// StatusOK for successful status
var StatusOK = Status{
	Code:    OK,
	Message: "",
}
