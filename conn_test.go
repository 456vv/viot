package viot

import(
	"testing"
	"context"
	"net"
	"time"
	"bytes"
	"io"
	"strconv"
	"fmt"
	"encoding/json"
	"io/ioutil"
//	"reflect"
)

func Test_conn_readLineBytes(t *testing.T){
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	
	ss := []string{
		`{"nonce": "1", "proto":"IOT/1.0", "method":"POST", "header":{}, "home":"a.com", "path":"/a"}`,
		`{"nonce": "2", "proto":"IOT/1.1", "method":"GET", "header":{}, "home":"b.com", "path":"/b"}`,
	}
	var s string
	for _, vs := range ss {
		s = fmt.Sprintf("%s%s\n", s, vs)
	}

	go func(){
		time.Sleep(time.Second)
		laddr := l.Addr().String()
		b := []byte(s)
		netConn, err := net.Dial("tcp", laddr)
		if err != nil {
			t.Fatal(err)
		}
		n, err := netConn.Write(b)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(b) {
			t.Fatalf("预测 %v, 发送 %v", len(b), n)
		}
		io.Copy(ioutil.Discard, netConn)
		netConn.Close()
	}()

//	for {
		netConn, err := l.Accept()
		if err != nil {
			t.Fatal(err)
		}
		c := &conn{
			server: &Server{
				ReadTimeout: time.Second*3,
				WriteTimeout: 0,
			},
			rwc: netConn,
		}
		c.r 	= &connReader{conn:c}
		c.bufr 	= newBufioReader(c.r)
		c.bufw 	= newBufioWriterSize(connWriter{conn:c}, 4<<10)
		
		c.ctx, c.cancelCtx = context.WithCancel(context.Background())
		
		var index int
		for {
			lb, err := c.readLineBytes()
			if err != nil {
				operr, ok := err.(*net.OpError)
				if (ok && operr.Timeout()) || err == io.EOF {
					break
				}
				t.Fatalf("%d, error(%v)", index, err)
			}
			if !bytes.Equal(lb, []byte(ss[index])) {
				t.Fatalf("%d, 错误，读取数据不一样", index)
			}
			index++
		}
		c.Close()
		netConn.Close()
//	}
	time.Sleep(time.Second)
}


func Test_conn_readRequest(t *testing.T){

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	
	go func(){
		time.Sleep(time.Second)
		laddr := l.Addr().String()
		s := `{"nonce": "1", "proto":"IOT/1.0", "method":"POST", "header":{}, "home":"a.com", "path":"/a"}
			{"nonce": "2", "proto":"IOT/1.1", "method":"GET", "header":{}, "home":"b.com", "path":"/b"}`
		b := []byte(s)
		netConn, err := net.Dial("tcp", laddr)
		n, err := netConn.Write(b)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(b) {
			t.Fatalf("预测 %v, 发送 %v", len(b), n)
		}
		netConn.Close()
	}()
	
//	for {
		netConn, err := l.Accept()
		if err != nil {
			t.Fatal(err)
		}
		
		c := &conn{
			server: &Server{
				ReadTimeout: time.Second*1,
				WriteTimeout: time.Second*1,
			},
			rwc: netConn,
		}
		c.r 	= &connReader{conn:c}
		c.bufr 	= newBufioReader(c.r)
		c.bufw 	= newBufioWriterSize(connWriter{conn:c}, 4<<10)
		
		ctx, cancelCtx := context.WithCancel(context.Background())
		c.cancelCtx = cancelCtx
		
		lb, err := c.readLineBytes()
		if err != nil {
			t.Fatal(err)
		}
		br := bytes.NewReader(lb)
		req, err := c.readRequest(ctx, br)
		if err != nil {
			t.Fatal(err)
		}
		if req.nonce != "1" {
			t.Fatalf("预测为 1，结果为 %v", req.nonce)
		}
		
		ctx, cancelCtx = context.WithCancel(context.Background())
		c.cancelCtx = cancelCtx
		lb, err = c.readLineBytes()
		if err != nil {
			t.Fatal(err)
		}
		br = bytes.NewReader(lb)
		req, err = c.readRequest(ctx, br)
		if err != nil {
			t.Fatal(err)
		}
		if req.nonce != "2" {
			t.Fatalf("预测为 1，结果为 %v", req.nonce)
		}
		
		ctx, cancelCtx = context.WithCancel(context.Background())
		c.cancelCtx = cancelCtx
		lb, err = c.readLineBytes()
		if err == nil {
			t.Fatal("没有数据，应该是io.EOF")
		}
		br = bytes.NewReader(lb)
		req, err = c.readRequest(ctx, br)
		if err == nil {
			t.Fatal("没有数据，应该格式发生错误")
		}
		
		c.Close()
		netConn.Close()
//	}
	time.Sleep(time.Second*5)

}

