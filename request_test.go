package viot

import (
	"testing"

	"github.com/issue9/assert/v2"
)

func Test_RequestConfig_MarshalAndUnmarshal(t *testing.T) {
	as := assert.New(t, true)
	var rc RequestConfig
	rc.SetBody("123")
	b, err := rc.Marshal()
	as.NotError(err)

	var rc1 RequestConfig
	err = rc1.Unmarshal(b)
	as.NotError(err).Equal(rc, rc1)
}

func Test_request_ProtoAtLeast(t *testing.T) {
	tests := []struct {
		major  int
		minor  int
		result bool
	}{
		{major: 1, minor: 1, result: true},
		{major: 1, minor: 0, result: true},
	}
	for index, test := range tests {
		req := &Request{
			ProtoMajor: test.major,
			ProtoMinor: test.minor,
		}
		if req.ProtoAtLeast(test.major, test.minor) != test.result {
			t.Fatalf("%d, error", index)
		}
	}
}

func Test_request_GetBasicAuth(t *testing.T) {
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
		req := &Request{
			Header: Header{"Authorization": test.basicAuth},
		}
		if user, pass, ok := req.GetBasicAuth(); (test.user != user || test.pass != pass) && ok != test.ok {
			t.Fatalf("%d，预测（user: %v, pass: %v, ok: %v），结果（user: %v, pass: %v, ok: %v）", index, test.user, test.pass, test.ok, user, pass, ok)
		}
	}
}

func Test_request_SetBasicAuth(t *testing.T) {
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
		req := &Request{
			Header: make(Header),
		}
		req.SetBasicAuth(test.user, test.pass)
		if user, pass, ok := req.GetBasicAuth(); (test.user != user || test.pass != pass) && ok != test.ok {
			t.Fatalf("%d，预测（user: %v, pass: %v, ok: %v），结果（user: %v, pass: %v, ok: %v）", index, test.user, test.pass, test.ok, user, pass, ok)
		}
	}
}

func Test_request_GetTokenAuth(t *testing.T) {
	tests := []struct {
		tokenAuth string
		token     string
	}{
		{tokenAuth: "token 123456790", token: "123456790"},
		{tokenAuth: "Token 123456790", token: ""},
		{tokenAuth: "Basic abc", token: ""},
	}
	for index, test := range tests {
		req := &Request{
			Header: Header{"Authorization": test.tokenAuth},
		}
		if token := req.GetTokenAuth(); test.token != token {
			t.Fatalf("%d，预测（token: %v），结果（token: %v）", index, test.token, token)
		}
	}
}

func Test_request_SetTokenAuth(t *testing.T) {
	as := assert.New(t, true)
	tests := []struct {
		token string
	}{
		{token: "123456790"},
		{token: "123456790"},
		{token: "abc"},
	}
	for _, test := range tests {
		req := &Request{
			Header: make(Header),
		}
		req.SetTokenAuth(test.token)
		token := req.GetTokenAuth()
		as.Equal(test.token, token)
	}
}

func Test_request_GetBody(t *testing.T) {
	req := &Request{
		Header: make(Header),
	}
	req.SetBody(123)
	var i interface{}
	req.GetBody(&i)
	if i != 123 {
		t.Fatal("error")
	}
}
