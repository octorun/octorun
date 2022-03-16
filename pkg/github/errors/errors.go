/*
Copyright 2022 The Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package errors

import (
	"net/http"

	"github.com/google/go-github/v41/github"
)

// IsReteLimit returns true if given error is github.RateLimitError
func IsReteLimit(err error) bool {
	if _, ok := err.(*github.RateLimitError); ok {
		return true
	}

	return false
}

// IsBadRequest returns true if given error is github.ErrorResponse
// and http response status code is http.StatusBadRequest
func IsBadRequest(err error) bool {
	if rerr := parseErrorResponse(err); rerr != nil {
		return rerr.StatusCode == http.StatusBadRequest
	}

	return false
}

// IsUnauthorized returns true if given error is github.ErrorResponse
// and http response status code is http.StatusUnauthorized
func IsUnauthorized(err error) bool {
	if rerr := parseErrorResponse(err); rerr != nil {
		return rerr.StatusCode == http.StatusUnauthorized
	}

	return false
}

// IsForbidden returns true if given error is github.ErrorResponse
// and http response status code is http.StatusForbidden
func IsForbidden(err error) bool {
	if rerr, ok := err.(*github.ErrorResponse); ok {
		return rerr.Response.StatusCode == http.StatusForbidden
	}

	return false
}

// IsNotFound returns true if given error is github.ErrorResponse
// and http response status code is http.StatusNotFound
func IsNotFound(err error) bool {
	if rerr := parseErrorResponse(err); rerr != nil {
		return rerr.StatusCode == http.StatusNotFound
	}

	return false
}

// IsRequestTimeout returns true if given error is github.ErrorResponse
// and http response status code is http.StatusRequestTimeout
func IsRequestTimeout(err error) bool {
	if rerr := parseErrorResponse(err); rerr != nil {
		return rerr.StatusCode == http.StatusRequestTimeout
	}

	return false
}

// IsConflict returns true if given error is github.ErrorResponse
// and http response status code is http.StatusConflict
func IsConflict(err error) bool {
	if rerr := parseErrorResponse(err); rerr != nil {
		return rerr.StatusCode == http.StatusConflict
	}

	return false
}

// IsGone returns true if given error is github.ErrorResponse
// and http response status code is http.StatusGone
func IsGone(err error) bool {
	if rerr := parseErrorResponse(err); rerr != nil {
		return rerr.StatusCode == http.StatusGone
	}

	return false
}

func parseErrorResponse(err error) *http.Response {
	if rerr, ok := err.(*github.ErrorResponse); ok {
		return rerr.Response
	}

	return nil
}
