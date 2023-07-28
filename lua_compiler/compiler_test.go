package lua_compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompile(t *testing.T) {
	code := readLuaCode("type_arrayarg.lua")
	byteCode, err := Compile(code)
	require.NoError(t, err)
	fmt.Println(len(byteCode), byteCode)
	fmt.Println(string(byteCode))
}

// utility function for tests
func readLuaCode(file string) (luaCode string) {
	_, filename, _, ok := runtime.Caller(0)
	if ok != true {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "test_files", file))
	if err != nil {
		return ""
	}
	return string(raw)
}
