package viot

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net"
	"reflect"
	"testing"

	"github.com/456vv/x/tcptest"
	"github.com/issue9/assert/v2"
)

func Test_strSliceContains(t *testing.T) {
	tests := []struct {
		ss     []string
		t      string
		result bool
	}{
		{ss: []string{"a1", "a2", "a3"}, t: "a2", result: true},
		{ss: []string{"a1", "a2", "a3"}, t: "a4", result: false},
	}
	for index, test := range tests {
		if test.result != strSliceContains(test.ss, test.t) {
			t.Fatalf("%d，错误", index)
		}
	}
}

func Test_validMethod(t *testing.T) {
	tests := []struct {
		method string
		result bool
	}{
		{method: "HEAD", result: true},
		{method: "head", result: false},
		{method: "GET", result: true},
		{method: "POST", result: true},
	}
	for index, test := range tests {
		if test.result != validMethod(test.method) {
			t.Fatalf("%d，错误", index)
		}
	}
}

func Test_ParseIOTVersion(t *testing.T) {
	tests := []struct {
		version      string
		major, minor int
		ok           bool
	}{
		{version: "IOT/1.1", major: 1, minor: 1, ok: true},
		{version: "IOT/1.0", major: 1, minor: 0, ok: true},
		{version: "IOT/2.3", major: 2, minor: 3, ok: false},
		{version: "IOT/4.3", major: 2, minor: 3, ok: true},
		{version: "A/1.1", major: 0, minor: 0, ok: false},
		{version: "B/1.1", major: 0, minor: 0, ok: false},
	}

	for index, test := range tests {
		major, minor, ok := ParseIOTVersion(test.version)
		if ok != test.ok {
			if major != test.major || minor != test.minor {
				t.Fatalf("%d，预测（major %d, minor %d），错误（major %d, minor %d）", index, test.major, test.minor, major, minor)
			}
		}
	}
}

func Test_shouldClose(t *testing.T) {
	tests := []struct {
		major, minor int
		header       Header
		result       bool
	}{
		{major: 1, minor: 0, header: Header{"Connection": "keep-alive"}, result: false},
		{major: 1, minor: 0, header: Header{"Connection": "a"}, result: true}, // 1.0默认关闭
		{major: 1, minor: 0, header: Header{"Connection": "close"}, result: true},
		{major: 1, minor: 1, header: Header{"Connection": "close"}, result: true},
		{major: 1, minor: 1, header: Header{"Connection": "keep-alive"}, result: false},
		{major: 1, minor: 1, header: Header{"Connection": "a"}, result: false}, // 1.1需要明确指示关闭
	}
	for index, test := range tests {
		if test.result != shouldClose(test.major, test.minor, test.header) {
			t.Fatalf("%d， 预测不等于结果", index)
		}
	}
}

