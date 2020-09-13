package viot
	
import(
	"testing"
	"net"
	"time"
	"encoding/json"
	"bytes"
	"reflect"
//	"fmt"
)

func Test_server_createDoneChan(t *testing.T){
	s := &Server{}
	c := s.createDoneChan()
	close(c)
}

func Test_server_getDoneChan(t *testing.T){
	s := &Server{}
	select{
		case <-s.getDoneChan():
		default: 
			if s.doneChan == nil {
				t.Fatal("错误")
			}
			c := s.createDoneChan()
			close(c)
	}

}

func Test_server_closeDoneChan(t *testing.T){
	s := &Server{}
	s.closeDoneChan()
	select{
	case <-s.doneChan:
	default:
		t.Fatal("错误")
	}
}

func Test_server_trackListener(t *testing.T){
	s := &Server{}
	s.trackListener(nil, true)
	if len(s.listeners) == 0 {
		t.Fatal("错误")
	}
	s.trackListener(nil, false)
	if len(s.listeners) != 0 {
		t.Fatal("错误")
	}
}

func Test_server_closeListeners(t *testing.T){
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{}
	s.trackListener(l, true)
	err = s.closeListeners()
	if err != nil {
		t.Fatal(err)
	}
	
}

func Test_server_trackConn(t *testing.T){
	s := &Server{}
	s.trackConn(nil, true)
	if len(s.activeConn) == 0 {
		t.Fatal("错误")
	}
	s.trackConn(nil, false)
	if len(s.activeConn) != 0 {
		t.Fatal("错误")
	}

}

func Test_server_closeConns(t *testing.T){
	
}

func Test_server_Serve(t *testing.T){
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
//	fmt.Println(l.Addr().String())
	s := &Server{}
	go func(t *testing.T){
		//退出服务器
		defer s.Close()
		
		time.Sleep(time.Second)
		tests := []struct{
			status int
			sand string
			nonce string
			body interface{}
		}{
			{status:400, nonce:"-1", sand:"{\"a\":\"a1\", \"nonce\":\"1\", \"proto\":\"1\"}\n"},
			{status:200, nonce:"1", body:"1", sand:"{\"nonce\":\"1\", \"proto\":\"IOT/1.1\", \"header\":{}, \"method\":\"POST\", \"path\":\"/a\", \"home\":\"a.com\", \"body\":\"1\"}\n"},
			{status:200, nonce:"2", body:float64(2), sand:"{\"nonce\":\"2\", \"proto\":\"IOT/1.1\", \"header\":{}, \"method\":\"POST\", \"path\":\"/b\", \"home\":\"b.com\", \"body\":2}\n"},
			{status:200, nonce:"3", body:map[string]interface{}{"a":"a1"}, sand:"{\"nonce\":\"3\", \"proto\":\"IOT/1.1\", \"header\":{}, \"method\":\"POST\", \"path\":\"/c\", \"home\":\"c.com\", \"body\":{\"a\":\"a1\"}}\n"},
			{status:200, nonce:"4", sand:"{\"nonce\":\"4\", \"proto\":\"IOT/1.1\", \"header\":{}, \"method\":\"POST\", \"path\":\"/d\", \"home\":\"d.com\", \"body\":[1]}\n"},
			{status:200, nonce:"5", sand:"{\"nonce\":\"5\", \"proto\":\"IOT/1.1\", \"header\":{}, \"method\":\"GET\", \"path\":\"/e\", \"home\":\"e.com\"}\n"},
			{status:200, nonce:"6", sand:"{\"nonce\":\"6\", \"proto\":\"IOT/1.1\", \"header\":{}, \"method\":\"GET\", \"path\":\"/f\", \"home\":\"f.com\"}\n"},
		}
		for index, test := range tests {
			c, err := net.Dial("tcp", l.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			
			n, err := c.Write([]byte(test.sand))
			if err != nil {
				t.Fatal(err)
			}
			if slen := len(test.sand); slen != n {
				t.Fatalf("预测长度 %v, 实际发送长度 %v", slen, n)
			}
			p := make([]byte,10240)
			n, err = c.Read(p)
			if err != nil {
				t.Fatal(err)
			}
			
			//关闭连接
			c.Close()
			
//			fmt.Printf("%s\n", p[:n])
			var ir ResponseConfig
			if err = json.NewDecoder(bytes.NewReader(p[:n])).Decode(&ir); err != nil {
				t.Fatalf("%d, 错误 %v", index, err)
			}
			if ir.Status != test.status {
				t.Fatalf("%d, 预测 %v，错误 %v", index, test.status, ir.Status)
			}
			if ir.Nonce != test.nonce {
				t.Fatalf("%d, 预测 %v，错误 %v", index, test.nonce, ir.Nonce)
			}
			if test.body != nil {
				if !reflect.DeepEqual(ir.Body, test.body) {
					t.Fatalf("%d, 预测 %v，错误 %v", index, test.body, ir.Body)
				}
			}
		}
	}(t)
	s.Handler=HandlerFunc(func(rw ResponseWriter, r *Request){
		if r.Method == "POST" {
			var inf interface{}
			err := r.GetBody(&inf)
			if err != nil {
				s.Close()
				t.Fatalf("Home: %v，错误：%v", r.Home, err)
			}
			rw.SetBody(&inf)
		}
	})
	s.Serve(l)
}

func Test_server_x(t *testing.T){
	
}

















