package payloads

// EchoParser is only for test usage
type EchoParser struct {
	Payload Payload
	err     error
}

// Parse nothing, just return EchoParser
func (p *EchoParser) Parse(_ string) (Payload, error) {
	return p.Payload, p.err
}
