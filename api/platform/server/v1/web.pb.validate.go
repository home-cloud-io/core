// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: platform/server/v1/web.proto

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

// Validate checks the field values on ShutdownHostRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *ShutdownHostRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ShutdownHostRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// ShutdownHostRequestMultiError, or nil if none found.
func (m *ShutdownHostRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *ShutdownHostRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return ShutdownHostRequestMultiError(errors)
	}

	return nil
}

// ShutdownHostRequestMultiError is an error wrapping multiple validation
// errors returned by ShutdownHostRequest.ValidateAll() if the designated
// constraints aren't met.
type ShutdownHostRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ShutdownHostRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ShutdownHostRequestMultiError) AllErrors() []error { return m }

// ShutdownHostRequestValidationError is the validation error returned by
// ShutdownHostRequest.Validate if the designated constraints aren't met.
type ShutdownHostRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ShutdownHostRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ShutdownHostRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ShutdownHostRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ShutdownHostRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ShutdownHostRequestValidationError) ErrorName() string {
	return "ShutdownHostRequestValidationError"
}

// Error satisfies the builtin error interface
func (e ShutdownHostRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sShutdownHostRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ShutdownHostRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ShutdownHostRequestValidationError{}

// Validate checks the field values on ShutdownHostResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *ShutdownHostResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ShutdownHostResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// ShutdownHostResponseMultiError, or nil if none found.
func (m *ShutdownHostResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *ShutdownHostResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return ShutdownHostResponseMultiError(errors)
	}

	return nil
}

// ShutdownHostResponseMultiError is an error wrapping multiple validation
// errors returned by ShutdownHostResponse.ValidateAll() if the designated
// constraints aren't met.
type ShutdownHostResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ShutdownHostResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ShutdownHostResponseMultiError) AllErrors() []error { return m }

// ShutdownHostResponseValidationError is the validation error returned by
// ShutdownHostResponse.Validate if the designated constraints aren't met.
type ShutdownHostResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ShutdownHostResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ShutdownHostResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ShutdownHostResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ShutdownHostResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ShutdownHostResponseValidationError) ErrorName() string {
	return "ShutdownHostResponseValidationError"
}

// Error satisfies the builtin error interface
func (e ShutdownHostResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sShutdownHostResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ShutdownHostResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ShutdownHostResponseValidationError{}

// Validate checks the field values on RestartHostRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *RestartHostRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on RestartHostRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// RestartHostRequestMultiError, or nil if none found.
func (m *RestartHostRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *RestartHostRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return RestartHostRequestMultiError(errors)
	}

	return nil
}

// RestartHostRequestMultiError is an error wrapping multiple validation errors
// returned by RestartHostRequest.ValidateAll() if the designated constraints
// aren't met.
type RestartHostRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m RestartHostRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m RestartHostRequestMultiError) AllErrors() []error { return m }

// RestartHostRequestValidationError is the validation error returned by
// RestartHostRequest.Validate if the designated constraints aren't met.
type RestartHostRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e RestartHostRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e RestartHostRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e RestartHostRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e RestartHostRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e RestartHostRequestValidationError) ErrorName() string {
	return "RestartHostRequestValidationError"
}

// Error satisfies the builtin error interface
func (e RestartHostRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sRestartHostRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = RestartHostRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = RestartHostRequestValidationError{}

// Validate checks the field values on RestartHostResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *RestartHostResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on RestartHostResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// RestartHostResponseMultiError, or nil if none found.
func (m *RestartHostResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *RestartHostResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return RestartHostResponseMultiError(errors)
	}

	return nil
}

// RestartHostResponseMultiError is an error wrapping multiple validation
// errors returned by RestartHostResponse.ValidateAll() if the designated
// constraints aren't met.
type RestartHostResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m RestartHostResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m RestartHostResponseMultiError) AllErrors() []error { return m }

// RestartHostResponseValidationError is the validation error returned by
// RestartHostResponse.Validate if the designated constraints aren't met.
type RestartHostResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e RestartHostResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e RestartHostResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e RestartHostResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e RestartHostResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e RestartHostResponseValidationError) ErrorName() string {
	return "RestartHostResponseValidationError"
}

// Error satisfies the builtin error interface
func (e RestartHostResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sRestartHostResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = RestartHostResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = RestartHostResponseValidationError{}

// Validate checks the field values on InstallAppRequest with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *InstallAppRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on InstallAppRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// InstallAppRequestMultiError, or nil if none found.
func (m *InstallAppRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *InstallAppRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Chart

	// no validation rules for Repo

	// no validation rules for Release

	// no validation rules for Values

	if len(errors) > 0 {
		return InstallAppRequestMultiError(errors)
	}

	return nil
}

// InstallAppRequestMultiError is an error wrapping multiple validation errors
// returned by InstallAppRequest.ValidateAll() if the designated constraints
// aren't met.
type InstallAppRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m InstallAppRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m InstallAppRequestMultiError) AllErrors() []error { return m }

// InstallAppRequestValidationError is the validation error returned by
// InstallAppRequest.Validate if the designated constraints aren't met.
type InstallAppRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e InstallAppRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e InstallAppRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e InstallAppRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e InstallAppRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e InstallAppRequestValidationError) ErrorName() string {
	return "InstallAppRequestValidationError"
}

