package errors

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithErr(t *testing.T) {
	src := []TypedError{
		NewFailedToGetStorageHost(errors.New("1")),
		NewFailedToGetBucketName(errors.New("2")),
	}
	codes := []Type{
		FailedToGetStorageHost,
		FailedToGetBucketName,
	}
	strs := []string{
		"Failed to get the storage host",
		"Failed to get the bucket name",
	}
	errStr := []string{
		"Failed to get the storage host: 1",
		"Failed to get the bucket name: 2",
	}
	for k, v := range src {
		item := v.(*withErr)
		assert.Equal(t, codes[k], item.Type())
		assert.Equal(t, strs[k], item.Typ.Reference())
		assert.Equal(t, errStr[k], item.Error())
	}
	// test nil
	assert.True(t, NewFailedToGetBucketName(nil) == nil)
}

func TestWithValue(t *testing.T) {
	src := []TypedError{
		NewOrganizationNameAlreadyExist("1"),
		NewUserNameAlreadyExist("2"),
	}
	codes := []Type{
		OrganizationNameAlreadyExist,
		UserNameAlreadyExist,
	}
	strs := []string{
		"organization with name %s already exists",
		"user with name %s already exists",
	}
	errStr := []string{
		"organization with name 1 already exists",
		"user with name 2 already exists",
	}
	for k, v := range src {
		item := v.(*withValue)
		assert.Equal(t, codes[k], item.Type())
		assert.Equal(t, strs[k], item.typ.Reference())
		assert.Equal(t, errStr[k], item.Error())
	}
}

func TestConst(t *testing.T) {
	src := []TypedError{
		AuthorizationNotFound,
		AuthorizationNotFoundContext,
	}
	codes := []Type{
		Type(AuthorizationNotFound),
		Type(AuthorizationNotFoundContext),
	}
	strs := []string{
		"authorization not found",
		"authorization not found on context",
	}

	for k, v := range src {
		assert.Equal(t, codes[k], v.Type())
		assert.Equal(t, strs[k], v.Type().Reference())
		assert.Equal(t, strs[k], v.Error())
	}
}

func TestHTTP(t *testing.T) {
	src := []HTTPError{
		NewInternalError(errors.New("1")),
		NewMalformedData(errors.New("2")),
		NewInvalidData(errors.New("3")),
		NewForbidden(errors.New("4")),
		NewForbidden(NewFailedToGetBucketName(errors.New("5"))),
	}
	codes := []Type{
		Type(InternalError),
		Type(MalformedData),
		Type(InvalidData),
		Type(Forbidden),
		Type(FailedToGetBucketName),
	}
	strs := []string{
		"Internal Error",
		"Malformed Data",
		"Invalid Data",
		"Forbidden",
		"Failed to get the bucket name",
	}
	errStrs := []string{
		"Internal Error: 1",
		"Malformed Data: 2",
		"Invalid Data: 3",
		"Forbidden: 4",
		"Failed to get the bucket name: 5",
	}
	httpCodes := []int{
		http.StatusInternalServerError,
		http.StatusBadRequest,
		http.StatusUnprocessableEntity,
		http.StatusForbidden,
		http.StatusForbidden,
	}
	for k, v := range src {
		item := v.(*httpError)
		assert.Equal(t, httpCodes[k], v.HTTPCode(), item.Error())
		assert.Equal(t, codes[k], item.InnerType(), item.Error())
		assert.Equal(t, strs[k], item.InnerType().Reference())
		assert.Equal(t, errStrs[k], item.Error())
		assert.Equal(t, httpCodes[k], item.HTTPCode())
	}
}

type mockResp struct {
	header     http.Header
	StatusCode int
	bytes.Buffer
}

func (m *mockResp) Header() http.Header {
	return m.header
}

func (m *mockResp) WriteHeader(statusCode int) {
	m.StatusCode = statusCode
}

