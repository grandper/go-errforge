package errforge

import (
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"

	"github.com/rs/zerolog"
	"go.uber.org/zap/zapcore"
)

// StackFrame represents a single frame in the stack where an error or panic
// was created.
type StackFrame struct {
	// Path is the import path of the directory holding the source file.
	Path string
	// Package is the name of the package containing the frame.
	Package string
	// Function is the name of the function (or method) of the frame.
	Function string
	// File is the base name of the source file.
	File string
	// Line is the line number within the source file.
	Line int
}

// String returns a description of the stack frame in the form
// "path/file.go:line". It implements the fmt.Stringer interface.
func (sf *StackFrame) String() string {
	return fmt.Sprintf("%s/%s:%d", sf.Path, sf.File, sf.Line)
}

// MarshalText formats a stacktrace Frame as a text string. The output is the
// same as that of fmt.Sprintf("%+v", f), but without newlines or tabs.
func (sf *StackFrame) MarshalText() ([]byte, error) {
	name := sf.Function
	if name == "unknown" {
		return []byte(name), nil
	}
	return []byte(fmt.Sprintf("%s %s:%d", name, sf.File, sf.Line)), nil
}

// LogValue implements the slog.LogValuer interface to provide
// a clean log of the location.
func (sf *StackFrame) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("path", sf.Path),
		slog.String("package", sf.Package),
		slog.String("function", sf.Function),
		slog.String("file", sf.File),
		slog.Int("line", sf.Line),
	)
}

// MarshalLogObject implements the zapcore.ObjectMarshaler interface to provide
// a clean log of the error when zap is used.
func (sf *StackFrame) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("path", sf.Path)
	encoder.AddString("package", sf.Package)
	encoder.AddString("function", sf.Function)
	encoder.AddString("file", sf.File)
	encoder.AddInt("line", sf.Line)
	return nil
}

// MarshalZerologObject implements the zerolog.LogObjectMarshaler interface to provide
// a clean log of the error when zerolog is used.
func (sf *StackFrame) MarshalZerologObject(e *zerolog.Event) {
	e.Str("path", sf.Path).
		Str("package", sf.Package).
		Str("function", sf.Function).
		Str("file", sf.File).
		Int("line", sf.Line)
}

// CreateStackFrame returns the stack frame where the function is called.
func CreateStackFrame() *StackFrame {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		return nil
	}
	// runtime.FuncForPC(pc).Name() returns strings such as:
	// - "github.com/grandper/repository/package.FuncName"
	// - "github.com/grandper/repository/package.Receiver.MethodName"
	// - "github.com/grandper/repository/package.(*PtrReceiver).MethodName"
	return frameFromParts(runtime.FuncForPC(pc).Name(), file, line)
}

// Format implements the fmt.Formatter interface.
//
// Available verbs:
//
//	%h    path
//	%p    package
//	%f    file name
//	%n    function name
//	%d	  line number
//	%v    equivalent to %s:%d
//	%s	  function, file and line number in a single line
//
// Format accepts flags that alter the printing of some verbs, as follows:
//
//	%+n   function name, the plus flag adds a package name
//	%+s   function name and path of source file relative to the compile time
//	      GOPATH separated by \n\t (<funcname>\n\t<path>)
//	%+v   equivalent to %+s:%d
func (sf *StackFrame) Format(s fmt.State, verb rune) {
	switch verb {
	case 'h':
		_, _ = io.WriteString(s, sf.Path)
	case 'p':
		_, _ = io.WriteString(s, sf.Package)
	case 'f':
		_, _ = io.WriteString(s, sf.File)
	case 'n':
		switch {
		case s.Flag('+') || s.Flag('#'):
			_, _ = io.WriteString(s, sf.Package)
			_, _ = io.WriteString(s, ".")
			_, _ = io.WriteString(s, sf.Function)
		default:
			_, _ = io.WriteString(s, sf.Function)
		}
	case 'd':
		_, _ = io.WriteString(s, strconv.Itoa(sf.Line))
	case 's':
		switch {
		case s.Flag('+') || s.Flag('#'):
			_, _ = io.WriteString(s, sf.Function)
			_, _ = io.WriteString(s, "\n\t")
			_, _ = io.WriteString(s, sf.File)
		default:
			_, _ = io.WriteString(s, sf.File)
		}
	case 'v':
		switch {
		case s.Flag('+') || s.Flag('#'):
			sf.Format(s, 's')
			_, _ = io.WriteString(s, ":")
			sf.Format(s, 'd')
		default:
			sf.Format(s, 's')
		}

	default:
		panic("not implemented")
	}
}
