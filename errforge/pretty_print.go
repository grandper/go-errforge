package errforge

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Print writes a detailed, human-readable representation of err to os.Stderr.
// The output includes the message of every error in the tree, any stack traces
// they carry, and the members of nested multierrors. It is a no-op when err is
// nil.
//
// Unlike the Error method, which returns a single line, Print is meant for
// diagnostics and can span multiple lines.
func Print(err error) {
	Fprint(os.Stderr, err)
}

// Sprint returns the detailed representation of err produced by Print as a
// string. It returns the empty string when err is nil.
func Sprint(err error) string {
	s := &strings.Builder{}
	Fprint(s, err)
	return s.String()
}

// Fprint writes the detailed representation of err (see Print) to w.
func Fprint(w io.Writer, err error) {
	fprint(w, err, 0, "Error: ")
}

func fprint(w io.Writer, err error, depth int, label string) {
	if err == nil {
		return
	}
	indent := strings.Repeat("\t", depth)
	fmt.Fprintf(w, "%s%s%s\n", indent, label, err.Error())

	// Transparent decorators — WithStack, ToTransient, a coded
	// decorator, a Propagate that keeps the wording — repeat the message of the
	// node they wrap. The whole run of nodes sharing this message is printed as
	// one entry: the message once, then the first stack trace found in the run.
	// The walk then continues from the last node of the run.
	trace, last := collapse(err)
	for _, f := range trace {
		fmt.Fprintf(w, "%s\tat %s (%s:%d)\n", indent, qualifiedFunc(f), f.File, f.Line)
	}

	if members := UnwrapErrors(last); len(members) > 0 {
		for i, member := range members {
			fprint(w, member, depth+1, fmt.Sprintf("%d. Error: ", i+1))
		}
		return
	}
	if cause := Unwrap(last); cause != nil {
		fprint(w, cause, depth, "Caused by: ")
	}
}

// collapse follows the single-cause chain from err while the message stays the
// same and returns the first stack trace carried by any node in that run (nil
// when none does) together with the last node of the run.
func collapse(err error) (StackTrace, error) {
	var trace StackTrace
	for {
		st, ok := err.(StackTracer) //nolint:errorlint // the loop steps through the run node by node
		if ok && trace == nil {
			trace = st.StackTrace()
		}
		cause := transparentCause(err)
		if cause == nil {
			return trace, err
		}
		err = cause
	}
}

func qualifiedFunc(f *StackFrame) string {
	if f.Package != "" {
		return f.Package + "." + f.Function
	}
	return f.Function
}
