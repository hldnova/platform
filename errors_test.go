package platform

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

const EFailedToGetStorageHost = "failed to get the storage host"

func TestErrorMsg(t *testing.T) {
	cases := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "simple error",
			err:  &Error{Code: EAuthorizationNotFound},
			msg:  "<authorization not found>",
		},
		{
			name: "with op",
			err: &Error{
				Code: EAuthorizationNotFound,
				Op:   "bolt.FindAuthorizationByID",
			},
			msg: "bolt.FindAuthorizationByID: <authorization not found>",
		},
		{
			name: "with op and value",
			err: &Error{
				Code: EAuthorizationNotFound,
				Op:   "bolt.FindAuthorizationByID",
				Msg:  fmt.Sprintf("with ID %d", 323),
			},
			msg: "bolt.FindAuthorizationByID: <authorization not found> with ID 323",
		},
		{
			name: "with a third party error",
			err: &Error{
				Code: EFailedToGetStorageHost,
				Op:   "cmd/fluxd.injectDeps",
				Err:  errors.New("empty value"),
			},
			msg: "cmd/fluxd.injectDeps: empty value",
		},
		{
			name: "with a internal error",
			err: &Error{
				Code: EFailedToGetStorageHost,
				Op:   "cmd/fluxd.injectDeps",
				Err:  &Error{Code: EEmptyValue, Op: "cmd/fluxd.getStrList"},
			},
			msg: "cmd/fluxd.injectDeps: cmd/fluxd.getStrList: <empty value>",
		},
	}
	for _, c := range cases {
		if c.msg != c.err.Error() {
			t.Fatalf("%s failed, want %s, got %s", c.name, c.msg, c.err.Error())
		}
	}
}

func TestErrorCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error",
		},
		{
			name: "simple error",
			err:  &Error{Code: EAuthorizationNotFound},
			want: EAuthorizationNotFound,
		},
		{
			name: "http error",
			err:  &HTTPError{Err: Error{Code: EAuthorizationNotFound}},
			want: EAuthorizationNotFound,
		},
		{
			name: "embed error",
			err:  &Error{Code: EAuthorizationNotFound, Err: &Error{Code: EURLMissingID}},
			want: EAuthorizationNotFound,
		},
		{
			name: "default error",
			err:  errors.New("s"),
			want: EInternal,
		},
	}
	for _, c := range cases {
		if result := ErrorCode(c.err); c.want != result {
			t.Fatalf("%s failed, want %s, got %s", c.name, c.want, result)
		}
	}
}

func TestJSON(t *testing.T) {
	cases := []struct {
		name string
		err  *Error
		json string
	}{
		{
			name: "simple error",
			err:  &Error{Code: EAuthorizationNotFound},
			json: "{\"code\":\"authorization not found\"}",
		},
		{
			name: "with op",
			err: &Error{
				Code: EAuthorizationNotFound,
				Op:   "bolt.FindAuthorizationByID",
			},
			json: "{\"code\":\"authorization not found\",\"op\":\"bolt.FindAuthorizationByID\"}",
		},
		{
			name: "with op and value",
			err: &Error{
				Code: EAuthorizationNotFound,
				Op:   "bolt.FindAuthorizationByID",
				Msg:  fmt.Sprintf("with ID %d", 323),
			},
			json: "{\"code\":\"authorization not found\",\"msg\":\"with ID 323\",\"op\":\"bolt.FindAuthorizationByID\"}",
		},
		{
			name: "with a third party error",
			err: &Error{
				Code: EFailedToGetStorageHost,
				Op:   "cmd/fluxd.injectDeps",
				Err:  errors.New("empty value"),
			},
			json: "{\"code\":\"failed to get the storage host\",\"op\":\"cmd/fluxd.injectDeps\",\"err\":\"empty value\"}",
		},
		{
			name: "with a internal error",
			err: &Error{
				Code: EFailedToGetStorageHost,
				Op:   "cmd/fluxd.injectDeps",
				Err:  &Error{Code: EEmptyValue, Op: "cmd/fluxd.getStrList"},
			},
			json: "{\"code\":\"failed to get the storage host\",\"op\":\"cmd/fluxd.injectDeps\",\"err\":\"{\\\"code\\\":\\\"empty value\\\",\\\"op\\\":\\\"cmd/fluxd.getStrList\\\"}\"}",
		},
	}
	for _, c := range cases {
		result, err := json.Marshal(c.err)
		// encode testing
		if err != nil {
			t.Fatalf("%s encode failed, want err: %v, should be nil", c.name, err)
		}
		if c.json != string(result) {
			t.Fatalf("%s encode failed, want %s, got %s", c.name, c.json, string(result))
		}
		// decode testing
		result2 := new(Error)
		err = json.Unmarshal([]byte(c.json), result2)
		if err != nil {
			t.Fatalf("%s decode failed, want err: %v, should be nil", c.name, err)
		}
		decodeEqual(t, c.err, result2, "decode: "+c.name)
	}
}

func decodeEqual(t *testing.T, want, result *Error, caseName string) {
	if want.Code != result.Code {
		t.Fatalf("%s code failed, want %s, got %s", caseName, want.Code, result.Code)
	}
	if want.Op != result.Op {
		t.Fatalf("%s op failed, want %s, got %s", caseName, want.Op, result.Op)
	}
	if want.Msg != result.Msg {
		t.Fatalf("%s msg failed, want %s, got %s", caseName, want.Msg, result.Msg)
	}
	if want.Err != nil {
		if _, ok := want.Err.(*Error); ok {
			decodeEqual(t, want.Err.(*Error), result.Err.(*Error), caseName)
		} else {
			if want.Err.Error() != result.Err.Error() {
				t.Fatalf("%s Err failed, want %s, got %s", caseName, want.Err.Error(), result.Err.Error())
			}
		}
	}
}
