package errforge

import (
	"bytes"
	"log/slog"
	"strings"

	"github.com/rs/zerolog"
	"go.uber.org/zap/zapcore"
)

// Params are the raw values a user-facing message is built from, keyed by
// name: what the client needs to compute "8 is too high, the maximum is 5" in
// the user's language when the English sentence is only a fallback. Every
// value reaches the client, so put in it only what a user may see; the
// internal data an operator needs belongs in the logs, not here. Values must
// be JSON-encodable for [WriteErrorJSON] to carry them.
type Params map[string]any

// PublicError represents a well designed error to be addressed to users.
//
// The five sentences below are the English message. Key and Params make the
// same error translatable: a client that knows the user's locale looks the
// key up in its own catalog and formats it with the raw values, and falls
// back to Message when it has no translation.
type PublicError struct {
	// Key is a stable identifier the client translates from, such as
	// "order.out_of_stock". Empty when the error is not translatable.
	Key string
	// Params are the values the message is computed from, such as
	// {"requested": 8, "available": 5}. Nil when the message has none.
	Params Params
	// Message says what happened, e.g., "unable to connect to your account".
	// When Key is set it is the fallback shown when no translation exists.
	Message string
	// Reassurance provides reassurance, e.g., "your changes were saved".
	Reassurance string
	// Reason says why the error happened, e.g., "we could not connect your account due to a technical issue on our end".
	Reason string
	// Resolution provide guidance to Help the user fix the error, e.g., "please try connecting again".
	Resolution string
	// WayOut gives the user a way out if the resolution failed, e.g., "if the issue keeps happening, contact Customer Care".
	WayOut string
}

// Error returns the error that happened.
func (pe *PublicError) Error() string {
	return pe.Message
}

// Details returns a string containing all the useful
// information for the user.
func (pe *PublicError) Details() string {
	detailStack := pe.detailStack()
	capitalize(&detailStack)
	return strings.Join(detailStack, ". ")
}

// LogValue implements the slog.LogValuer interface to provide
// a clean log of the error when slog is used. The key and the params are
// logged only when set.
func (pe *PublicError) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, numLogFields)
	if pe.Key != "" {
		attrs = append(attrs, slog.String("key", pe.Key))
	}
	attrs = append(attrs,
		slog.String("message", pe.Message),
		slog.String("details", pe.Details()),
	)
	if len(pe.Params) > 0 {
		attrs = append(attrs, slog.Any("params", pe.Params))
	}
	return slog.GroupValue(attrs...)
}

// MarshalLogObject implements the zapcore.ObjectMarshaler interface to provide
// a clean log of the error when zap is used. The key and the params are
// logged only when set.
func (pe *PublicError) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	if pe.Key != "" {
		encoder.AddString("key", pe.Key)
	}
	encoder.AddString("message", pe.Message)
	encoder.AddString("details", pe.Details())
	if len(pe.Params) > 0 {
		return encoder.AddReflected("params", pe.Params)
	}
	return nil
}

// MarshalZerologObject implements the zerolog.LogObjectMarshaler interface to provide
// a clean log of the error when zerolog is used. The key and the params are
// logged only when set.
func (pe *PublicError) MarshalZerologObject(e *zerolog.Event) {
	if pe.Key != "" {
		e.Str("key", pe.Key)
	}
	e.Str("message", pe.Message)
	e.Str("details", pe.Details())
	if len(pe.Params) > 0 {
		e.Interface("params", pe.Params)
	}
}

// numLogFields is the number of fields a PublicError can log (key, message,
// details, params).
const numLogFields = 4

// numDetailFields is the number of optional detail fields a PublicError can
// contribute to its detail stack (Reassurance, Reason, Resolution, WayOut).
const numDetailFields = 4

func (pe *PublicError) detailStack() []string {
	details := make([]string, 0, numDetailFields)
	if pe.Reassurance != "" {
		details = append(details, pe.Reassurance)
	}
	if pe.Reason != "" {
		details = append(details, pe.Reason)
	}
	if pe.Resolution != "" {
		details = append(details, pe.Resolution)
	}
	if pe.WayOut != "" {
		details = append(details, pe.WayOut)
	}
	return details
}

func capitalize(detailStack *[]string) {
	for i := range *detailStack {
		(*detailStack)[i] = toUpper((*detailStack)[i])
	}
}

func toUpper(str string) string {
	return string(bytes.ToUpper([]byte{str[0]})) + str[1:]
}
