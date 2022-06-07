package viot

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/456vv/vconn"
	"github.com/456vv/x/tcptest"
	"github.com/issue9/assert/v2"
)

func Test_conn_readLineBytes(t *testing.T) {
	as := assert.New(t, true)
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

		c.vc = vconn.New(c.rwc)
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
			as.Equal(lb, []byte(ss[index]))
			index++
		}

		c.close()
	}, func(netConn net.Conn) {
		b := []byte(str)
		n, err := netConn.Write(b)
		as.NotError(err).Equal(n, len(b))

		io.Copy(ioutil.Discard, netConn)
		netConn.Close()
	})
}

func Test_conn_readRequest(t *testing.T) {
	as := assert.New(t, true)
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

		c.vc = vconn.New(c.rwc)
		c.bufr = newBufioReader(c.vc)
		c.ctx, c.cancelCtx = context.WithCancel(context.Background())
		defer c.cancelCtx()

		var index int
		for {
			c.parserChange()
			lb, err := c.readLineBytes()
			if err != nil {
				if isCommonNetReadError(err) {
					break
				}
				t.Fatalf("%d, error(%v)", index, err)
			}
			as.Equal(lb, []byte(ss[index]), index)

			index++

			req, err := c.readRequest(c.ctx, lb)
			as.NotError(err)
			as.Equal(req.nonce, fmt.Sprintf("%d", index))
		}
		c.close()
	}, func(netConn net.Conn) {
		_, err := netConn.Write([]byte(str))
		as.NotError(err)
	})
}

func Test_conn_serve1(t *testing.T) {
	as := assert.New(t, true)
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
		c.server.Handler = HandlerFunc(func(w ResponseWriter, r *Request) {
			as.Equal(r.GetNonce(), fmt.Sprintf("%d", index))
			as.Equal(r.Host, fmt.Sprintf("%d.com", index))

			w.Status(200)
			w.Header().Set("a", "a1")
			w.SetBody(index)

			res := w.(*responseWrite)
			a1 := res.header.Get("a")
			as.Equal(res.status, 200)
			as.NotEqual(res.Header, nil)
			as.Equal(a1, "a1")
			as.False(res.handlerDone.isTrue())

			index++
		})
		c.serve(context.Background())
	}, func(netConn net.Conn) {
		b := []byte(str)
		n, err := netConn.Write(b)
		as.NotError(err).Equal(n, len(b))

		for i := 1; i < len(ss)+1; i++ {
			var rc ResponseConfig
			err = json.NewDecoder(netConn).Decode(&rc)
			as.NotError(err)
			as.Equal(rc.Status, 200)
			as.Equal(rc.Body, i)
		}
	})
}

func Test_conn_serve2(t *testing.T) {
	as := assert.New(t, true)
	tcptest.C2S("127.0.0.1:0", func(netConn net.Conn) {
		c := &conn{
			server: &Server{
				ReadTimeout:  time.Second * 3,
				WriteTimeout: 0,
			},
			rwc: netConn,
		}
		c.server.SetKeepAlivesEnabled(true)
		c.server.Handler = HandlerFunc(func(w ResponseWriter, r *Request) {
			launch := w.(Launcher).Launch()
			go func(launch RoundTripper, r *Request) {
				r = r.WithContext(context.Background())
				res, err := launch.RoundTrip(r)
				as.NotError(err).Equal(res.Status, 200)
			}(launch, r)
		})
		c.serve(context.Background())
	}, func(netConn net.Conn) {
		req, err := NewRequest("POST", "iot://a.com/", "123")
		as.NotError(err)

		b, err := req.Marshal()
		as.NotError(err)

		var resp ResponseConfig
		//------------------------------------------client to server
		// 请求
		netConn.Write(b)

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
	})
}

type customParse struct {
	*defaultParse
}

func (T *customParse) Request(b []byte) (*Request, error) {
	if !bytes.Equal(b, []byte{0, 1, 2, 3, 4, 5}) {
		return nil, ErrReqUnavailable
	}
	return NewRequest("POST", "iot://www.leihe.com/abc", "第二请求")
}

func (T *customParse) Unresponse(res *Response) ([]byte, error) {
	if res.Status != 400 {
		return nil, errors.New("error")
	}
	return []byte{5, 4, 3, 2, 1, 0}, nil
}

