package vault

import (
	"fmt"
	"log"
)

// changepassSecEng is a SecEng implementation which works for Vault secrets engines listening to
// endpoint .../creds/rolename for performing a password change. Currently, secrets engines of type
// ad and ontap belong to this implementation. This secrets engines don't work with leases and
// there is nothing like a lease duration or lease revocation to worry about.
type changepassSecEng struct {
	name           string
	changeCredsURL string
	storeDataURL   string
}

func (secEng changepassSecEng) getName() string {
	return secEng.name
}

// startBooking for a changepassSecEng means to change the credentials and store it inside the respective
// kv secret engine inside Vault.
func (secEng changepassSecEng) startBooking(vaultToken, _ string, _ int) {
	data := fmt.Sprintf("{\"data\": \"%v\"}", secEng.changeCreds(vaultToken))
	log.Println(data)
	vaultStorageWrite(vaultToken, secEng.storeDataURL, data)
}

// endBooking for a changepassSecEng means to delete the stored credentials from kv storage and then
// change the credentials again for them to become unknown.
func (secEng changepassSecEng) endBooking(vaultToken string) {
	vaultStorageDelete(vaultToken, secEng.storeDataURL)
	log.Println(secEng.changeCreds(vaultToken))
}

func (secEng changepassSecEng) readCreds(vaultToken string) (interface{}, error) {
	return vaultStorageRead(vaultToken, secEng.storeDataURL)
}

func (secEng changepassSecEng) changeCreds(vaultToken string) interface{} {

	data, err := sendVaultDataRequest("GET", secEng.changeCredsURL, vaultToken, nil)
	if err != nil {
		log.Println(err)
	}
	return data
}
