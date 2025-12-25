package mode

import (
	"log"
	"net/http"
)

// EasyDNSMode is intentionally kept as a distinct type even though it embeds
// DynMode. This preserves a clear separation between the EasyDNS protocol
// surface and the generic DynDNS behavior, and allows EasyDNS-specific
// behavior or response mapping to evolve independently of DynMode.
type EasyDNSMode struct {
	*DynMode
}

func NewEasyDNSMode(debug func(format string, args ...interface{})) Mode {
	return &EasyDNSMode{DynMode: NewDynMode(false, debug)}
}

// Respond writes EasyDNS dynamic DNS result codes as plain-text responses.
// It implements the EasyDNS API contract by mapping internal outcomes to
// EasyDNS result strings (each terminated with a newline):
//   - OutcomeSuccess       -> "NOERROR\n"
//   - OutcomeAuthFailure   -> "NOACCESS\n"
//   - OutcomeInvalidDomain -> "ILLEGAL INPUT\n"
//   - OutcomeSystemError   -> "NOSERVICE\n"
//   - any other outcome    -> "NOSERVICE\n"
//
// For protocol details, see the EasyDNS dynamic DNS API documentation.
// The "NOERROR\n" response is required by many easyDNS clients (e.g., inadyn).
func (m *EasyDNSMode) Respond(w http.ResponseWriter, req *Request, outcome Outcome) {
	var body string
	switch outcome {
	case OutcomeSuccess:
		body = "NOERROR\n"
	case OutcomeAuthFailure:
		body = "NOACCESS\n"
	case OutcomeInvalidDomain:
		body = "ILLEGAL INPUT\n"
	case OutcomeSystemError:
		body = "NOSERVICE\n"
	default:
		body = "NOSERVICE\n"
	}

	if _, err := w.Write([]byte(body)); err != nil {
		log.Printf("Failed to write EasyDNS response %q: %v", body, err)
	}
}
