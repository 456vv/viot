package iottest

import (
	"bufio"
	"context"
	"net"

	"github.com/456vv/vconn"
	"github.com/456vv/viot/v2"
	"github.com/456vv/x/tcptest"
)

type ResponseWrite struct {
	Code      int
	HeaderMap viot.Header
	Body      interface{}

	//服务端的连接
	conn net.Conn

	rr func(*viot.Request) (*viot.Response, error)
}

func (T *ResponseWrite) HookHijack(client func(net.Conn)) {
	tcptest.C2S("127.0.0.1:0", func(c net.Conn) {
		T.conn = c
	}, client)
}

func (T *ResponseWrite) HookRoundTrip(rr func(*viot.Request) (*viot.Response, error)) {
	T.rr = rr
}

func (T *ResponseWrite) Flush() {}

func (T *ResponseWrite) init() {
	if T.HeaderMap == nil {
		T.HeaderMap = make(viot.Header)
	}
}

func (T *ResponseWrite) Header() viot.Header {
	T.init()
	return T.HeaderMap
}

func (T *ResponseWrite) Status(code int) {
	T.Code = code
}

func (T *ResponseWrite) SetBody(body interface{}) error {
	T.Body = body
	return nil
}

func (T *ResponseWrite) Result() *viot.Response {
	return &viot.Response{
		Status:  T.Code,
		Header:  T.HeaderMap,
		Body:    T.Body,
		Close:   false,
		Request: nil,
	}
}

func (T *ResponseWrite) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	vc := vconn.NewConn(T.conn)
	return vc, bufio.NewReadWriter(bufio.NewReader(vc), bufio.NewWriter(T.conn)), nil
}

func (T *ResponseWrite) Write(buf []byte) (int, error) {
	T.init()
	return T.conn.Write(buf)
}

func (T *ResponseWrite) WriteString(str string) (int, error) {
	return T.Write([]byte(str))
}

func (T *ResponseWrite) RoundTrip(req *viot.Request) (resp *viot.Response, err error) {
	return T.RoundTripContext(context.Background(), req)
}

func (T *ResponseWrite) RoundTripContext(ctx context.Context, req *viot.Request) (resp *viot.Response, err error) {
	if T.rr == nil {
		panic("Need set HookRoundTrip")
	}
	return T.rr(req)
}

func (T *ResponseWrite) Launch() viot.RoundTripper {
	return T
}
