package collection

// Example is a named snapshot of a response, saved alongside the request
// that produced it.
type Example struct {
	Name       string
	StatusCode int
	Status     string
	Headers    []Header
	Body       string
}
