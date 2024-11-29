package syncengine

import (
	"errors"
	"net/url"
	"strings"
)

func CreateRemoteRepoURI(instanceUrl string) (string, error) {
	u, err := url.Parse(instanceUrl)
	if err != nil {
		return "", err
	}

	if u.Scheme == "http" {
		u.Scheme = "backrest"
	} else if u.Scheme == "https" {
		u.Scheme = "sbackrest"
	} else {
		return "", errors.New("unsupported scheme")
	}

	return u.String(), nil
}

func IsBackrestRemoteRepoURI(repoUri string) bool {
	u, err := url.Parse(repoUri)
	if err != nil {
		return false
	}

	return u.Scheme == "backrest" || u.Scheme == "sbackrest"
}

func backrestRemoteUrlToHTTPUrl(repoUri string) string {
	if strings.HasPrefix(repoUri, "backrest:") {
		return "http://" + strings.TrimPrefix(repoUri, "backrest:")
	} else if strings.HasPrefix(repoUri, "sbackrest:") {
		return "https://" + strings.TrimPrefix(repoUri, "sbackrest:")
	} else {
		return repoUri
	}
}