func TestHandleHTTP(t *testing.T) {
	e := NewInternalError(errors.New("1"))
	w := &mockResp{
		header: make(http.Header),
	}
	HandleHTTPErr(context.Background(), e, w)
	assert.Equal(t, http.StatusInternalServerError, w.StatusCode)
	assert.Equal(t, http.Header{
		"X-Influx-Error":     []string{"Internal Error: 1"},
		"X-Influx-Reference": []string{"60000"},
	}, w.header)
	w = &mockResp{
		header: make(http.Header),
	}
	HandleHTTPErr(context.Background(), nil, w)
	assert.Equal(t, 0, w.StatusCode)
	assert.Equal(t, http.Header{}, w.header)
}

func TestMarshalUnmarshal(t *testing.T) {
	src := []TypedError{
		UserNotFound,
		//withValue
		NewOrganizationNameAlreadyExist("2"),
		// withErr
		NewFailedToGetBucketName(errors.New("4")),
		NewFailedToGetBucketName(AuthorizationNotFound),
		NewFailedToGetStorageHost(NewUserNameAlreadyExist("5")),
		// httpErr
		NewInternalError(errors.New("1")),
		NewForbidden(OrganizationNotFound),
		NewMalformedData(NewOrganizationNameAlreadyExist("3")),
	}
	res := []string{
		`{"code":20003,"message":"user not found","raw":20003}`,
		`{"code":40000,"message":"organization with name 2 already exists","raw":{"values":["2"]}}`,
		`{"code":2,"message":"Failed to get the bucket name: 4","raw":{"code":2,"has_type":false,"message":"4"}}`,
		`{"code":2,"message":"Failed to get the bucket name: authorization not found","embed":{"code":20000,"message":"authorization not found","raw":{"code":20000,"message":"authorization not found","raw":20000}}}`,
		`{"code":1,"message":"Failed to get the storage host: user with name 5 already exists","embed":{"code":40001,"message":"user with name 5 already exists","raw":{"code":40001,"message":"user with name 5 already exists","raw":{"values":["5"]}}}}`,
		`{"code":60000,"message":"Internal Error: 1","raw":{"http_type":60000,"has_type":false,"message":"1"}}`,
		`{"code":60003,"message":"organization not found","embed":{"code":20002,"message":"organization not found","raw":{"code":20002,"message":"organization not found","raw":20002}}}`,
		`{"code":60001,"message":"organization with name 3 already exists","embed":{"code":40000,"message":"organization with name 3 already exists","raw":{"code":40000,"message":"organization with name 3 already exists","raw":{"values":["3"]}}}}`,
	}
	for k, v := range src {
		b, err := MarshalJSON(v)
		assert.Nil(t, err)
		assert.Equal(t, res[k], string(b))
	}

	for k, v := range res {
		s, err := UnmarshalJSON([]byte(v))
		assert.Nil(t, err)
		if errHTTP, ok := src[k].(*httpError); ok {
			errHTTPDecoded := s.(*httpError)
			if !errHTTPDecoded.HasType {
				assert.Equal(t, errHTTP.TypedErr, errHTTPDecoded.TypedErr)
			}
			assert.Equal(t, errHTTP.HasType, errHTTPDecoded.HasType)
			assert.Equal(t, errHTTP.Msg, errHTTPDecoded.Msg)
			assert.Equal(t, errHTTP.HTTPTyp, errHTTPDecoded.HTTPTyp)
			continue
		}
		if errWithErr, ok := src[k].(*withErr); ok {
			errWithErrDecoded := s.(*withErr)
			if errWithErrDecoded.HasType {
				assert.Equal(t, errWithErr.TypedErr, errWithErrDecoded.TypedErr)
			}
			assert.Equal(t, errWithErr.HasType, errWithErrDecoded.HasType)
			assert.Equal(t, errWithErr.Msg, errWithErrDecoded.Msg)
			assert.Equal(t, errWithErr.Typ, errWithErrDecoded.Typ)
			continue
		}
		assert.Equal(t, src[k], s)
	}
}

func TestBadUnmarshal(t *testing.T) {
	src := []string{
		``,
		`{"code":60003,"message":"organization not found","embed":{"code":20002,"message":"organization not found","raw":{"code":"bad","message":"organization not found","raw":{}}}}`,
		`{"code":60003,"message":"organization not found","embed":{}}}`,
	}
	for _, v := range src {
		_, err := UnmarshalJSON([]byte(v))
		assert.Equal(t, JSONUnmarshal, err.Type())
	}

}
