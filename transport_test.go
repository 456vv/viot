package viot
/*	
import(
	"testing"
	"fmt"
	"net"
	"time"
	"context"
)

func Test_Transport_RoundTripContext(t *testing.T){
	return
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	
	go func(){
		time.Sleep(time.Second)
		//laddr := l.Addr().String()
		//s := `{"nonce": 1, "proto":"IOT/1.1", "method":"POST", "header":{"Connection":"keep-alive"}, "home":"a.com", "path":"/a"}
		//	{"nonce": 2, "proto":"IOT/1.0", "method":"GET", "header":{"Connection":"keep-alive"}, "home":"b.com", "path":"/b"}`
		//b := []byte(s)
		//netConn, err := net.Dial("tcp", laddr)
		//n, err := netConn.Write(b)
		//if err != nil {
		//	t.Fatal(err)
		//}
		//if n != len(b) {
		//	t.Fatalf("预测 %v, 发送 %v", len(b), n)
		//}
		//time.Sleep(time.Second*2)
		//netConn.Close()
	}()

//	for {
		netConn, err := l.Accept()
		if err != nil {
			t.Fatal(err)
		}
		
		c := &conn{
			server: &Server{
				ReadTimeout: 1000,
				WriteTimeout: 1000,
			},
			rwc: netConn,
		}
		c.server.SetKeepAlivesEnabled(true)
		c.server.Handler=HandlerFunc(func(w ResponseWriter, r *Request){
			launch := w.(Launcher)
			tr, err := launch.Launch()
			if err != nil {
				return
			}
			fmt.Println(tr)
		})
		c.serve(context.Background())

//	}
	
}
*/