func Test_conn_serve1(t *testing.T){

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	
	go func(){
		time.Sleep(time.Second)
		laddr := l.Addr().String()
		s := `{"nonce": "1", "proto":"IOT/1.1", "method":"POST", "header":{"Connection":"keep-alive"}, "home":"a.com", "path":"/a"}
			{"nonce": "2", "proto":"IOT/1.0", "method":"GET", "header":{"Connection":"keep-alive"}, "home":"b.com", "path":"/b"}`
		b := []byte(s)
		netConn, err := net.Dial("tcp", laddr)
		n, err := netConn.Write(b)
		if err != nil {
			t.Fatalf("写入错误：%v",err)
		}
		if n != len(b) {
			t.Fatalf("预测 %v, 发送 %v", len(b), n)
		}
		time.Sleep(time.Second)
		var riot ResponseConfig
		err = json.NewDecoder(netConn).Decode(&riot)
		if err != nil {
			t.Fatalf("读取错误：%v",err)
		}
		if riot.Status != 200 {
			t.Fatalf("返回状态是：%d", riot.Status)
		}
		netConn.Close()
	}()
	
//	for {
		netConn, err := l.Accept()
		if err != nil {
			t.Fatal(err)
		}
		
		c := &conn{
			server: &Server{
				ReadTimeout: 0,
				WriteTimeout: 0,
			},
			rwc: netConn,
		}
		c.server.SetKeepAlivesEnabled(true)
		var index int = 1
		c.server.Handler=HandlerFunc(func(w ResponseWriter, r *Request){
			if r.nonce != strconv.Itoa(index) {
				t.Fatalf("预测 %v，错误 %v", index, r.nonce)
			}
			index++
				
			w.Status(200)
			w.Header().Set("a","a1")
			
			res := w.(*responseWrite)
			if res.status != 200 {
				t.Fatal("错误 status 不是 200")
			}
			if res.header == nil {
				t.Fatal("错误 header 是 nil")
			}
			if a1 := res.header.Get("a"); a1 != "a1" {
				t.Fatalf("预测 a1，错误 %v", a1)
			}
			if res.handlerDone.isTrue() {
				t.Fatal("错误 handlerDone 是 true")
			}
		
		})
		c.serve(context.Background())
		
		
	//	t.Log(c)
//	}
	time.Sleep(time.Second*5)

}

func Test_conn_serve2(t *testing.T){

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	
	go func(t *testing.T){
		time.Sleep(time.Second)
		laddr := l.Addr().String()
		netConn, err := net.Dial("tcp", laddr)
		if err != nil {
			t.Fatal(err)
		}
		
		//->
		var riotb = requestConfigBody{
			RequestConfig: &RequestConfig{Nonce:"1",
				Proto:"IOT/1.1",
				Method:"POST",
				Header:Header{"Connection":"keep-alive"},
				Home:"a.com",
				Path:"/a"},
			Body:"123",
		}
		//请求
		err = json.NewEncoder(netConn).Encode(&riotb)
		if err != nil {
			t.Fatal(err)
		}
		//得到响应
		var riot ResponseConfig
		err = json.NewDecoder(netConn).Decode(&riot)
		if err != nil {
			t.Fatal(err)
		}
		if riot.Status != 200 {
			t.Fatal(riot)
		}
		//<-
		
		//->
		//得到请求
		var riotb1 requestConfigBody
		err = json.NewDecoder(netConn).Decode(&riotb1)
		if err != nil {
			t.Fatal(err)
		}
		if riotb.Home != riotb1.Home {
			t.Fatalf("发 %s, 收 %s", riotb.Home, riotb1.Home)
		}
		//回复
		riot.Nonce = riotb1.Nonce
		err = json.NewEncoder(netConn).Encode(&riot)
		if err != nil {
			t.Fatal(err)
		}
	//	netConn.Write([]byte("\n"))
		//<-
		
		//给5秒
		//time.Sleep(time.Second*5)
		netConn.Close()
	}(t)
	
//	for {
		netConn, err := l.Accept()
		if err != nil {
			t.Fatal(err)
		}
		
		c := &conn{
			server: &Server{
				ReadTimeout: 0,
				WriteTimeout: 0,
			},
			rwc: netConn,
		}
		c.server.SetKeepAlivesEnabled(true)
		
		var iotRes = make(map[string]RoundTripper) //map[ip]launcher
		var iotLaunch = func(ipAddr string, r *Request){
			//给一秒让服务返回信息到客户端
			time.Sleep(time.Second)
			res, err := iotRes[ipAddr].RoundTripContext(context.Background(), r)
			if err != nil {
				t.Fatal(err)
			}
			if res.Status != 200 {
				t.Fatal("错误")
			}
		}
		c.server.Handler=HandlerFunc(func(w ResponseWriter, r *Request){
			if _, ok := iotRes[r.RemoteAddr]; !ok {
				launch := w.(Launcher).Launch()
				iotRes[r.RemoteAddr] = launch
				go iotLaunch(r.RemoteAddr, r)
			}
		})
		c.serve(context.Background())
		
		
	//	t.Log(c)
//	}
	
	time.Sleep(time.Second*5)

}


func Test_conn_setState(t *testing.T){
	
	
	c := &conn{
		server: &Server{
			
		},
	}
	c.curState.Store(connStateInterface[StateNew])
	c.setState(StateNew)
	if _, ok := c.server.activeConn[c]; !ok {
		t.Fatal("连接无法记录")
	}
	cs, ok := c.curState.Load().(ConnState)
	if !ok {
		t.Fatal("无法记录连接状态")
	}
	if cs != StateNew {
		t.Fatal("记录连接状态不正确")
	}
	
	c.setState(StateClosed)
	if _, ok := c.server.activeConn[c]; ok {
		t.Fatal("连接无法清除")
	}
	
}

func Test_conn_closeWriteAndWait(t *testing.T){
	
}

func Test_conn_finalFlush(t *testing.T){
	c := &conn{}
	c.bufr = newBufioReader(bytes.NewReader(nil))
	c.bufw = newBufioWriterSize((io.Writer)(bytes.NewBuffer(nil)), 4<<10)
	
	c.finalFlush()
	if c.bufr != nil {
		t.Fatal("bufr 应该为nil")
	}
	if c.bufw != nil {
		t.Fatal("bufw 应该为nil")
	}
}


func Test_conn_launchLocked(t *testing.T){
	
}


















































