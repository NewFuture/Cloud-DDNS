package mode

// DynDNS-like HTTP mode (DynDNS/NIC update).
type DynDNSMode struct {
	*DynMode
}

func NewDynDNSMode(debug func(format string, args ...interface{})) Mode {
	return &DynDNSMode{DynMode: NewDynMode(false, debug)}
}
