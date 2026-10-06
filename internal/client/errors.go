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
		return fmt.Sprintf("unifi API error: status=%d code=%s message=%s%s", e.StatusCode, e.Code, e.Message, e.bodySuffix())
	case e.Message != "":
		return fmt.Sprintf("unifi API error: status=%d message=%s%s", e.StatusCode, e.Message, e.bodySuffix())
	default:
		return fmt.Sprintf("unifi API error: status=%d body=%s", e.StatusCode, e.Body)
	}
}

// bodySuffix keeps controller detail (for example a legacy validationError)
// visible in diagnostics when the body says more than the code and message.
func (e *Error) bodySuffix() string {
	if e.Body == "" || e.Body == e.Message {
		return ""
	}
	return " body=" + e.Body
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
