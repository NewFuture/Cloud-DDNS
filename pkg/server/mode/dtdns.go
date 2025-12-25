package mode

import "net/http"

// DtDNSMode uses DynMode behavior but kept as an explicit mode for clarity.
type DtDNSMode struct {
	*DynMode
}

func NewDtDNSMode(debug func(format string, args ...interface{})) Mode {
	return &DtDNSMode{DynMode: NewDynMode(false, debug)}
}

func (m *DtDNSMode) Prepare(r *http.Request) (*Request, Outcome) {
	return m.DynMode.Prepare(r)
}

func (m *DtDNSMode) Process(req *Request) Outcome {
	return m.DynMode.Process(req)
}

func (m *DtDNSMode) Respond(w http.ResponseWriter, req *Request, outcome Outcome) {
	m.DynMode.Respond(w, req, outcome)
}
