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


func Test_dataWriter_generateResponse(t *testing.T) {
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
		cw := &dataWriter{
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
			t.Fatalf("%v, 预测 %v, 错误 %v", index, test.nonce, cw.data.Get("nonce"))
		}
		//t.Log(cw.data)
	}
}

func Test_dataWriter_Body(t *testing.T) {
	bytesBuffer := bytes.NewBuffer(nil)
	cw := &dataWriter{
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

func Test_dataWriter_done(t *testing.T) {
	bytesBuffer := bytes.NewBuffer(nil)
	cw := &dataWriter{
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



















































