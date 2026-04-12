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
