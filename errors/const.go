package errors

// ConstError implements TypedError interface, is a constant int type error.
type ConstError Type

// Error implement the error interface and returns the string value.
func (e ConstError) Error() string {
	return constStrMap[e]
}

// Type implement the TypedError interface.
func (e ConstError) Type() Type {
	return Type(e)
}

// InnerErr returns nil for ConstError, implements TypedError interface.
func (e ConstError) InnerErr() TypedError {
	return nil
}

// const errors
const (
	// AuthorizationNotFound indicate an error when authorization is not found.
	AuthorizationNotFound ConstError = baseConst // 20000
	// AuthorizationNotFoundContext indicate an error when authorization is not found in context.
	AuthorizationNotFoundContext ConstError = 20001
	// OrganizationNotFound indicate an error when organization is not found.
	OrganizationNotFound ConstError = 20002
	// UserNotFound indicate an error when user is not found
	UserNotFound ConstError = 20003
	// TokenNotFoundContext indicate an error when token is not found in context
	TokenNotFoundContext ConstError = 20004
	// URLMissingID indicate the request URL missing id parameter.
	URLMissingID ConstError = 20005
	// EmptyValue indicate an error of empty value.
	EmptyValue ConstError = 20006
)

// common phases
const (
	notFound        = " not found"
	notFoundContext = " not found on context"
)

var constStrMap = map[ConstError]string{
	AuthorizationNotFound:        "authorization not found",
	AuthorizationNotFoundContext: "authorization not found on context",
	OrganizationNotFound:         "organization not found",
	UserNotFound:                 "user not found",
	TokenNotFoundContext:         "token not found on context",
	URLMissingID:                 "url missing id",
	EmptyValue:                   "empty value",
}
