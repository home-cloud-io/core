// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: platform/daemon/v1/host.proto

package v1

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
	_ = sort.Sort
)

// Validate checks the field values on ShutdownAlertRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *ShutdownAlertRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ShutdownAlertRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// ShutdownAlertRequestMultiError, or nil if none found.
func (m *ShutdownAlertRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *ShutdownAlertRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return ShutdownAlertRequestMultiError(errors)
	}

	return nil
}

// ShutdownAlertRequestMultiError is an error wrapping multiple validation
// errors returned by ShutdownAlertRequest.ValidateAll() if the designated
// constraints aren't met.
type ShutdownAlertRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ShutdownAlertRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ShutdownAlertRequestMultiError) AllErrors() []error { return m }

// ShutdownAlertRequestValidationError is the validation error returned by
// ShutdownAlertRequest.Validate if the designated constraints aren't met.
type ShutdownAlertRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ShutdownAlertRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ShutdownAlertRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ShutdownAlertRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ShutdownAlertRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ShutdownAlertRequestValidationError) ErrorName() string {
	return "ShutdownAlertRequestValidationError"
}

// Error satisfies the builtin error interface
func (e ShutdownAlertRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sShutdownAlertRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ShutdownAlertRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ShutdownAlertRequestValidationError{}

// Validate checks the field values on ShutdownAlertResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *ShutdownAlertResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ShutdownAlertResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// ShutdownAlertResponseMultiError, or nil if none found.
func (m *ShutdownAlertResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *ShutdownAlertResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return ShutdownAlertResponseMultiError(errors)
	}

	return nil
}

// ShutdownAlertResponseMultiError is an error wrapping multiple validation
// errors returned by ShutdownAlertResponse.ValidateAll() if the designated
// constraints aren't met.
type ShutdownAlertResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ShutdownAlertResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ShutdownAlertResponseMultiError) AllErrors() []error { return m }

// ShutdownAlertResponseValidationError is the validation error returned by
// ShutdownAlertResponse.Validate if the designated constraints aren't met.
type ShutdownAlertResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ShutdownAlertResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ShutdownAlertResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ShutdownAlertResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ShutdownAlertResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ShutdownAlertResponseValidationError) ErrorName() string {
	return "ShutdownAlertResponseValidationError"
}

// Error satisfies the builtin error interface
func (e ShutdownAlertResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sShutdownAlertResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ShutdownAlertResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ShutdownAlertResponseValidationError{}