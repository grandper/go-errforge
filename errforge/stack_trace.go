package errforge

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// maxStackDepth is the maximum number of frames captured for a stack trace.
const maxStackDepth = 64

// StackTrace is a stack of frames, ordered from the innermost frame (where the
// trace was captured) to the outermost.
type StackTrace []*StackFrame

// StackTracer is implemented by errors that carry a stack trace.
type StackTracer interface {
	error
	StackTrace() StackTrace
}

// WithStack annotates err with a stack trace captured at the point WithStack is
// called. The message and identity of err are preserved: the result reports the
// same Error string, unwraps to err, and matches err under errors.Is/As.
//
// If err is nil, WithStack returns nil.
func WithStack(err error) error {
	if err == nil {
		return nil
	}
	return &stackError{err: err, stack: captureStack(1)}
}

// GetStackTrace returns the first stack trace found while walking err's tree, or
// nil if no error in the tree carries one. The tree is traversed depth-first
// through both Unwrap() error and Unwrap() []error, so the outermost trace
// wins.
func GetStackTrace(err error) StackTrace {
	for node := range walk(err) {
		if st, ok := node.(StackTracer); ok { //nolint:errorlint // walk visits every node; the first carrier wins
			return st.StackTrace()
		}
	}
	return nil
}

// stackError wraps an error with a stack trace.
type stackError struct {
	err   error
	stack StackTrace
}

// Error implements the error interface.
func (w *stackError) Error() string { return w.err.Error() }

// Unwrap returns the wrapped error.
func (w *stackError) Unwrap() error { return w.err }

// StackTrace implements the StackTracer interface.
func (w *stackError) StackTrace() StackTrace { return w.stack }

// captureStack captures the call stack. skip is the number of frames to skip
// above the caller of captureStack (skip == 0 starts at that caller).
func captureStack(skip int) StackTrace {
	pcs := make([]uintptr, maxStackDepth)
	// skipInternalFrames skips runtime.Callers and captureStack itself.
	const skipInternalFrames = 2
	n := runtime.Callers(skip+skipInternalFrames, pcs)
	if n == 0 {
		return nil
	}
	frames := runtime.CallersFrames(pcs[:n])
	trace := make(StackTrace, 0, n)
	for {
		frame, more := frames.Next()
		trace = append(trace, frameFromParts(frame.Function, frame.File, frame.Line))
		if !more {
			break
		}
	}
	return trace
}

// frameFromParts builds a StackFrame from a fully-qualified function name (such
// as "github.com/grandper/repo/pkg.(*T).Method"), a file path and a line.
func frameFromParts(funcName, file string, line int) *StackFrame {
	dir, pkgFunc := filepath.Split(funcName)
	pkg, fn := splitPackageFunc(pkgFunc)
	currentDir := filepath.Base(filepath.Dir(file))
	return &StackFrame{
		Path:     dir + currentDir,
		Package:  pkg,
		Function: fn,
		File:     filepath.Base(file),
		Line:     line,
	}
}

// splitPackageFunc splits the trailing "pkg.func" portion of a function name
// into its package and function. The function retains any receiver, so
// "pkg.(*T).Method" yields ("pkg", "(*T).Method").
func splitPackageFunc(s string) (string, string) {
	if i := strings.IndexByte(s, '.'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return "", s
}

// String renders the stack trace with one frame per line, in the form
// "\tat <function> (<file>:<line>)".
func (st StackTrace) String() string {
	s := &strings.Builder{}
	st.writeTo(s)
	return s.String()
}

// Format implements the fmt.Formatter interface.
//
//	%s, %v   one frame per line as "\tat <function> (<file>:<line>)"
//	%+v      same, but each function is qualified with its package
func (st StackTrace) Format(s fmt.State, verb rune) {
	switch verb {
	case 's', 'v':
		st.writeFormatted(s, s.Flag('+'))
	default:
		st.writeFormatted(s, false)
	}
}

func (st StackTrace) writeTo(w io.Writer) {
	st.writeFormatted(w, false)
}

func (st StackTrace) writeFormatted(w io.Writer, withPackage bool) {
	for _, f := range st {
		fn := f.Function
		if withPackage && f.Package != "" {
			fn = f.Package + "." + f.Function
		}
		_, _ = io.WriteString(w, "\tat ")
		_, _ = io.WriteString(w, fn)
		_, _ = io.WriteString(w, " (")
		_, _ = io.WriteString(w, f.File)
		_, _ = io.WriteString(w, ":")
		_, _ = io.WriteString(w, strconv.Itoa(f.Line))
		_, _ = io.WriteString(w, ")\n")
	}
}
