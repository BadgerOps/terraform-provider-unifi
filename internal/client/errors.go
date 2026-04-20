package client

import (
	"errors"
	"fmt"
	"net/http"
)

type Error struct {
	StatusCode int
	Code       string
	Message    string
	Body       string
}

func (e *Error) Error() string {
	switch {
	case e == nil:
		return ""
	case e.Code != "" && e.Message != "":
		return fmt.Sprintf("unifi API error: status=%d code=%s message=%s", e.StatusCode, e.Code, e.Message)
	case e.Message != "":
		return fmt.Sprintf("unifi API error: status=%d message=%s", e.StatusCode, e.Message)
	default:
		return fmt.Sprintf("unifi API error: status=%d body=%s", e.StatusCode, e.Body)
	}
}

func IsNotFound(err error) bool {
	var clientErr *Error
	return errors.As(err, &clientErr) && clientErr.StatusCode == http.StatusNotFound
}

type MissingClientError struct {
	SiteID     string
	MACAddress string
}

func (e *MissingClientError) Error() string {
	if e == nil {
		return ""
	}

	return fmt.Sprintf(
		"client with MAC %s was not found in UniFi site %s; the legacy DHCP reservation API only accepts updates for clients that already exist in the controller database, so retry after the client appears",
		e.MACAddress,
		e.SiteID,
	)
}

func IsMissingClient(err error) bool {
	var missingClientErr *MissingClientError
	return errors.As(err, &missingClientErr)
}
