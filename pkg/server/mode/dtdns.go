package mode

// DtDNSMode uses DynMode behavior but kept as an explicit mode for clarity.
type DtDNSMode struct {
	*DynMode
}

func NewDtDNSMode(debug func(format string, args ...interface{})) Mode {
	return &DtDNSMode{DynMode: NewDynMode(false, debug)}
}
