package mode

import (
	"log"
	"net/http"
)

// EasyDNSMode uses DynMode behavior but kept as an explicit mode for clarity.
type EasyDNSMode struct {
	*DynMode
}

func NewEasyDNSMode(debug func(format string, args ...interface{})) Mode {
	return &EasyDNSMode{DynMode: NewDynMode(false, debug)}
}

// Respond writes EasyDNS-specific result codes.
func (m *EasyDNSMode) Respond(w http.ResponseWriter, req *Request, outcome Outcome) {
	var body string
	switch outcome {
	case OutcomeSuccess:
		body = "NOERROR"
	case OutcomeAuthFailure:
		body = "NOACCESS"
	case OutcomeInvalidDomain:
		body = "ILLEGAL INPUT"
	case OutcomeSystemError:
		body = "NOSERVICE"
	default:
		body = "NOSERVICE"
	}

	if _, err := w.Write([]byte(body)); err != nil {
		log.Printf("Failed to write EasyDNS response %q: %v", body, err)
	}
}
