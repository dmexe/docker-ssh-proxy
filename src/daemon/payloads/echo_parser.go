package payloads

// EchoParser is only for usage in test
type EchoParser struct {
	Payload Payload
	err     error
}

// Parse do nothing, just return EchoParser
func (p *EchoParser) Parse(_ string) (Payload, error) {
	return p.Payload, p.err
}
