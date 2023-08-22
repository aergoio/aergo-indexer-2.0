package transaction

import (
	"encoding/json"
	"fmt"
)

func UnmarshalMetadataVerifyToken(metadata string) (contractAddr, owner, comment, email, regDate, homepageUrl, imageUrl string, err error) {
	var args map[string]interface{}
	err = json.Unmarshal([]byte(metadata), &args)
	if err != nil {
		return "", "", "", "", "", "", "", fmt.Errorf("err : [%v], metadata : [%s]", err, metadata)
	}

	if field := args["contract"]; field != nil {
		contractAddr, _ = field.(string)
	}
	if field := args["owner"]; field != nil {
		owner, _ = field.(string)
	}
	if field := args["comment"]; field != nil {
		comment, _ = field.(string)
	}
	if field := args["email"]; field != nil {
		email, _ = field.(string)
	}
	if field := args["homepage"]; field != nil {
		homepageUrl, _ = field.(string)
	}
	if field := args["image"]; field != nil {
		imageUrl, _ = field.(string)
	}
	if field := args["regdate"]; field != nil {
		regDate, _ = field.(string)
	}
	return
}

func UnmarshalMetadataVerifyContract(metadata string) (contractAddr, codeUrl, owner string, err error) {
	var args map[string]interface{}
	err = json.Unmarshal([]byte(metadata), &args)
	if err != nil {
		return "", "", "", fmt.Errorf("%v | %s", err, metadata)
	}

	if field := args["contract"]; field != nil {
		contractAddr, _ = field.(string)
	}
	if field := args["owner"]; field != nil {
		owner, _ = field.(string)
	}
	if field := args["code"]; field != nil {
		codeUrl, _ = field.(string)
	}
	return
}
