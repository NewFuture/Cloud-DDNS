package mode

// Oray (ph update) mode shares DynDNS request/response semantics.
type OrayMode struct {
	*DynMode
}

func NewOrayMode(debug func(format string, args ...interface{})) Mode {
	return &OrayMode{DynMode: NewDynMode(false, debug)}
}