// Error satisfies the builtin error interface
func (e InstallAppRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sInstallAppRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = InstallAppRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = InstallAppRequestValidationError{}

// Validate checks the field values on InstallAppResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *InstallAppResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on InstallAppResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// InstallAppResponseMultiError, or nil if none found.
func (m *InstallAppResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *InstallAppResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return InstallAppResponseMultiError(errors)
	}

	return nil
}

// InstallAppResponseMultiError is an error wrapping multiple validation errors
// returned by InstallAppResponse.ValidateAll() if the designated constraints
// aren't met.
type InstallAppResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m InstallAppResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m InstallAppResponseMultiError) AllErrors() []error { return m }

// InstallAppResponseValidationError is the validation error returned by
// InstallAppResponse.Validate if the designated constraints aren't met.
type InstallAppResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e InstallAppResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e InstallAppResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e InstallAppResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e InstallAppResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e InstallAppResponseValidationError) ErrorName() string {
	return "InstallAppResponseValidationError"
}

// Error satisfies the builtin error interface
func (e InstallAppResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sInstallAppResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = InstallAppResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = InstallAppResponseValidationError{}

// Validate checks the field values on DeleteAppRequest with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *DeleteAppRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on DeleteAppRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// DeleteAppRequestMultiError, or nil if none found.
func (m *DeleteAppRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *DeleteAppRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Release

	if len(errors) > 0 {
		return DeleteAppRequestMultiError(errors)
	}

	return nil
}

// DeleteAppRequestMultiError is an error wrapping multiple validation errors
// returned by DeleteAppRequest.ValidateAll() if the designated constraints
// aren't met.
type DeleteAppRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m DeleteAppRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m DeleteAppRequestMultiError) AllErrors() []error { return m }

// DeleteAppRequestValidationError is the validation error returned by
// DeleteAppRequest.Validate if the designated constraints aren't met.
type DeleteAppRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e DeleteAppRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e DeleteAppRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e DeleteAppRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e DeleteAppRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e DeleteAppRequestValidationError) ErrorName() string { return "DeleteAppRequestValidationError" }

// Error satisfies the builtin error interface
func (e DeleteAppRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sDeleteAppRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = DeleteAppRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = DeleteAppRequestValidationError{}

// Validate checks the field values on DeleteAppResponse with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *DeleteAppResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on DeleteAppResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// DeleteAppResponseMultiError, or nil if none found.
func (m *DeleteAppResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *DeleteAppResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return DeleteAppResponseMultiError(errors)
	}

	return nil
}

// DeleteAppResponseMultiError is an error wrapping multiple validation errors
// returned by DeleteAppResponse.ValidateAll() if the designated constraints
// aren't met.
type DeleteAppResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m DeleteAppResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m DeleteAppResponseMultiError) AllErrors() []error { return m }

// DeleteAppResponseValidationError is the validation error returned by
// DeleteAppResponse.Validate if the designated constraints aren't met.
type DeleteAppResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e DeleteAppResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e DeleteAppResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e DeleteAppResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e DeleteAppResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e DeleteAppResponseValidationError) ErrorName() string {
	return "DeleteAppResponseValidationError"
}

// Error satisfies the builtin error interface
func (e DeleteAppResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sDeleteAppResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = DeleteAppResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = DeleteAppResponseValidationError{}

// Validate checks the field values on UpdateAppRequest with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *UpdateAppRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on UpdateAppRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// UpdateAppRequestMultiError, or nil if none found.
func (m *UpdateAppRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *UpdateAppRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Chart

	// no validation rules for Repo

	// no validation rules for Release

	// no validation rules for Values

	if len(errors) > 0 {
		return UpdateAppRequestMultiError(errors)
	}

	return nil
}

// UpdateAppRequestMultiError is an error wrapping multiple validation errors
// returned by UpdateAppRequest.ValidateAll() if the designated constraints
// aren't met.
type UpdateAppRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m UpdateAppRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m UpdateAppRequestMultiError) AllErrors() []error { return m }

// UpdateAppRequestValidationError is the validation error returned by
// UpdateAppRequest.Validate if the designated constraints aren't met.
type UpdateAppRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e UpdateAppRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e UpdateAppRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e UpdateAppRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e UpdateAppRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e UpdateAppRequestValidationError) ErrorName() string { return "UpdateAppRequestValidationError" }

// Error satisfies the builtin error interface
func (e UpdateAppRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sUpdateAppRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = UpdateAppRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = UpdateAppRequestValidationError{}

// Validate checks the field values on UpdateAppResponse with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *UpdateAppResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on UpdateAppResponse with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// UpdateAppResponseMultiError, or nil if none found.
func (m *UpdateAppResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *UpdateAppResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return UpdateAppResponseMultiError(errors)
	}

	return nil
}

// UpdateAppResponseMultiError is an error wrapping multiple validation errors
// returned by UpdateAppResponse.ValidateAll() if the designated constraints
// aren't met.
type UpdateAppResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m UpdateAppResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m UpdateAppResponseMultiError) AllErrors() []error { return m }

// UpdateAppResponseValidationError is the validation error returned by
// UpdateAppResponse.Validate if the designated constraints aren't met.
type UpdateAppResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e UpdateAppResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e UpdateAppResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e UpdateAppResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e UpdateAppResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e UpdateAppResponseValidationError) ErrorName() string {
	return "UpdateAppResponseValidationError"
}

// Error satisfies the builtin error interface
func (e UpdateAppResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sUpdateAppResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = UpdateAppResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = UpdateAppResponseValidationError{}