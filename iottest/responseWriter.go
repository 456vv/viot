package iottest

import(
	"github.com/456vv/vconn"
	"github.com/456vv/viot/v2"
	"strings"
	"bufio"
	"net"
)

type ResponseWrite struct{
	Code int
	HeaderMap viot.Header
	Body interface{}
	
	//服务端的连接
	conn net.Conn
}
func (T *ResponseWrite) ClientConn(client func(net.Conn)) {
	C2S("127.0.0.1:0", func(c net.Conn){
		T.conn = c
	}, client)
}

func (T *ResponseWrite) Flush(){}

func (T *ResponseWrite) init(){
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
		Status: T.Code, 
		Header: T.HeaderMap, 
		Body: T.Body, 
		Close: false, 
		Request: nil, 
		RemoteAddr: "", 
	}
}

func (T *ResponseWrite) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	vc := vconn.NewConn(T.conn).(*vconn.Conn)
	return vc, bufio.NewReadWriter(bufio.NewReader(vc), bufio.NewWriter(T.conn)), nil
}

func (T *ResponseWrite) Write(buf []byte) (int, error) {
	T.init()
	return T.conn.Write(buf)
}

func (T *ResponseWrite) WriteString(str string) (int, error) {
	return T.Write([]byte(str))
}

func C2S(addr string, server func(c net.Conn), client func(c net.Conn)) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	
	go func(){
		laddr := l.Addr().String()
		netConn, err := net.Dial("tcp", laddr)
		if err != nil {
			 panic(err)
		}
		client(netConn)
		netConn.Close()
		l.Close()
	}()
	
	for {
		netConn, err := l.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				panic(err)
			}
			return
		}
		server(netConn)
	}
}












