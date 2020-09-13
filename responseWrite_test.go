package viot
	
import(
	"testing"
	"bytes"
	"encoding/json"
	"bufio"
)


func Test_response_bodyAllowed(t *testing.T) {
	
	tests := []struct{
		status	int
		result	bool
	}{
		{status:100, result: false},//>=100
		{status:199, result: false},//<=199
		{status:200, result: true},
		{status:204, result: false},
		{status:304, result: false},
		{status:500, result: true},
	}
	for index, test := range tests {
		res := &responseWrite{
			status: test.status,
		}
		if res.bodyAllowed() != test.result {
			t.Fatalf("%d, 状态 %v, 预测 %v, 错误 %v", index, test.status, test.result, res.bodyAllowed(),)
		}
	
	}
}


func Test_response_Header(t *testing.T) {
	res := &responseWrite{
	}
	if res.header != nil {
		t.Fatal("错误：.header 不是 nil")
	}
	res.Header()
	if res.header == nil {
		t.Fatal("错误：.header 是 nil")
	}

}


func Test_response_Status(t *testing.T) {
	res := &responseWrite{
		conn: &conn{
			server: &Server{
				
			},
		},
	}
	if res.status != 0 {
		t.Fatal("错误：.status 不是 0")
	}
	if res.wroteStatus {
		t.Fatal("错误：.wroteStatus 是 true")
	}
	res.Status(200)
	if !res.wroteStatus {
		t.Fatal("错误：.wroteStatus 是 false")
	}
	if res.status != 200 {
		t.Fatal("错误：.status 不是 200")
	}
	

}

func Test_response_SetBody(t *testing.T) {

	bytesBuffer := bytes.NewBuffer(nil)
	res := &responseWrite{
		conn: &conn{
			server	: &Server{},
			bufw	: bufio.NewWriter(bytesBuffer),
		},
		req: &Request{
		},
	}
	res.cw = chunkWriter{res:res}
	res.SetBody([1]int{1})
	err := res.done()
	if err != nil {
		t.Fatal(err)
	}

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

func Test_response_done(t *testing.T) {
	bytesBuffer := bytes.NewBuffer(nil)
	res := &responseWrite{
		conn: &conn{
			server	: &Server{},
			bufw	: bufio.NewWriter(bytesBuffer),
		},
		req: &Request{
		},
	}
	res.cw = chunkWriter{res:res}
	
	if res.handlerDone.isTrue() {
		t.Fatalf("错误：handlerDone 是 true")
	}
	
	res.done()
	
	if !res.handlerDone.isTrue() {
		t.Fatalf("错误：handlerDone 是 false")
	}
	var ir ResponseConfig
	err := json.NewDecoder(bytesBuffer).Decode(&ir)
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
		t.Fatalf("错误：Body 不是 nil")
	}

}
















