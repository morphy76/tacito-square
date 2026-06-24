package model

import "errors"

// ErrSessionNotFound is returned when a session cannot be located by its ID.
var ErrSessionNotFound = errors.New("session not found")

// ErrSessionExpired is returned when a session exists but its access token has expired.
var ErrSessionExpired = errors.New("session expired")

// ErrSessionInvalidated is returned when a session has been explicitly invalidated
// (e.g., via logout or backchannel logout).
var ErrSessionInvalidated = errors.New("session invalidated")
