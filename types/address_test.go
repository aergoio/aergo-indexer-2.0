package types

import (
	"github.com/magiconair/properties/assert"
	"reflect"
	"testing"
)

// address "AmMRo3jey9pL5qwJSNJFghe9t9J6fPEoZfNv8XjsqSrajGFuJGpi"를 디코딩한 결과
var SampleADdr = []byte{2, 121, 162, 246, 60, 51, 231, 236, 195, 45, 247, 238, 195, 7, 49, 113, 91, 32, 97, 18, 139, 233, 159, 167, 142, 227, 249, 111, 192, 3, 184, 162, 109}

func TestDecodeAddress(t *testing.T) {
	type args struct {
		encodedAddr string
	}
	tests := []struct {
		name    string
		args    args
		want    Address
		wantErr bool
	}{
		{"normal", args{"AmMRo3jey9pL5qwJSNJFghe9t9J6fPEoZfNv8XjsqSrajGFuJGpi"}, SampleADdr, false},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeAddress(tt.args.encodedAddr)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DecodeAddress() got = %v, want %v", got, tt.want)
			}
			encoded := EncodeAddress(got)
			assert.Equal(t, encoded, tt.args.encodedAddr)
		})
	}
}
