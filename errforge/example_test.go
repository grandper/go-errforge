package errforge_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/grandper/go-errforge/errforge"
)

func ExampleNew() {
	err := errforge.New("an error occurred")
	fmt.Println(err)
	// Output: an error occurred
}

func ExampleNew_sentinel() {
	errNotFound := errforge.New("not found")

	err := errforge.Wrap("loading user", errNotFound)
	fmt.Println(errforge.Is(err, errNotFound))
	// Output: true
}

func ExampleNewf() {
	cause := errforge.New("connection refused")
	err := errforge.Newf("dialing database: %w", cause)
	fmt.Println(err)
	fmt.Println(errforge.Is(err, cause))
	// Output:
	// dialing database: connection refused
	// true
}

func ExampleWrap() {
	cause := errforge.New("permission denied")
	err := errforge.Wrap("unable to open resource", cause)

	// Error() returns only the wrapping message, not the whole chain.
	fmt.Println(err.Error())
	// The cause is still reachable.
	fmt.Println(errforge.Is(err, cause))
	fmt.Println(errforge.Unwrap(err))
	// Output:
	// unable to open resource
	// true
	// permission denied
}

func ExampleWrap_multiple() {
	errEmptyName := errforge.New("name is empty")
	errBadEmail := errforge.New("email is invalid")

	err := errforge.Wrap("invalid user", errEmptyName, errBadEmail)
	fmt.Println(err.Error())
	fmt.Println(errforge.Is(err, errEmptyName), errforge.Is(err, errBadEmail))
	// Output:
	// invalid user
	// true true
}

func ExampleWrapf() {
	cause := errforge.New("no such file")
	err := errforge.Wrapf("reading %s", "config.yaml", cause)
	fmt.Println(err.Error())
	// Output: reading config.yaml
}

func ExampleLink() {
	errRequest := errforge.New("request failed")
	errTimeout := errforge.New("timeout")

	err := errforge.Link(errRequest, errTimeout)
	// Unlike Wrap, Link renders both messages.
	fmt.Println(err.Error())
	fmt.Println(errforge.Is(err, errRequest), errforge.Is(err, errTimeout))
	// Output:
	// request failed: timeout
	// true true
}

func ExampleAppend() {
	var err error
	err = errforge.Append(err, errforge.New("username cannot be empty"))
	err = errforge.Append(err, errforge.New("password is too short"))

	fmt.Println(err.Error())
	// Output: errors occurred: [username cannot be empty, password is too short]
}

func ExampleJoin() {
	err := errforge.Join(
		errforge.New("disk full"),
		nil, // nil errors are discarded
		errforge.New("quota exceeded"),
	)
	fmt.Println(err.Error())
	// Output: errors occurred: [disk full, quota exceeded]
}

func ExampleRootCause() {
	root := errforge.New("connection refused")
	wrapped := errforge.Newf("query failed: %w", root)
	top := errforge.Newf("request failed: %w", wrapped)

	fmt.Println(errforge.RootCause(top))
	// Output: connection refused
}

func ExamplePublicError() {
	err := &errforge.PublicError{
		Message:     "unable to connect to your account",
		Reassurance: "your changes were saved",
		Reason:      "we could not connect your account due to a technical issue on our end",
		Resolution:  "please try connecting again",
		WayOut:      "if the issue keeps happening, contact Customer Care",
	}

	fmt.Println(err.Error())
	fmt.Println(err.Details())
	// Output:
	// unable to connect to your account
	// Your changes were saved. We could not connect your account due to a technical issue on our end. Please try connecting again. If the issue keeps happening, contact Customer Care
}

func ExampleNewWithCode() {
	err := errforge.NewWithCode(errforge.NotFound, "track %s not found", "t-42")

	code := errforge.GetCode(err)
	fmt.Println(err)
	fmt.Println(code, code.HTTP(), code.GRPC())
	// Output:
	// track t-42 not found
	// NOT_FOUND 404 NotFound
}

func ExamplePropagate() {
	cause := errforge.NewWithCode(errforge.DeadlineExceeded, "timed out")
	err := errforge.Propagate(cause, "calling payment service")

	fmt.Println(err)
	fmt.Println(errforge.GetCode(err)) // the code is preserved
	// Output:
	// calling payment service
	// DEADLINE_EXCEEDED
}

