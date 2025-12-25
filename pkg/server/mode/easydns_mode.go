package mode

// EasyDNS mode shares DynDNS request/response semantics.
type EasyDNSMode struct {
	*DynMode
}

func NewEasyDNSMode(debug func(format string, args ...interface{})) Mode {
	return &EasyDNSMode{DynMode: NewDynMode(false, debug)}
}
