package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
)

// Error message from Jira
// See https://docs.atlassian.com/jira/REST/cloud/#error-responses
type Error struct {
	HTTPError     error
	ErrorMessages []string          `json:"errorMessages"`
	Errors        map[string]string `json:"errors"`
}

// NewJiraError creates an error from an unsuccessful response returned by the Jira
// API. It is the caller's responsibility to ensure that the given response
// represents a failed API request.
//
// If the response body contains a JSON-encoded error from the Jira API, the error
// message is included in the returned error. If the response body contains non-JSON
// data (e.g. XML, due to a non-API-related Jira failure), a generic error is
// returned; the caller is then responsible for analysing the request body to
// determine the cause of the failure.
func NewJiraError(resp *Response, httpError error) error {
	if resp == nil {
		return errors.Wrap(httpError, "No response returned")
	}

	// Read the response body (so we can parse it), but allow the caller to read it
	// for themselves should they want to. This is inefficient if the response body
	// is large (because it resides in memory for as long as the response is in
	// scope), but failure-case responses from Jira shouldn't be large enough to
	// cause significant problems.
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	resp.Body = ioutil.NopCloser(bytes.NewReader(body))
	if err != nil {
		return errors.Wrap(err, httpError.Error())
	}
	jerr := Error{HTTPError: httpError}
	contentType := resp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		err = json.Unmarshal(body, &jerr)
		if err != nil {
			httpError = errors.Wrap(errors.New("could not parse JSON"), httpError.Error())
			return errors.Wrap(err, httpError.Error())
		}
	} else {
		if httpError == nil {
			return fmt.Errorf("got response status %s:%s", resp.Status, string(body))
		}
		return errors.Wrap(httpError, fmt.Sprintf("%s: %s", resp.Status, string(body)))
	}

	return &jerr
}

// Error is a short string representing the error
func (e *Error) Error() string {
	if len(e.ErrorMessages) > 0 {
		// return fmt.Sprintf("%v", e.HTTPError)
		return fmt.Sprintf("%s: %v", e.ErrorMessages[0], e.HTTPError)
	}
	if len(e.Errors) > 0 {
		for key, value := range e.Errors {
			return fmt.Sprintf("%s - %s: %v", key, value, e.HTTPError)
		}
	}
	return e.HTTPError.Error()
}

// LongError is a full representation of the error as a string
func (e *Error) LongError() string {
	var msg bytes.Buffer
	if e.HTTPError != nil {
		msg.WriteString("Original:\n")
		msg.WriteString(e.HTTPError.Error())
		msg.WriteString("\n")
	}
	if len(e.ErrorMessages) > 0 {
		msg.WriteString("Messages:\n")
		for _, v := range e.ErrorMessages {
			msg.WriteString(" - ")
			msg.WriteString(v)
			msg.WriteString("\n")
		}
	}
	if len(e.Errors) > 0 {
		for key, value := range e.Errors {
			msg.WriteString(" - ")
			msg.WriteString(key)
			msg.WriteString(" - ")
			msg.WriteString(value)
			msg.WriteString("\n")
		}
	}
	return msg.String()
}
