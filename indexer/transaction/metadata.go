package transaction

import (
	"encoding/json"
)

func UnmarshalMetadataVerifyToken(metadata string) (contractAddr, owner, comment, email, regDate, homepageUrl, imageUrl string) {
	var args map[string]interface{}
	err := json.Unmarshal([]byte(metadata), &args)
	if err != nil {
		return "", "", "", "", "", "", ""
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

func UnmarshalMetadataVerifyContract(metadata string) (contractAddr, owner, codeUrl string) {
	var args map[string]interface{}
	err := json.Unmarshal([]byte(metadata), &args)
	if err != nil {
		return "", "", ""
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
	return contractAddr, owner, codeUrl
}
