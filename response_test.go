package viot

import(
	"testing"
	"reflect"
	"bytes"
	"encoding/json"
)

type respw struct{
	code 	int
	header	Header
	body	interface{}
}
func (T *respw) Status(code int){
	T.code = code
}
func (T *respw) Header() Header {
	if T.header == nil {
		T.header = make(Header)
	}
	return T.header 
}
func (T *respw) SetBody(i interface{}) error {
	T.body = i
	return nil
}


func Test_Response_WriteTo(t *testing.T){
	resp := &Response{
	    Status	:200,
		Header	:Header{"a":"b"},
		Body	:123,
	}
	w := &respw{}
	resp.WriteTo(w)
	
	if w.code != resp.Status {
		t.Fatalf("错误，预计 %d， 结果 %d", resp.Status, w.code)
	}
	if len(w.header) != len(resp.Header) {
		t.Fatalf("错误，预计 %d， 结果 %d", len(resp.Header), len(w.header))
	}
	if !reflect.DeepEqual(w.body, resp.Body) {
		t.Fatalf("错误，预计 %v， 结果 %v", resp.Body, w.body)
	}
}


func Test_Response_Write(t *testing.T){
	resp := &Response{
		nonce	:"123",
	    Status	:200,
		Header	:Header{"a":"b"},
		Body	:float64(123),
	}
	w := bytes.NewBuffer(nil)
	if err := resp.Write(w); err != nil {
		t.Fatal(err)
	}
	var ir ResponseIOT
	if err := json.NewDecoder(w).Decode(&ir); err != nil {
		t.Fatal(err)
	}
	
	if ir.Nonce != resp.nonce {
		t.Fatalf("错误，预计 %v， 结果 %v", resp.nonce, ir.Nonce)
	}
	if ir.Status != resp.Status {
		t.Fatalf("错误，预计 %v， 结果 %v", resp.Status, ir.Status)
	}
	if !reflect.DeepEqual(ir.Header, resp.Header ) {
		t.Fatalf("错误，预计 %v， 结果 %v", resp.Header, ir.Header)
	}
	if !reflect.DeepEqual(ir.Body, resp.Body ) {
		t.Fatalf("错误，预计 %v， 结果 %v", resp.Body, ir.Body)
	}
}