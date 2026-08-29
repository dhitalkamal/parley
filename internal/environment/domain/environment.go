package environment

// Variable is one entry in an Environment or the globals scope.
type Variable struct {
	Key     string
	Value   string
	Enabled bool
	Secret  bool
}

// Environment is a named set of variables (dev/staging/prod/...). The
// globals scope is represented the same way, just without a meaningful Name.
type Environment struct {
	Name      string
	Variables []Variable
	// ExpectedVPN is the name of the VPN/tunnel interface this environment's
	// endpoints sit behind, if any - purely a hint the UI shows (connected vs
	// not). Empty means "any": no expectation, no warning. parley never
	// connects or disconnects it; see internal/netcheck.
	ExpectedVPN string
}
