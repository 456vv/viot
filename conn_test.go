package viot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/456vv/vconn"
	"github.com/456vv/x/tcptest"
	"github.com/issue9/assert/v2"
)

func Test_conn_readLineBytes(t *testing.T) {
	ss := []string{
		`{"nonce": "1", "proto":"IOT/1.0", "method":"POST", "header":{}, "host":"a.com", "path":"/a"}`,
		`{"nonce": "2", "proto":"IOT/1.1", "method":"GET", "header":{}, "host":"b.com", "path":"/b"}`,
	}
	str := strings.Join(ss, "\n")

	tcptest.C2S("127.0.0.1:0", func(netConn net.Conn) {
		c := &conn{
			server: &Server{
				ReadTimeout:  time.Second,
				WriteTimeout: 0,
			},
			rwc: netConn,
		}

		c.vc = vconn.NewConn(c.rwc).(*vconn.Conn)
		c.vc.DisableBackgroundRead(true)
		c.bufr = newBufioReader(c.vc)
		c.ctx, c.cancelCtx = context.WithCancel(context.Background())
		defer c.cancelCtx()

		var index int
		for {
			lb, err := c.readLineBytes()
			if err != nil {
				if isCommonNetReadError(err) {
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
	}, func(netConn net.Conn) {
		b := []byte(str)
		n, err := netConn.Write(b)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(b) {
			t.Fatalf("预测 %v, 发送 %v", len(b), n)
		}
		io.Copy(ioutil.Discard, netConn)
		netConn.Close()
	})
}

func Test_conn_readRequest(t *testing.T) {
	ss := []string{
		`{"nonce": "1", "proto":"IOT/1.0", "method":"POST", "header":{}, "host":"a.com", "path":"/a"}`,
		`{"nonce": "2", "proto":"IOT/1.1", "method":"GET", "header":{}, "host":"b.com", "path":"/b"}`,
	}
	str := strings.Join(ss, "\n")

	tcptest.C2S("127.0.0.1:0", func(netConn net.Conn) {
		c := &conn{
			server: &Server{
				ReadTimeout:  time.Second * 3,
				WriteTimeout: 0,
			},
			rwc: netConn,
		}

		c.vc = vconn.NewConn(c.rwc).(*vconn.Conn)
		c.vc.DisableBackgroundRead(true)
		c.bufr = newBufioReader(c.vc)
		// c.bufw = newBufioWriterSize(c.vc, 4<<10)
		c.ctx, c.cancelCtx = context.WithCancel(context.Background())
		defer c.cancelCtx()

		var index int
		for {
			lb, err := c.readLineBytes()
			if err != nil {
				if isCommonNetReadError(err) {
					break
				}
				t.Fatalf("%d, error(%v)", index, err)
			}
			if !bytes.Equal(lb, []byte(ss[index])) {
				t.Fatalf("%d, 错误，读取数据不一样", index)
			}
			index++

			req, err := c.readRequest(c.ctx, lb)
			if err != nil {
				t.Fatal(err)
			}
			if req.nonce != fmt.Sprintf("%d", index) {
				t.Fatalf("预测为 1，结果为 %v", req.nonce)
			}
		}
		c.Close()
	}, func(netConn net.Conn) {
		_, err := netConn.Write([]byte(str))
		if err != nil {
			t.Fatal(err)
		}
		netConn.Close()
	})
}

func Test_conn_serve1(t *testing.T) {
	ss := []string{
		`{"nonce": "1", "proto":"IOT/1.1", "method":"POST", "header":{"Connection":"keep-alive", "a":"h1"}, "host":"1.com", "path":"/a"}`,
		`{"nonce": "2", "proto":"IOT/1.0", "method":"GET", "header":{"Connection":"keep-alive", "a":"h2"}, "host":"2.com", "path":"/b"}`,
	}
	str := strings.Join(ss, "\n")

	tcptest.C2S("127.0.0.1:0", func(netConn net.Conn) {
		c := &conn{
			server: &Server{
				ReadTimeout:  time.Second * 3,
				WriteTimeout: 0,
			},
			rwc: netConn,
		}
		c.server.SetKeepAlivesEnabled(true)
		var index int = 1
		c.server.Handler = HandlerFunc(func(w ResponseWriter, vc *Request) {
			if vc.nonce != fmt.Sprintf("%d", index) {
				t.Fatalf("预测 %v，错误 %v", index, vc.nonce)
			}

			if vc.Host != fmt.Sprintf("%d.com", index) {
				t.Fatalf("预测 %d.com，错误 %v", index, vc.Host)
			}

			w.Status(200)
			w.Header().Set("a", "a1")
			w.SetBody(index)

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

			index++
		})
		c.serve(context.Background())
	}, func(netConn net.Conn) {
		b := []byte(str)
		n, err := netConn.Write(b)
		if err != nil {
			t.Fatalf("写入错误：%v", err)
		}
		if n != len(b) {
			t.Fatalf("预测 %v, 发送 %v", len(b), n)
		}
		time.Sleep(time.Second)

		for i := 1; i < len(ss)+1; i++ {
			var riot ResponseConfig
			err = json.NewDecoder(netConn).Decode(&riot)
			if err != nil {
				t.Fatalf("读取错误：%v", err)
			}
			if riot.Status != 200 {
				t.Fatalf("返回状态是：%d", riot.Status)
			}
			if riot.Body.(float64) != float64(i) {
				t.Fatalf("返回内容是：%v，预期是：%d", riot.Body, i)
			}
		}
		netConn.Close()
	})
}

func Test_conn_serve2(t *testing.T) {
	as := assert.New(t, true)

	req, err := NewRequest("POST", "iot://a.com/", "123")
	as.NotError(err)
	riot, err := req.RequestConfig("1")
	as.NotError(err)

	var resp ResponseConfig

	tcptest.C2S("127.0.0.1:0", func(netConn net.Conn) {
		c := &conn{
			server: &Server{
				ReadTimeout:  time.Second * 3,
				WriteTimeout: 0,
				// ErrorLogLevel: LogErr | LogDebug,
			},
			rwc: netConn,
		}
		c.server.SetKeepAlivesEnabled(true)
		c.server.Handler = HandlerFunc(func(w ResponseWriter, vc *Request) {
			launch := w.(Launcher).Launch()
			go func(launch RoundTripper, r *Request) {
				r = r.WithContext(context.Background())
				res, err := launch.RoundTrip(r)
				if err != nil {
					panic(err)
				}
				if res.Status != 200 {
					panic("error")
				}
			}(launch, vc)
		})
		c.serve(context.Background())
	}, func(netConn net.Conn) {
		//------------------------------------------client to server
		// 请求
		err = json.NewEncoder(netConn).Encode(&riot)
		as.NotError(err)

		// 响应
		err = json.NewDecoder(netConn).Decode(&resp)
		as.NotError(err).Equal(resp.Status, 200)

		//------------------------------------------server to client
		// 请求
		req1, err := readRequest(netConn)
		as.NotError(err).Equal(req.Host, req1.Host)

		// 响应
		resp.Nonce = req1.GetNonce()
		err = json.NewEncoder(netConn).Encode(&resp)
		as.NotError(err)

		// 1秒后关闭
		time.Sleep(time.Second)
		netConn.Close()
	})
}

func Test_conn_setState(t *testing.T) {
	c := &conn{
		server: &Server{},
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

func Test_conn_finalFlush(t *testing.T) {
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
