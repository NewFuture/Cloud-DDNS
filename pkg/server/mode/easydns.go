package mode

import "net/http"

// EasyDNSMode uses DynMode behavior but kept as an explicit mode for clarity.
type EasyDNSMode struct {
	*DynMode
}

func NewEasyDNSMode(debug func(format string, args ...interface{})) Mode {
	return &EasyDNSMode{DynMode: NewDynMode(false, debug)}
}

func (m *EasyDNSMode) Prepare(r *http.Request) (*Request, Outcome) {
	return m.DynMode.Prepare(r)
}

func (m *EasyDNSMode) Process(req *Request) Outcome {
	return m.DynMode.Process(req)
}

func (m *EasyDNSMode) Respond(w http.ResponseWriter, req *Request, outcome Outcome) {
	m.DynMode.Respond(w, req, outcome)
}
