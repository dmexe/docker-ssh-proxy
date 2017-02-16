package payload

type TestingParser struct {
	request *Request
	err     error
}

func NewTestingParser(request *Request, err error) *TestingParser {
	return &TestingParser{
		request: request,
		err:     err,
	}
}

func (p *TestingParser) Parse(unused string) (*Request, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.request, nil
}
