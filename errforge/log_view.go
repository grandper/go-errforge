package errforge

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// logKey is the key under which Attr and ZapField log the error.
const logKey = "error"

// Attr renders err, and everything its tree holds, as a slog attribute under
// the "error" key:
//
//	slog.Error("request failed", errforge.Attr(err))
//
// The attribute is a group with the error's message and, when set, its code,
// transient flag, field name, location, stack, public_error and causes — see
// [LogObject] for the exact shape. It does not matter which node is on top:
// a [ToTransient] or [WithStack] marker, a [Wrap], a [FieldError] or a plain
// [New] over a coded error all log the same thing, which is what
// slog.Any("error", err) cannot promise. [ZapField] and [LogObject] are the
// zap and zerolog renderings of the same view.
//
// Attr(nil) is the empty attribute, which the standard handlers drop, so it is
// safe in a deferred log call. To log under another key, set it on the result:
//
//	a := errforge.Attr(err)
//	a.Key = "err"
func Attr(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.Attr{Key: logKey, Value: newLogView(err).slogValue()}
}

// ZapField renders err as a zap field under the "error" key, with the same
// content as [Attr]:
//
//	zap.L().Error("request failed", errforge.ZapField(err))
//
// ZapField(nil) is zap.Skip(), which zap ignores. To log under another key,
// set it on the result.
func ZapField(err error) zap.Field {
	if err == nil {
		return zap.Skip()
	}
	return zap.Object(logKey, newLogView(err))
}

// LogObject renders err as a zerolog object, with the same content as [Attr]:
//
//	zerolog.Ctx(ctx).Error().Object("error", errforge.LogObject(err)).Msg("request failed")
//
// LogObject(nil) is nil, which zerolog logs as null.
//
// The object describes the whole tree. Its keys, in this order, each omitted
// when empty:
//
//	message       the error's message, always present
//	code          the code GetCode answers for the error, such as "NOT_FOUND"
//	transient     true when IsTransient reports the error retryable
//	field         the input name given by FieldError, nested names joined by a dot
//	location      the frame where the error was created, as logged by StackFrame
//	stack         the stack trace attached with WithStack, one "func (file:line)" per frame
//	public_error  the user-facing PublicError, as logged by PublicError
//	causes        the errors below this one, each rendered the same way
//
// A run of nodes sharing one message — a ToTransient over a Propagate that
// keeps the wording — is one object, its facts merged, exactly as [Print]
// collapses it into one entry; a new entry in causes starts only where the
// message changes. The members of a multierror are each a cause.
func LogObject(err error) zerolog.LogObjectMarshaler {
	if err == nil {
		return nil
	}
	return newLogView(err)
}

// logView is the structured rendering of an error tree that Attr, ZapField,
// LogObject and DetailedError's marshalers all share: one node per run of
// same-message errors, holding what the getters answer at that level, and the
// causes below it rendered the same way.
type logView struct {
	message   string
	code      Code
	transient bool
	field     string
	location  *StackFrame
	stack     StackTrace
	public    *PublicError
	causes    logViews
}

// logViews is a list of causes. It marshals to JSON so that slog, which has
// no array kind, can log it as an array of objects.
type logViews []*logView

// newLogView builds the view of err's tree. err must not be nil.
func newLogView(err error) *logView {
	v := &logView{message: err.Error()}
	// The run is err and the same-message nodes below it, the entry Print
	// collapses. Each fact is taken from the first node of the run that answers
	// it: the getters look down from their node, so a code or a PublicError set
	// below the run is inherited, while a location or a stack is only reported
	// by the node that carries it.
	last := err
	for node := err; node != nil; node = transparentCause(node) {
		last = node
		if fe, ok := node.(*fieldError); ok { //nolint:errorlint // the loop steps through the run node by node
			v.field = joinField(v.field, fe.field)
		}
		if v.code == NoCode {
			v.code = GetCode(node)
		}
		if !v.transient {
			v.transient = IsTransient(node)
		}
		if v.public == nil {
			v.public = GetDetails(node)
		}
		if located, ok := node.(interface{ Location() *StackFrame }); ok && v.location == nil {
			v.location = located.Location()
		}
		st, ok := node.(StackTracer) //nolint:errorlint // only the node that carries the stack reports it
		if ok && v.stack == nil {
			v.stack = st.StackTrace()
		}
	}
	for _, child := range children(last) {
		if child != nil {
			v.causes = append(v.causes, newLogView(child))
		}
	}
	return v
}

// joinField appends a nested field name to its parent's, so that
// FieldError("address", FieldError("city", err)) reads "address.city".
func joinField(parent, field string) string {
	if parent == "" {
		return field
	}
	return parent + "." + field
}

// slogValue renders the view as a slog group.
func (v *logView) slogValue() slog.Value {
	attrs := make([]slog.Attr, 0, numLogViewFields)
	attrs = append(attrs, slog.String("message", v.message))
	if v.code != NoCode {
		attrs = append(attrs, slog.String("code", v.code.String()))
	}
	if v.transient {
		attrs = append(attrs, slog.Bool("transient", true))
	}
	if v.field != "" {
		attrs = append(attrs, slog.String("field", v.field))
	}
	if v.location != nil {
		attrs = append(attrs, slog.Any("location", v.location))
	}
	if len(v.stack) > 0 {
		attrs = append(attrs, slog.Any("stack", stackLines(v.stack)))
	}
	if v.public != nil {
		attrs = append(attrs, slog.Any("public_error", v.public))
	}
	if len(v.causes) > 0 {
		attrs = append(attrs, slog.Any("causes", v.causes))
	}
	return slog.GroupValue(attrs...)
}

