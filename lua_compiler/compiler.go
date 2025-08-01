package lua_compiler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/mr-tron/base58"
)

const (
	url = "https://luac.aergo.io/compile"
)

func GetCode(url string) (code string, err error) {
	// HTTP GET 요청 보내기
	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("Error sending request: %v", err)
	}
	defer response.Body.Close()

	// 응답 본문 읽기
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("Error reading response body: %v", err)
	}

	// 문자열로 변환하여 출력
	content := string(body)
	return content, nil
}

// CompileCodeLocal compiles Lua code using the local aergoluac binary
func CompileCodeLocal(code string) ([]byte, error) {
	// Create a temporary file for the Lua code
	tmpFile, err := os.CreateTemp("", "contract-*.lua")
	if err != nil {
		return nil, fmt.Errorf("Error creating temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write the Lua code to the temporary file
	_, err = tmpFile.WriteString(code)
	if err != nil {
		return nil, fmt.Errorf("Error writing to temp file: %v", err)
	}

	// Let the OS search for aergoluac in PATH
	aergoluacPath, err := exec.LookPath("aergoluac")
	if err != nil {
		return nil, fmt.Errorf("aergoluac binary not found in PATH: %v", err)
	}

	// Execute the aergoluac command
	cmd := exec.Command(aergoluacPath, "--payload", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Error executing aergoluac: %v, output: %s", err, string(output))
	}

	// The output should be base58 encoded bytecode
	byteCode, err := base58.Decode(strings.TrimSpace(string(output)))
	if err != nil {
		return nil, fmt.Errorf("Error decoding base58 output: %v", err)
	}

	// trim header and footer
	if len(byteCode) > 5 {
		byteCode = byteCode[1 : len(byteCode)-4]
	}

	return byteCode, nil
}

func CompileCodeRemote(code string) ([]byte, error) {
	data := []byte(base64.StdEncoding.EncodeToString([]byte(code)))
	resp, err := http.Post(url, "text/plain", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	retRaw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %v", err)
	}
	retData := string(retRaw)
	if len(retData) < 8 || retData[:8] != "result: " {
		return nil, fmt.Errorf("Error in response: %v", retData)
	}

	// trim space - result:
	byteCodeB58 := strings.TrimSpace(retData[8:])
	byteCode, err := base58.Decode(byteCodeB58)
	if err != nil {
		return nil, fmt.Errorf("Error decoding b58 byte code: %v", err)
	}

	// trim header and footer
	if len(byteCode) > 5 {
		byteCode = byteCode[1 : len(byteCode)-4]
	}

	return byteCode, nil
}

/*
// This requires C dependencies, so we use the external compiler method for now
func CompileCode(code string) ([]byte, error) {
	// Use the luac package to compile the Lua code directly
	result, err := luac.Compile(nil, code)
	if err != nil {
		return nil, fmt.Errorf("Error compiling Lua code: %v", err)
	}

	byteCode := result.ByteCode()
	return byteCode, nil
}
*/

func CompileCode(code string) ([]byte, error) {
	// Try local compiler first
	byteCodeABI, err := CompileCodeLocal(code)
	if err == nil {
		return byteCodeABI, nil
	}

	// If local compiler fails, try remote compiler
	return CompileCodeRemote(code)
}
