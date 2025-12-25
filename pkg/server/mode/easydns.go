package mode

// EasyDNSMode uses DynMode behavior but kept as an explicit mode for clarity.
type EasyDNSMode struct {
	*DynMode
}

func NewEasyDNSMode(debug func(format string, args ...interface{})) Mode {
	return &EasyDNSMode{DynMode: NewDynMode(false, debug)}
}
