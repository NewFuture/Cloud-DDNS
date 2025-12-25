package mode

// DtDNS mode shares DynDNS request/response semantics.
type DtDNSMode struct {
	*DynMode
}

func NewDtDNSMode(debug func(format string, args ...interface{})) Mode {
	return &DtDNSMode{DynMode: NewDynMode(false, debug)}
}
