package errforge

import "google.golang.org/grpc/status"

// GRPCStatus converts err into a gRPC status error.
//
// The gRPC code is the one matching err's [Code] (via [GetCode] and
// [Code.GRPC]); an error without a code answers codes.Unknown, as
// status.Convert does for an error carrying no status. When err carries a
// user-facing [PublicError] (via [GetDetails]), its message is used; otherwise
// err.Error() is used.
//
// The returned error carries a *status.Status, so a client's status.FromError
// observes the mapped code. GRPCStatus returns nil when err is nil, so it can be
// returned inline from a handler. It mirrors [WriteError] over the gRPC
// transport: the same coded error serves both.
func GRPCStatus(err error) error {
	if err == nil {
		return nil
	}
	return status.Error(GetCode(err).GRPC(), errorBody(err))
}
