package syncapi

import (
	"errors"
	"net/url"
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

	return u.Scheme == "backrest"
}

func InstanceForBackrestURI(repoUri string) (string, error) {
	u, err := url.Parse(repoUri)
	if err != nil {
		return "", err
	}

	if u.Scheme != "backrest" {
		return "", errors.New("not a backrest URI")
	}

	return u.Hostname(), nil
}
