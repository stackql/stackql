package auth

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

const (
	gcloudErrFmt          string = "gcloud error: %s"
	gcloudInstallInfo     string = `Visit https://cloud.google.com/sdk to download the SDK or alternatively use serviceaccount mode, see https://docs.stackql.io/language-spec/auth for more information`
	gcloudNotPresentError string = `Interactive mode requires the Google Cloud SDK to be installed on the system being used for stackql. ` + gcloudInstallInfo
	gcloudRevokeError     string = "Error revoking gcloud credentials.  Please ensure that gcloud is installed and credentials are valid before running this command. " + gcloudInstallInfo
)

func waitErrWrapper(cmd *exec.Cmd, prefixStr string) error {
	return errWrapper(cmd.Wait(), prefixStr)
}

func errWrapper(err error, prefixStr string) error {
	if err != nil {
		err = fmt.Errorf(prefixStr+gcloudErrFmt, err.Error())
	}
	return err
}

func OAuthToGoogle() error {
	cmd := exec.Command("gcloud", "auth", "login")
	fmt.Fprintln(os.Stderr, "Authenticating to Google, a browser window should open...")
	err := cmd.Start()
	if err != nil {
		return errors.New(gcloudNotPresentError)
	}
	return waitErrWrapper(cmd, "")
}

func RevokeGoogleAuth() error {
	cmd := exec.Command("gcloud", "auth", "revoke")
	fmt.Fprintln(os.Stderr, "Revoking Google credentials...")
	err := cmd.Start()
	if err != nil {
		return errors.New(gcloudRevokeError)
	}
	return waitErrWrapper(cmd, `Auth revoke for Google Failed: `)
}

func GetAccessToken() ([]byte, error) {
	re := regexp.MustCompile(`\r?\n`)
	var token []byte
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	response, err := cmd.Output()
	if err == nil && response != nil {
		token = re.ReplaceAll(response, []byte{})
	}
	return token, errWrapper(err, "")
}

func GetCurrentAuthUser() ([]byte, error) {
	re := regexp.MustCompile(`\r?\n`)
	var token []byte
	cmd := exec.Command("gcloud", "auth", "list", "--filter", "status:ACTIVE", "--format", "value(account)")
	response, err := cmd.CombinedOutput()
	if err == nil && response != nil {
		token = re.ReplaceAll(response, []byte{})
	}
	return token, errWrapper(err, "")
}
