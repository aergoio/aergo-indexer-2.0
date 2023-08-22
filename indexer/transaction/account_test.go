package transaction

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountEncodeDecode(t *testing.T) {
	fn_test := func(account string) {
		enc := DecodeAccount(account)
		dec := EncodeAccount(enc)
		require.Equal(t, account, dec)
	}

	fn_test("aergo.system")
	fn_test("aergo.vault")
	fn_test("aergossystem")
	fn_test("AmPxDZYmc7f6cTEzv8rTkbDxYtziTkRQEBbCW9r5GgVntpSmgXWb")
	fn_test("AmMK3LZiR1oEf66xzXir7mA5SUVVHSinWUYmh5FwueoVmciH3CuJ")
	fn_test("AmMjrVRQbrgDYChnWgyYL6gfneGT5ui6DwuvUXp8nTdUz8wwstAq")
	fn_test("AmPERyyJgoDLm6GBTEaVSwennQeyQGDSFycGUuSMsvupL1qKfTFo")
}
