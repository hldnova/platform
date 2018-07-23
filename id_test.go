package platform

import (
	"bytes"
	"reflect"
	"testing"
)

func TestIDFromString(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		want    *ID
		wantErr bool
		err     string
	}{
		{
			name: "Should be able to decode an ID",
			id:   "0000000000000000", //020f755c3c082000
			want: &ID{value: func(i uint64) *uint64 { return &i }(0)},
		},
		{
			name:    "Should not be able to decode a non hex ID",
			id:      "gggggggggggggggg",
			wantErr: true,
			err:     "encoding/hex: invalid byte: U+0067 'g'",
		},
		{
			name:    "Should not be able to decode inputs with length other than 8 bytes",
			id:      "abc",
			wantErr: true,
			err:     "input must be an array of 8 bytes",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IDFromString(tt.id)

			// Check negative test cases
			if (err != nil) && tt.wantErr {
				if tt.err != err.Error() {
					t.Errorf("IDFromString() errors out \"%s\", want \"%s\"", err, tt.err)
				}
				return
			}

			// Check positive test cases
			if !reflect.DeepEqual(got, tt.want) && !tt.wantErr {
				t.Errorf("IDFromString() outputs %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecodeFromString(t *testing.T) {
	var id ID
	err := id.DecodeFromString("020f755c3c082000")
	if err != nil {
		t.Errorf(err.Error())
	}
	want := []byte{48, 50, 48, 102, 55, 53, 53, 99, 51, 99, 48, 56, 50, 48, 48, 48}
	got, _ := id.Encode()
	if !bytes.Equal(want, got) {
		t.Errorf("encoding error")
	}
	if id.String() != "020f755c3c082000" {
		t.Errorf("expecting string representation to contain the right value")
	}
}

func TestEncode(t *testing.T) {
	var id ID
	if !id.Empty() {
		t.Errorf("ID must be empty at first stage")
	}
	got, err := id.Encode()
	if len(got) != 0 {
		t.Errorf("encoding an empty ID must result in an empty byte array plus an error")
	}
	if err == nil {
		t.Errorf("encoding an empty ID must result in an empty byte array plus an error")
	}

	id.DecodeFromString("5ca1ab1eba5eba11")
	want := []byte{53, 99, 97, 49, 97, 98, 49, 101, 98, 97, 53, 101, 98, 97, 49, 49}
	got, err = id.Encode()
	if !bytes.Equal(want, got) {
		t.Errorf("encoding error")
	}
	if id.String() != "5ca1ab1eba5eba11" {
		t.Errorf("expecting string representation to contain the right value")
	}
}

func TestDecodeFromShorterString(t *testing.T) {
	var id ID
	err := id.DecodeFromString("020f75")
	if err == nil {
		t.Errorf("expecting shorter inputs to error")
	}
	if id.String() != "" {
		t.Errorf("expecting empty ID to be serialized into empty string")
	}
}

func TestDecodeFromLongerString(t *testing.T) {
	var id ID
	err := id.DecodeFromString("020f755c3c082000aaa")
	if err == nil {
		t.Errorf("expecting shorter inputs to error")
	}
	if id.String() != "" {
		t.Errorf("expecting empty ID to be serialized into empty string")
	}
}

func TestDecodeFromEmptyString(t *testing.T) {
	var id ID
	err := id.DecodeFromString("")
	if err == nil {
		t.Errorf("expecting empty inputs to error")
	}
	if !id.Empty() {
		t.Errorf("expecting ID to be empty")
	}
	if id.String() != "" {
		t.Errorf("expecting empty ID to be serialized into empty string")
	}
}

func TestAllZerosDoesNotMeanEmpty(t *testing.T) {
	var id ID
	id.DecodeFromString("0000000000000000")
	if id.Empty() {
		t.Errorf("expecting id to not be empty")
	}
}
