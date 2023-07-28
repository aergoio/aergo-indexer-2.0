package lua_compiler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mr-tron/base58"
)

const (
	url = "https://luac.aergo.io/compile"
)

func Compile(code string) ([]byte, error) {
	data := []byte(base64.StdEncoding.EncodeToString([]byte(code)))
	resp, err := http.Post(url, "text/plain", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	retRaw, err := ioutil.ReadAll(resp.Body)
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

	// trim header and footer experimentally : Need more test
	if len(byteCode) > 5 {
		byteCode = byteCode[1 : len(byteCode)-4]
	}

	return byteCode, nil
}
