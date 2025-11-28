// Package lnkerr provides a single error wrapper type for the lnk application.
package lnkerr

// Error wraps a sentinel error with optional context for display.
// This is the only custom error type in the codebase.
type Error struct {
	Err        error  // Underlying sentinel error
	Path       string // Optional path for display
	Suggestion string // Optional suggestion for user
}

func (e *Error) Error() string {
	msg := e.Err.Error()
	if e.Path != "" {
		msg += ": " + e.Path
	}
	if e.Suggestion != "" {
		msg += " (" + e.Suggestion + ")"
	}
	return msg
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Wrap creates an Error with just the sentinel.
func Wrap(err error) *Error {
	return &Error{Err: err}
}

// WithPath creates an Error with path context.
func WithPath(err error, path string) *Error {
	return &Error{Err: err, Path: path}
}

// WithSuggestion creates an Error with a suggestion.
func WithSuggestion(err error, suggestion string) *Error {
	return &Error{Err: err, Suggestion: suggestion}
}

// WithPathAndSuggestion creates an Error with both.
func WithPathAndSuggestion(err error, path, suggestion string) *Error {
	return &Error{Err: err, Path: path, Suggestion: suggestion}
}
