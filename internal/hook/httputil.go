package hook

import (
	"fmt"
	"io"
	"net/http"
)

func post(url string, contentType string, body io.Reader) (string, error) {
	r, err := http.Post(url, contentType, body)
	if err != nil {
		return "", fmt.Errorf("send request: %w", url, err)
	}
	if r.StatusCode == 204 {
		return "", nil
	} else if r.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status %v: %s", r.StatusCode, r.Status)
	}
	defer r.Body.Close()
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	return string(bodyBytes), nil
}
