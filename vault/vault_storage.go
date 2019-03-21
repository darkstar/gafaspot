package vault

import (
	"bytes"
	"log"
)

func vaultStorageWrite(vaultToken, url string, data []byte) {

	err := sendVaultRequestEmtpyResponse("POST", url, vaultToken, bytes.NewReader(data))
	if err != nil {
		log.Println(err)
	}
}

func vaultStorageRead(vaultToken, url string) (map[string]interface{}, error) {
	return sendVaultDataRequest("GET", url, vaultToken, nil)
}

func vaultStorageDelete(vaultToken, url string) {
	err := sendVaultRequestEmtpyResponse("DELETE", url, vaultToken, nil)
	if err != nil {
		log.Println(err)
	}
}
