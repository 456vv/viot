package viot
	
import(
	"testing"
//	"context"
//	"net/http"
	"reflect"
	"bufio"
	"bytes"
	"encoding/json"
)

func Test_chunkWriter_ctype(t *testing.T) {
	var v1 int = 1
	var v2 *int = &v1
	var v3 interface{} = &v2
	tests := []struct{
		result 	string
		v		interface{}
		err		bool
	}{
		{result: "int", v: int(1)},
		{result: "object", v: float64(1), err: true},
		{result: "string", v: int(1), err: true},
		{result: "string", v: string("1")},
		{result: "string", v: (*string)(nil)},
		{result: "int", v: (interface{})(v2)},
		{result: "bool", v: (interface{})(true)},
		{result: "invalid", v: (interface{})(nil)},
		{result: "slice", v: []int{1}},
		{result: "array", v: [1]int{1}},
		{result: "map", v: map[int]int{1: 2}},
		{result: "invalid", v: nil},
		{result: "interface", v: (*interface{})(&v3)},
	}
	for index, test := range tests {
		cw := chunkWriter{}
		cw.ctype(test.v)
		if test.err != (cw.ct != test.result) {
			t.Fatalf("%d, 预测 %v, 错误 %v", index, test.result, cw.ct)
		}
	}
}


func Test_chunkWriter_generateResponse(t *testing.T) {
	tests := []struct{
		SetKeepAlivesEnabled bool
		SetKeepAlivesEnabledString	string
			
		nonce string
			
		ProtoMajor	int
		ProtoMinor	int
			
		closeAfterReply bool
		header	Header
	}{
		{SetKeepAlivesEnabled:true, SetKeepAlivesEnabledString: "keep-alive", nonce:"1", ProtoMajor:1, ProtoMinor:1, closeAfterReply:false, header:Header{"Connection":"keep-alive"}},
		{SetKeepAlivesEnabled:true, SetKeepAlivesEnabledString: "close", nonce:"5", ProtoMajor:1, ProtoMinor:1, closeAfterReply:true, header:Header{"Connection":"close"}},
		{SetKeepAlivesEnabled:true, SetKeepAlivesEnabledString: "close", nonce:"5", ProtoMajor:1, ProtoMinor:0, closeAfterReply:true, header:Header{"Connection":"keep-alive"}},
		{SetKeepAlivesEnabled:true, SetKeepAlivesEnabledString: "close", nonce:"5", ProtoMajor:1, ProtoMinor:0, closeAfterReply:true, header:Header{"Connection":"close"}},
		{SetKeepAlivesEnabled:false, SetKeepAlivesEnabledString: "close", nonce:"2", ProtoMajor:1, ProtoMinor:0, closeAfterReply:true, header:Header{"Connection":"close"}},
		{SetKeepAlivesEnabled:false, SetKeepAlivesEnabledString: "close", nonce:"2", ProtoMajor:1, ProtoMinor:0, closeAfterReply:true, header:Header{"Connection":"keep-alive"}},
		{SetKeepAlivesEnabled:false, SetKeepAlivesEnabledString: "close", nonce:"2", ProtoMajor:1, ProtoMinor:1, closeAfterReply:true, header:Header{"Connection":"close"}},
		{SetKeepAlivesEnabled:false, SetKeepAlivesEnabledString: "close", nonce:"2", ProtoMajor:1, ProtoMinor:1, closeAfterReply:true, header:Header{"Connection":"keep-alive"}},
	}
	for index, test := range tests {
		cw := &chunkWriter{
			res: &responseWrite{
				conn: &conn{
					server: &Server{},
				},
				req: &Request{
					nonce		: test.nonce,
					ProtoMajor	: test.ProtoMajor,
					ProtoMinor	: test.ProtoMinor,
					Header		: test.header,
				},
			},
		}
		cw.res.conn.server.SetKeepAlivesEnabled(test.SetKeepAlivesEnabled)
		cw.generateResponse()
		
		if test.SetKeepAlivesEnabled {
			conn := cw.data.Index("header","Connection").(string)
			if conn != test.SetKeepAlivesEnabledString {
				t.Fatalf("%v, 预测 %v, 错误 %v", index, test.SetKeepAlivesEnabledString, conn)
			}
		}
		if test.closeAfterReply != cw.res.closeAfterReply {
			t.Fatalf("%v, 预测 %v, 错误 %v", index, test.closeAfterReply, cw.res.closeAfterReply)
		}
		if !reflect.DeepEqual(test.nonce, cw.data.Get("nonce")) {
			t.Fatalf("%v, 预测 %v, 错误 %v", index, test.nonce, cw.data.Get("nonce"))		}
		//t.Log(cw.data)
	}
}

func Test_chunkWriter_Body(t *testing.T) {
	bytesBuffer := bytes.NewBuffer(nil)
	cw := &chunkWriter{
		res: &responseWrite{
			conn: &conn{
				server	: &Server{},
				bufw	: bufio.NewWriter(bytesBuffer),
			},
			req: &Request{
			},
		},
	}
	cw.SetBody([1]int{1})
	err := cw.done()
	if err != nil {
		t.Fatal(err)
	}
	cw.res.conn.bufw.Flush()
	
	var ir ResponseConfig
	err = json.NewDecoder(bytesBuffer).Decode(&ir)
	if err != nil {
		t.Fatal(err)
	}
	if ir.Nonce != "" {
		t.Fatalf("预测 \"\", 错误 %v", ir.Nonce)
	}
	if ir.Header == nil {
		t.Fatalf("错误：Header 是 nil")
	}
	if ir.Status != 200 {
		t.Fatalf("预测 %v, 错误 %v", 200, ir.Status)
	}
	if ir.Body == nil {
		t.Fatalf("错误：Body 是 nil")
	}
}

func Test_chunkWriter_done(t *testing.T) {
	bytesBuffer := bytes.NewBuffer(nil)
	cw := &chunkWriter{
		res: &responseWrite{
			conn: &conn{
				server	: &Server{},
				bufw	: bufio.NewWriter(bytesBuffer),
			},
			req: &Request{
			},
		},
	}
	err := cw.done()
	if err != nil {
		t.Fatal(err)
	}
	cw.res.conn.bufw.Flush()
	
	var ir ResponseConfig
	err = json.NewDecoder(bytesBuffer).Decode(&ir)
	if err != nil {
		t.Fatal(err)
	}
	if ir.Nonce != "" {
		t.Fatalf("预测 \"\", 错误 %v", ir.Nonce)
	}
	if ir.Header == nil {
		t.Fatalf("错误：Header 是 nil")
	}
	if ir.Status != 200 {
		t.Fatalf("预测 %v, 错误 %v", 200, ir.Status)
	}
	if ir.Body != nil {
		t.Fatalf("错误：Body 不为 nil")
	}
}



















