func Test_readRequest(t *testing.T) {
	tests := []struct {
		Nonce  string
		Proto  string
		Method string
		Path   string
		Host   string
		Header Header
		Body   interface{}
		err    bool
	}{
		{Nonce: "1", Proto: "IOT/1.1", Method: "GET", Path: "/a", Host: "a.com", Header: Header{"A": "a"}, Body: "aaaa", err: true},
		{Nonce: "2", Proto: "IOT/1.1", Method: "POST", Path: "/b", Host: "b.com", Header: Header{"B": "c"}, Body: "bbbb"},
		{Nonce: "3", Proto: "IOT/2.1", Method: "GET", Path: "/c", Host: "c.com", Header: Header{"C": "c"}, Body: "cccc", err: true},
	}
	for index, test := range tests {
		b, err := json.Marshal(test)
		if err != nil {
			t.Fatal(err)
		}
		req, err := readRequest(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		if test.Method != req.Method {
			t.Fatalf("%v, 预测 %v， 错误 %v", index, test.Method, req.Method)
		}
		if test.Nonce != req.nonce {
			t.Fatalf("%v, 预测 %v， 错误 %v", index, test.Nonce, req.nonce)
		}
		if test.Proto != req.Proto {
			t.Fatalf("%v, 预测 %v， 错误 %v", index, test.Proto, req.Proto)
		}
		if test.Path != req.RequestURI {
			t.Fatalf("%v, 预测 %v， 错误 %v", index, test.Path, req.RequestURI)
		}
		if test.Host != req.Host {
			t.Fatalf("%v, 预测 %v， 错误 %v", index, test.Host, req.Host)
		}
		if len(test.Header) != len(req.Header) {
			t.Fatalf("%v, 预测 %v， 错误 %v", index, test.Header, req.Header)
		}
		if test.Body != "" {
			var i interface{}
			err := req.GetBody(&i)
			if err != nil && !test.err {
				t.Fatal(index, err)
			}
			if err == nil && !reflect.DeepEqual(test.Body, i) {
				t.Fatalf("%v, 预测 %v， 错误 %v", index, test.Body, i)
			}
		}
		// t.Log(index, req)
	}
}

func Test_readResponse(t *testing.T) {
	tests := []struct {
		Nonce  string
		Status int
		Header Header
		Body   interface{}
		Close  bool
		err    bool
	}{
		{Nonce: "1", Status: 200, Header: Header{"A": "a", "Connection": "close"}, Body: "1", Close: true},
		{Nonce: "2", Status: 500, Header: Header{"A": "a", "Connection": "keep-alive"}, Body: "2", Close: false},
	}
	for index, test := range tests {
		jt, err := json.Marshal(test)
		if err != nil {
			t.Fatalf("%d, 编码错误 %v", index, err)
		}
		var br bytes.Reader
		br.Reset(jt)
		res, err := readResponse((io.Reader)(&br))
		if err != nil {
			t.Fatalf("%d, 响应错误 %v", index, err)
		}
		if test.Nonce != res.nonce {
			t.Fatalf("%v, 预测 %v， 错误 %v", index, test.Nonce, res.nonce)
		}
		if len(test.Header) != len(res.Header) {
			t.Fatalf("%v, 预测 %v， 错误 %v", index, test.Header, res.Header)
		}
		if test.Status != res.Status {
			t.Fatalf("%v, 预测 %v， 错误 %v", index, test.Status, res.Status)
		}
		if !reflect.DeepEqual(test.Body, res.Body) {
			t.Fatalf("%v, 预测 %v， 错误 %v", index, test.Body, res.Body)
		}
		if test.Close != res.Close {
			t.Fatalf("%v, 预测 %v， 错误 %v", index, test.Close, res.Close)
		}
	}
}

func Test_readResponse_1(t *testing.T) {
	as := assert.New(t, true)
	tcptest.C2L("127.0.0.1:0", func(l net.Listener) {
		srv := &Server{}
		srv.Handler = HandlerFunc(func(w ResponseWriter, r *Request) {
			Error(w, "error", 500)
		})
		err := srv.Serve(l)
		as.Error(err)
	}, func(c net.Conn) {
		defer c.Close()
		req, err := NewRequest("GET", "iot://lh/", nil)
		as.NotError(err)

		rc, err := req.RequestConfig("1234567890")
		as.NotError(err)

		b, err := rc.Marshal()
		as.NotError(err)

		n, err := c.Write(b)
		as.NotError(err).NotEqual(n, 0)

		res, err := readResponse(c)
		as.NotError(err).Equal(res.Status, 500)
	})
}

func Test_parseBasicAuth(t *testing.T) {
	tests := []struct {
		basicAuth string
		user      string
		pass      string
		ok        bool
	}{
		{basicAuth: "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==", user: "Aladdin", pass: "open sesame", ok: true},
		{basicAuth: "Basic YTpi", user: "a", pass: "b", ok: true},
		{basicAuth: "Basic abc", user: "a", pass: "b", ok: false},
	}
	for index, test := range tests {
		if user, pass, ok := parseBasicAuth(test.basicAuth); (test.user != user || test.pass != pass) && ok != test.ok {
			t.Fatalf("%d，预测（user: %v, pass: %v, ok: %v），结果（user: %v, pass: %v, ok: %v）", index, test.user, test.pass, test.ok, user, pass, ok)
		}
	}
}

func Test_Nonce(t *testing.T) {
	nonce, err := Nonce()
	if err != nil {
		t.Fatal(err)
	}
	if nonce == "" {
		t.Fatal("错误")
	}
}

func Test_newTextprotoReader(t *testing.T) {
	as := assert.New(t, true)
	rd := bytes.NewBufferString("{}\n\r")
	br := bufio.NewReader(rd)
	tr := newTextprotoReader(br)

	// 读取一行数据
	b, err := tr.ReadLineBytes()
	as.NotError(err).Equal(b, []byte("{}"))

	// br没有\n可以读取
	rb, err := rd.ReadByte()
	as.Error(err).Equal(rb, 0)
}
