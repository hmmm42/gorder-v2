package consts

const (
	ErrnoSuccess = 0
	ErrnoUnknown = 1

	// param error 1xxx
	ErrorBindRequestError     = 1000
	ErrorRequestValidateError = 1001

	// mysql error 2xxx
)

var ErrMsg = map[int]string{
	ErrnoSuccess: "success",
	ErrnoUnknown: "unknown error",

	ErrorBindRequestError:     "bind request error",
	ErrorRequestValidateError: "request validate error",
}