func ExampleSprint() {
	inner := errforge.New("connection refused")
	err := errforge.Wrap("query failed", inner)

	fmt.Print(errforge.Sprint(err))
	// Output:
	// Error: query failed
	// Caused by: connection refused
}

func ExampleFromPanic() {
	err := errforge.FromPanic(func() {
		panic("something blew up")
	})
	fmt.Println(err)
	// Output: panic: something blew up
}

func ExampleRecover() {
	defer errforge.Recover(func(err error) {
		fmt.Println("recovered:", err)
	})
	panic("boom")
	// Output: recovered: panic: boom
}

func ExampleError_WithDetail() {
	errNotFound := errforge.New("not found")

	err := errNotFound.WithDetail("user with id 42")
	fmt.Println(err.Error())
	fmt.Println(errforge.Is(err, errNotFound))
	// Output:
	// not found: user with id 42
	// true
}

func ExampleError_WithDetailf() {
	errNotFound := errforge.New("not found")

	err := errNotFound.WithDetailf("user with id %d", 42)
	fmt.Println(err.Error())
	fmt.Println(errforge.Is(err, errNotFound))
	// Output:
	// not found: user with id 42
	// true
}

func ExampleError_WithErr() {
	errNotFound := errforge.New("not found")
	cause := errforge.New("connection refused")

	err := errNotFound.WithErr(cause)
	fmt.Println(err.Error())
	fmt.Println(errforge.Is(err, errNotFound))
	fmt.Println(errforge.Is(err, cause))
	// Output:
	// not found: connection refused
	// true
	// true
}

func ExampleToTransient() {
	err := errforge.ToTransient(errforge.New("service unavailable"))

	fmt.Println(err.Error())
	fmt.Println(errforge.IsTransient(err))
	// Output:
	// service unavailable
	// true
}

func ExampleWriteErrorJSON() {
	rec := httptest.NewRecorder()
	errforge.WriteErrorJSON(rec, errforge.NewWithCode(errforge.NotFound, "track 42 not found"))

	fmt.Println(rec.Code)
	fmt.Println(strings.TrimSpace(rec.Body.String()))
	// Output:
	// 404
	// {"code":"NOT_FOUND","message":"track 42 not found"}
}

func ExampleAttr() {
	errConnRefused := errforge.New("connection refused")
	codeOf := errforge.NewMapper(errforge.Map(errforge.Unavailable).To(errConnRefused))

	// Whatever sits on top — here a ToTransient marker over a coded Wrap — the
	// whole tree is logged: message, code, flags and causes.
	err := errforge.ToTransient(codeOf.AttachCodeTo(errforge.Wrap("calling payment service", errConnRefused)))

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{} // keep the output stable
			}
			return a
		},
	}))
	logger.Error("request failed", errforge.Attr(err))
	// Output:
	// {"level":"ERROR","msg":"request failed","error":{"message":"calling payment service","code":"UNAVAILABLE","transient":true,"causes":[{"message":"connection refused"}]}}
}

func ExampleMapper_AttachCodeTo() {
	errTrackNotFound := errforge.New("track not found")
	codeOf := errforge.NewMapper(
		errforge.Map(errforge.NotFound).To(errTrackNotFound),
	)

	err := codeOf.AttachCodeTo(errforge.Wrap("loading track 42", errTrackNotFound))

	fmt.Println(err.Error())
	fmt.Println(errforge.GetCode(err))
	fmt.Println(errforge.Is(err, errTrackNotFound))
	// Output:
	// loading track 42
	// NOT_FOUND
	// true
}

func ExampleGetCode_context() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := errforge.Propagate(ctx.Err(), "loading track")
	fmt.Println(errforge.GetCode(err))
	fmt.Println(errforge.GetCode(err).HTTP())
	// Output:
	// CANCELLED
	// 499
}

func ExampleIsAny() {
	errIDEmpty := errforge.New("id is empty")
	errNameTooLong := errforge.New("name is too long")

	err := errforge.Wrap("validating request", errNameTooLong)
	fmt.Println(errforge.IsAny(err, errIDEmpty, errNameTooLong))
	// Output: true
}
