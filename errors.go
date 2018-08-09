package platform

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// some error code constant, ideally we want define common platform codes here
// projects on use platform's error, should have their own central place like this
const (
	EInternal                     = "internal error"
	ENotFound                     = "not found"
	ESourceNotFound               = "source not found"
	EAuthorizationNotFound        = "authorization not found"
	EAuthorizationNotFoundContext = "authorization not found on context"
	EOrganizationNotFound         = "organization not found"
	EUserNotFound                 = "user is not found"
	ETokenNotFoundContext         = "token not found on context"
	EURLMissingID                 = "url missing id"
	EEmptyValue                   = "empty value"
	EFailedToGetBucketName        = "failed to get the bucket name"
	EOrganizationNameAlreadyExist = "organization already exists"
)

// ErrOpCode is the string of logical operation.
// for example: bolt.UserCreate.
// OpCode should be defined inside each package.
type ErrOpCode string

// Error is the error struct of platform.
// To create a simple error, &Error{Code:EUserNotFound}
// To show where the error happens, add Op.
// &Error{
//     Code: EUserNotFound,
//     Op: "bolt.FindUserByID"
// }
// To show an error with a unpredictable value, add the value in Msg.
// &Error{
//	   Code: EOrganizationNameAlreadyExist,
//	   Message: fmt.Sprintf("with name %s", aName),
// }
// To show an error wrapped with another error.
// &Error{Code:EFailedToGetStorageHost, Err: err}.
type Error struct {
	// Machine-readable error code.
	Code string `json:"code"`
	// Human-readable message.
	Msg string `json:"msg,omitempty"`
	// Logical operation and nested error.
	Op string `json:"op,omitempty"`
	// the embed error
	Err error `json:"err,omitempty"`
}

// Error implement the error interface
func (e Error) Error() string {
	var b strings.Builder

	// Print the current operation in our stack, if any.
	if e.Op != "" {
		fmt.Fprintf(&b, "%s: ", e.Op)
	}

	// If wrapping an error, print its Error() message.
	// Otherwise print the error code & message.
	if e.Err != nil {
		b.WriteString(e.Err.Error())
	} else {
		if e.Code != "" {
			fmt.Fprintf(&b, "<%s>", e.Code)
			if e.Msg != "" {
				b.WriteString(" ")
			}
		}
		b.WriteString(e.Msg)
	}
	return b.String()
}

// to avoid cyclical marshaling
type errEncode struct {
	// Machine-readable error code.
	Code string `json:"code"`
	// Human-readable message.
	Msg string `json:"msg,omitempty"`
	// Logical operation and nested error.
	Op  string `json:"op,omitempty"`
	Err string `json:"err,omitempty"`
}

// MarshalJSON implements json.Marshaler interface.
func (e *Error) MarshalJSON() (result []byte, err error) {
	ee := errEncode{
		Code: e.Code,
		Msg:  e.Msg,
		Op:   e.Op,
	}
	if e.Err != nil {
		if _, ok := e.Err.(*Error); ok {
			b, err := e.Err.(*Error).MarshalJSON()
			if err != nil {
				return result, err
			}
			ee.Err = string(b)
		} else {
			ee.Err = e.Err.Error()
		}
	}
	return json.Marshal(ee)
}

// UnmarshalJSON implement the json.Unmarshaler interface
func (e *Error) UnmarshalJSON(b []byte) (err error) {
	ee := new(errEncode)
	err = json.Unmarshal(b, ee)
	e.Code = ee.Code
	e.Msg = ee.Msg
	e.Op = ee.Op
	if ee.Err != "" {
		var innerErr error
		innerResult := new(Error)
		innerErr = innerResult.UnmarshalJSON([]byte(ee.Err))
		if innerErr != nil {
			e.Err = errors.New(ee.Err)
			return err
		}
		e.Err = innerResult
	}
	return err
}

// HTTPError embed Error with additional HTTPCode
type HTTPError struct {
	Err      Error
	HTTPCode int `json:"http_code"`
}

func (e HTTPError) Error() string {
	return e.Err.Error()
}

// ErrorCode returns the code of the root error, if available. Otherwise returns EINTERNAL.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	} else if e, ok := err.(*Error); ok && e.Code != "" {
		return e.Code
	} else if eHTTP, ok := err.(*HTTPError); ok && eHTTP.Err.Code != "" {
		return eHTTP.Err.Code
	}
	return EInternal
}
