package viot

import (
	"bytes"
)

type Parser interface {
	Request([]byte) (*Request, error)     // binary转请求
	Response([]byte) (*Response, error)   // binary转响应
	Unrequest(*Request) ([]byte, error)   // 请求转binary
	Unresponse(*Response) ([]byte, error) // 响应转binary
}

type defaultParse struct{}

func (T *defaultParse) Request(b []byte) (*Request, error) {
	if !isRequest(b) {
		return nil, ErrReqUnavailable
	}
	return readRequest(bytes.NewReader(b))
}

func (T *defaultParse) Response(b []byte) (*Response, error) {
	if !isResponse(b) {
		return nil, ErrRespUnavailable
	}
	return readResponse(bytes.NewReader(b))
}

func (T *defaultParse) Unrequest(p *Request) ([]byte, error) {
	return p.Marshal()
}

func (T *defaultParse) Unresponse(p *Response) ([]byte, error) {
	return p.Marshal()
}