// MarshalLogObject renders the view for zap.
func (v *logView) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("message", v.message)
	if v.code != NoCode {
		encoder.AddString("code", v.code.String())
	}
	if v.transient {
		encoder.AddBool("transient", true)
	}
	if v.field != "" {
		encoder.AddString("field", v.field)
	}
	if v.location != nil {
		if err := encoder.AddObject("location", v.location); err != nil {
			return err
		}
	}
	if len(v.stack) > 0 {
		lines := stackLines(v.stack)
		err := encoder.AddArray("stack", zapcore.ArrayMarshalerFunc(func(arr zapcore.ArrayEncoder) error {
			for _, line := range lines {
				arr.AppendString(line)
			}
			return nil
		}))
		if err != nil {
			return err
		}
	}
	if v.public != nil {
		if err := encoder.AddObject("public_error", v.public); err != nil {
			return err
		}
	}
	if len(v.causes) > 0 {
		return encoder.AddArray("causes", v.causes)
	}
	return nil
}

// MarshalZerologObject renders the view for zerolog.
func (v *logView) MarshalZerologObject(e *zerolog.Event) {
	e.Str("message", v.message)
	if v.code != NoCode {
		e.Str("code", v.code.String())
	}
	if v.transient {
		e.Bool("transient", true)
	}
	if v.field != "" {
		e.Str("field", v.field)
	}
	if v.location != nil {
		e.Object("location", v.location)
	}
	if len(v.stack) > 0 {
		e.Strs("stack", stackLines(v.stack))
	}
	if v.public != nil {
		e.Object("public_error", v.public)
	}
	if len(v.causes) > 0 {
		arr := zerolog.Arr()
		for _, cause := range v.causes {
			arr.Object(cause)
		}
		e.Array("causes", arr)
	}
}

// MarshalJSON renders the view with the keys and order of the three loggers,
// for the causes slog nests as JSON.
func (v *logView) MarshalJSON() ([]byte, error) {
	out := logViewJSON{
		Message:   v.message,
		Transient: v.transient,
		Field:     v.field,
		Causes:    v.causes,
	}
	if v.code != NoCode {
		out.Code = v.code.String()
	}
	if v.location != nil {
		out.Location = &stackFrameJSON{
			Path:     v.location.Path,
			Package:  v.location.Package,
			Function: v.location.Function,
			File:     v.location.File,
			Line:     v.location.Line,
		}
	}
	if len(v.stack) > 0 {
		out.Stack = stackLines(v.stack)
	}
	if v.public != nil {
		out.PublicError = &publicErrorJSON{
			Key:     v.public.Key,
			Message: v.public.Message,
			Details: v.public.Details(),
			Params:  v.public.Params,
		}
	}
	return json.Marshal(out)
}

// MarshalLogArray renders the causes for zap.
func (vs logViews) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	for _, v := range vs {
		if err := encoder.AppendObject(v); err != nil {
			return err
		}
	}
	return nil
}

// MarshalJSON renders the causes as a JSON array, for slog's JSON handler. The
// conversion to the plain slice keeps json.Marshal from calling this method
// again.
func (vs logViews) MarshalJSON() ([]byte, error) {
	return json.Marshal([]*logView(vs))
}

// MarshalText renders the causes as JSON for slog's text handler, which would
// otherwise print the pointers.
func (vs logViews) MarshalText() ([]byte, error) {
	return vs.MarshalJSON()
}

// numLogViewFields is the number of keys a view can log (message, code,
// transient, field, location, stack, public_error, causes).
const numLogViewFields = 8

// logViewJSON is the JSON shape of a logView, mirroring the keys the three
// loggers emit. StackFrame and PublicError marshal to JSON differently from
// how they log, so they are copied into tagged structs.
type logViewJSON struct {
	Message     string           `json:"message"`
	Code        string           `json:"code,omitempty"`
	Transient   bool             `json:"transient,omitempty"`
	Field       string           `json:"field,omitempty"`
	Location    *stackFrameJSON  `json:"location,omitempty"`
	Stack       []string         `json:"stack,omitempty"`
	PublicError *publicErrorJSON `json:"public_error,omitempty"`
	Causes      logViews         `json:"causes,omitempty"`
}

type stackFrameJSON struct {
	Path     string `json:"path"`
	Package  string `json:"package"`
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

type publicErrorJSON struct {
	Key     string `json:"key,omitempty"`
	Message string `json:"message"`
	Details string `json:"details"`
	Params  Params `json:"params,omitempty"`
}

// stackLines renders a stack trace one frame per string, "func (file:line)",
// the way Print prints an "at" line.
func stackLines(st StackTrace) []string {
	lines := make([]string, 0, len(st))
	for _, f := range st {
		lines = append(lines, fmt.Sprintf("%s (%s:%d)", qualifiedFunc(f), f.File, f.Line))
	}
	return lines
}
