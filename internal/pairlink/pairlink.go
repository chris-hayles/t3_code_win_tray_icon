package pairlink

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var ErrNoPairingURL = errors.New("no Pairing URL line in output")

// ParsePairingURLLine finds a line "Pairing URL: <url>" in CLI output.
func ParsePairingURLLine(output string) (string, error) {
	const prefix = "Pairing URL:"
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			u := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			if u == "" {
				return "", fmt.Errorf("empty pairing url")
			}
			return u, nil
		}
	}
	return "", ErrNoPairingURL
}

// BuildAppLink builds t3code://pair?pairingUrl=... with proper query encoding.
func BuildAppLink(pairingURL string) (string, error) {
	if pairingURL == "" {
		return "", errors.New("empty pairing url")
	}
	v := url.Values{}
	v.Set("pairingUrl", pairingURL)
	return "t3code://pair?" + v.Encode(), nil
}