func Test_conn_serve3(t *testing.T) {
	as := assert.New(t, true)
	cp := &customParse{new(defaultParse)}

	tcptest.C2S("127.0.0.1:0", func(netConn net.Conn) {
		c := &conn{
			server: &Server{
				ReadTimeout:  time.Second * 3,
				WriteTimeout: 0,
			},
			rwc: netConn,
		}
		c.server.SetKeepAlivesEnabled(true)
		var once sync.Once
		c.server.Handler = HandlerFunc(func(w ResponseWriter, r *Request) {
			var first bool
			var body interface{}
			once.Do(func() {
				first = true
				err := r.GetBody(&body)
				as.NotError(err).Equal(body, "第一请求")

				w.(SetParser).SetParse(cp)

				w.Status(200)
				w.SetBody("第一响应")
			})

			if !first {
				err := r.GetBody(&body)
				as.NotError(err).Equal(body, "第二请求").Equal(r.Method, "POST")

				w.Status(400)
				w.SetBody("第二响应")
			}
		})
		c.serve(context.Background())
	}, func(netConn net.Conn) {
		req, err := NewRequest("POST", "iot://a.com/abc", "第一请求")
		as.NotError(err)

		//------------------------------------------client to server
		// 请求
		b, err := req.Marshal()
		as.NotError(err)
		netConn.Write(b)

		// 响应
		var resp ResponseConfig
		err = json.NewDecoder(netConn).Decode(&resp)
		as.NotError(err).Equal(resp.Status, 200).Equal(resp.Body, "第一响应")

		//------------------------------------------custom format
		//自定义发送
		netConn.Write([]byte{0, 1, 2, 3, 4, 5})

		// 收到自定义
		p := make([]byte, 100)
		n, err := netConn.Read(p)
		as.NotError(err).Equal([]byte{5, 4, 3, 2, 1, 0}, p[:n])
	})
}

func Test_conn_serve4(t *testing.T) {
	as := assert.New(t, true)
	tcptest.C2S("127.0.0.1:0", func(netConn net.Conn) {
		c := &conn{
			server: &Server{
				ReadTimeout:  time.Second * 3,
				WriteTimeout: 0,
			},
			rwc: netConn,
		}
		c.server.SetKeepAlivesEnabled(true)
		var once sync.Once
		var second bool
		c.server.Handler = HandlerFunc(func(w ResponseWriter, r *Request) {
			var body interface{}
			once.Do(func() {
				err := r.GetBody(&body)
				as.NotError(err).Equal(body, "第一请求")

				w.Status(200)
				w.SetBody("第一响应")

				w.(RawControler).RawControl(func(c net.Conn, r *bufio.Reader) error {
					second = true
					// 第二请求
					p := make([]byte, 100)
					n, err := r.Read(p)
					as.NotError(err).Equal([]byte{0, 1, 2, 3, 4, 5}, p[:n])

					// 第二响应
					c.Write([]byte{5, 4, 3, 2, 1, 0})
					return nil
				})
			})

			if second {
				err := r.GetBody(&body)
				as.NotError(err).Equal(body, "第三请求").Equal(r.Method, "POST")

				w.Status(400)
				w.SetBody("第三响应")
			}
		})
		c.serve(context.Background())
	}, func(netConn net.Conn) {
		req, err := NewRequest("POST", "iot://a.com/abc", "第一请求")
		as.NotError(err)

		//------------------------------------------client to server
		// 请求
		b, err := req.Marshal()
		as.NotError(err)
		netConn.Write(b)

		// 响应
		var resp ResponseConfig
		err = json.NewDecoder(netConn).Decode(&resp)
		as.NotError(err).Equal(resp.Status, 200).Equal(resp.Body, "第一响应")

		//------------------------------------------binary format
		//二进制发送
		netConn.Write([]byte{0, 1, 2, 3, 4, 5})

		// 收到二进制
		p := make([]byte, 100)
		n, err := netConn.Read(p)
		as.NotError(err).Equal([]byte{5, 4, 3, 2, 1, 0}, p[:n])

		//--------------------------------------------third client to server
		// 请求
		req.SetBody("第三请求")
		b, err = req.Marshal()
		as.NotError(err)
		netConn.Write(b)

		// 响应
		err = json.NewDecoder(netConn).Decode(&resp)
		as.NotError(err).Equal(resp.Status, 400).Equal(resp.Body, "第三响应")
	})
}

func Test_conn_setState(t *testing.T) {
	c := &conn{
		server: &Server{},
	}
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
