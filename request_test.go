package viot
	
	
import(
	"testing"
//	"encoding/json"
	"fmt"
	"reflect"
)

func Test_RequestIOT_MarshalAndUnmarshal(t *testing.T) {
	var riot RequestConfig
	riot.SetBody("123")
	b, err := riot.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	
	var riot1 RequestConfig
	err = riot1.Unmarshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(riot, riot1) {
		fmt.Println("error")
	}
}


func Test_request_ProtoAtLeast(t *testing.T) {
	
	tests := []struct{
		major int
		minor int
		result bool
	}{
		{major:1, minor:1, result: true},
		{major:1, minor:0, result: true},
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
	tests := []struct{
		basicAuth string
		user string
		pass string
		ok bool
	}{
		{basicAuth:"Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==", user:"Aladdin", pass:"open sesame", ok: true},
		{basicAuth:"Basic YTpi", user:"a", pass:"b", ok: true},
		{basicAuth:"Basic abc", user:"a", pass:"b", ok: false},
	}
	for index, test := range tests {
		req := &Request{
			Header:Header{"Authorization":test.basicAuth},
		}
		if user, pass, ok := req.GetBasicAuth(); (test.user != user || test.pass != pass ) && ok != test.ok {
			t.Fatalf("%d，预测（user: %v, pass: %v, ok: %v），结果（user: %v, pass: %v, ok: %v）", index, test.user, test.pass, test.ok, user, pass, ok)
		}
	}

}

func Test_request_SetBasicAuth(t *testing.T) {
	tests := []struct{
		basicAuth string
		user string
		pass string
		ok bool
	}{
		{basicAuth:"Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==", user:"Aladdin", pass:"open sesame", ok: true},
		{basicAuth:"Basic YTpi", user:"a", pass:"b", ok: true},
		{basicAuth:"Basic abc", user:"a", pass:"b", ok: false},
	}
	for index, test := range tests {
		req := &Request{
			Header:make(Header),
		}
		req.SetBasicAuth(test.user, test.pass)
		if user, pass, ok := req.GetBasicAuth(); (test.user != user || test.pass != pass ) && ok != test.ok {
			t.Fatalf("%d，预测（user: %v, pass: %v, ok: %v），结果（user: %v, pass: %v, ok: %v）", index, test.user, test.pass, test.ok, user, pass, ok)
		}
	}

}


func Test_request_GetTokenAuth(t *testing.T) {
	tests := []struct{
		tokenAuth string
		token string
		ok bool
	}{
		{tokenAuth:"token 123456790", token:"123456790", ok: true},
		{tokenAuth:"Token 123456790", token:"123456790", ok: false},
		{tokenAuth:"Basic abc", token:"abc", ok: false},
	}
	for index, test := range tests {
		req := &Request{
			Header:Header{"Authorization":test.tokenAuth},
		}
		if token, ok := req.GetTokenAuth(); test.token != token  && ok != test.ok {
			t.Fatalf("%d，预测（token: %v, ok: %v），结果（token: %v,  ok: %v）", index, test.token, test.ok, token, ok)
		}
	}

}
func Test_request_SetTokenAuth(t *testing.T) {
	tests := []struct{
		tokenAuth string
		token string
		ok bool
	}{
		{tokenAuth:"token 123456790", token:"123456790", ok: true},
		{tokenAuth:"Token 123456790", token:"123456790", ok: false},
		{tokenAuth:"Basic abc", token:"abc", ok: false},
	}
	for index, test := range tests {
		req := &Request{
			Header:make(Header),
		}
		req.SetTokenAuth(test.token)
		if token, ok := req.GetTokenAuth(); test.token != token  && ok != test.ok {
			t.Fatalf("%d，预测（token: %v, ok: %v），结果（token: %v,  ok: %v）", index, test.token, test.ok, token, ok)
		}
	}

}


func Test_request_x(t *testing.T) {
	
}
























