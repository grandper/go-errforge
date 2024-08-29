package errforge

import (
	"encoding/json"
	"net/http"
)

// WriteError writes err to w as a plain-text HTTP error response.
//
// The status is the one matching err's [Code] (via [GetCode] and [Code.HTTP]);
// an error without a code answers http.StatusInternalServerError (500). When
// err carries a user-facing [PublicError] (via [GetDetails]), its message is
// used as the response body; otherwise err.Error() is used. An Unauthenticated
// error also sets the WWW-Authenticate header to that message, as a 401
// requires.
//
// A code set with [WithCode] deep in the domain layer thus surfaces, unchanged,
// at the HTTP boundary, so a handler can turn any error into a correct response
// in one line. WriteError is a no-op when err is nil. [WriteErrorJSON] answers
// the same status with a JSON body instead.
func WriteError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	code, body := GetCode(err), errorBody(err)
	setAuthenticate(w, code, body)
	http.Error(w, body, code.HTTP())
}

// ErrorResponse is the body written by [WriteErrorJSON].
type ErrorResponse struct {
	// Code is the name of the error's code as spelled in google.rpc.Code, such
	// as "NOT_FOUND". An error without a code reports "UNKNOWN", as
	// [GRPCStatus] does.
	Code string `json:"code"`
	// Key is the translation key of the attached [PublicError], such as
	// "order.out_of_stock". Omitted when the error is not translatable.
	Key string `json:"key,omitempty"`
	// Message is the user-facing message: the [PublicError] message when one is
	// attached, otherwise the error's own message. When Key is set, it is the
	// fallback for a client that has no translation.
	Message string `json:"message"`
	// Details is the full explanation of the attached [PublicError]
	// ([PublicError.Details]): its reassurance, reason, resolution and way out,
	// rendered as one string. Omitted when no PublicError is attached or none
	// of those sentences is set.
	Details string `json:"details,omitempty"`
	// Params are the raw values of the attached [PublicError], for the client to
	// format the translated message with. Omitted when there are none.
	Params Params `json:"params,omitempty"`
	// Fields lists the failing inputs marked with [FieldError], one entry per
	// field in the order they were added. Omitted when there are none.
	Fields []FieldResponse `json:"fields,omitempty"`
}

// FieldResponse is one entry of [ErrorResponse.Fields]: a failing input and
// the message about it, chosen as for the response as a whole — the field's
// own [PublicError] when it carries one, otherwise its error message.
type FieldResponse struct {
	// Field is the name of the input, as given to [FieldError]; nested names
	// are joined by a dot ("address.city").
	Field string `json:"field"`
	// Key is the translation key of the field's [PublicError], if any.
	Key string `json:"key,omitempty"`
	// Message is the field's [PublicError] message when one is attached,
	// otherwise the field error's own message.
	Message string `json:"message"`
	// Params are the raw values of the field's [PublicError], if any.
	Params Params `json:"params,omitempty"`
}

// WriteErrorJSON writes err to w as a JSON HTTP error response.
//
// The status and the message are chosen exactly as [WriteError] does; the body
// is an [ErrorResponse], so a client reads the code by name rather than
// inferring it from the status:
//
//	HTTP/1.1 404 Not Found
//	Content-Type: application/json
//
//	{"code":"NOT_FOUND","message":"track 42 not found"}
//
// When the attached [PublicError] carries a Key and Params, the body carries
// them too, so a client that knows the user's locale translates the key and
// formats it with the raw values instead of showing the English message:
//
//	{"code":"FAILED_PRECONDITION","key":"order.out_of_stock",
//	 "message":"not enough items in stock","params":{"requested":8,"available":5}}
//
// When it carries the optional sentences (Reassurance, Reason, Resolution,
// WayOut), their rendering by [PublicError.Details] is carried under "details":
//
//	{"code":"UNAVAILABLE","message":"unable to connect to your account",
//	 "details":"Your changes were saved. Please try connecting again"}
//
// A validation error whose failures are marked with [FieldError] lists them
// under "fields", each with its own message or translation key, so a form
// highlights every bad input in one round-trip:
//
//	{"code":"INVALID_ARGUMENT","message":"invalid user",
//	 "fields":[{"field":"name","message":"cannot be empty"},
//	           {"field":"email","key":"validation.format","message":"has a bad format"}]}
//
// An Unauthenticated error also sets the WWW-Authenticate header. WriteErrorJSON
// is a no-op when err is nil.
func WriteErrorJSON(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	code, pe := GetCode(err), GetDetails(err)
	resp := ErrorResponse{Code: responseCode(code).String(), Message: err.Error()}
	if pe != nil {
		resp.Key, resp.Message, resp.Details, resp.Params = pe.Key, pe.Error(), pe.Details(), pe.Params
	}
	for _, v := range GetFieldErrors(err) {
		resp.Fields = append(resp.Fields, fieldResponse(v))
	}
	setAuthenticate(w, code, resp.Message)
	h := w.Header()
	h.Del("Content-Length")
	h.Set("Content-Type", "application/json")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code.HTTP())
	// Encoding fails only when Params holds a value JSON cannot represent,
	// which is a programming error at the site that built the PublicError; the
	// write error is the client's connection going away, which a handler cannot
	// act on. Neither can be reported once the status is written.
	_ = json.NewEncoder(w).Encode(resp)
}

// responseCode is the code reported in a response body: the error's own code,
// or Unknown when it has none, mirroring GRPCStatus.
func responseCode(code Code) Code {
	if code == NoCode {
		return Unknown
	}
	return code
}

// fieldResponse renders one field violation for the body, choosing the message
// as the response as a whole does: the field's PublicError when it has one,
// otherwise its error message.
func fieldResponse(v FieldViolation) FieldResponse {
	fr := FieldResponse{Field: v.Field, Message: v.Err.Error()}
	if pe := GetDetails(v.Err); pe != nil {
		fr.Key, fr.Message, fr.Params = pe.Key, pe.Error(), pe.Params
	}
	return fr
}

// setAuthenticate sets the WWW-Authenticate header a 401 response requires,
// carrying the message as the challenge, and does nothing for any other code.
func setAuthenticate(w http.ResponseWriter, code Code, body string) {
	if code == Unauthenticated {
		w.Header().Set("WWW-Authenticate", body)
	}
}

// errorBody returns the user-facing body for err: the PublicError message when
// one is attached, otherwise the error's own message.
func errorBody(err error) string {
	if pe := GetDetails(err); pe != nil {
		return pe.Error()
	}
	return err.Error()
}
