package syncengine

import (
	"errors"
	"net/url"
)

func CreateRemoteRepoURI(instanceUrl, keyID, keySecret string) (string, error) {
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

	u.User = url.UserPassword(keyID, keySecret)

	return u.String(), nil
}

func ParseRemoteRepoURI(repoUri string) (instanceUrl, keyID, keySecret string, err error) {
	u, err := url.Parse(repoUri)
	if err != nil {
		return "", "", "", err
	}

	if u.Scheme == "backrest" {
		u.Scheme = "http"
	} else if u.Scheme == "sbackrest" {
		u.Scheme = "https"
	}

	instanceUrl = u.Scheme + "://" + u.Host + "/" + u.Path

	if u.User.Username() != "" {
		keyID = u.User.Username()
	} else {
		return "", "", "", errors.New("missing key ID")
	}

	if password, ok := u.User.Password(); ok {
		keySecret = password
	} else {
		return "", "", "", errors.New("missing key secret")
	}

	return
}

func IsBackrestRemoteRepoURI(repoUri string) bool {
	u, err := url.Parse(repoUri)
	if err != nil {
		return false
	}

	return u.Scheme == "backrest" || u.Scheme == "sbackrest"
}
