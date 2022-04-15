package viot

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/456vv/vconnpool/v2"
	"github.com/456vv/x/tcptest"
	"github.com/issue9/assert/v2"
)

func connPool() *vconnpool.ConnPool {
	return &vconnpool.ConnPool{
		Dialer: &net.Dialer{
			Timeout:   5 * time.Second,
			DualStack: true,
			KeepAlive: time.Minute,
		},
		IdeConn:    100,
		MaxConn:    0,
		IdeTimeout: 60 * time.Second,
	}
}

func Test_Client_1(t *testing.T) {
	tcptest.D2S("127.0.0.1:0", func(c net.Conn) {
		defer c.Close()
		for {
			req, err := readRequest(c)
			if err != nil {
				break
			}
			resqConfig := ResponseConfig{
				Nonce:  req.nonce,
				Status: 200,
			}
			req.GetBody(&resqConfig.Body)

			respByte, err := resqConfig.Marshal()
			if err != nil {
				break
			}
			c.Write(respByte)
		}
	}, func(laddr net.Addr) {
		cp := connPool()
		defer cp.CloseIdleConnections()
		client := Client{
			Dialer:        cp,
			Host:          "*",
			Addr:          laddr.String(),
			WriteDeadline: 5 * time.Second,
			ReadDeadline:  5 * time.Second,
		}
		t.Parallel()
		for i := 0; i < 10; i++ {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				resq, err := client.Post("/test", Header{}, "123456")
				assert.New(t, true).NotError(err).Equal(resq.Status, 200)
			})
		}
	})
}

func Test_Client_2(t *testing.T) {
	as := assert.New(t, true)
	cp := connPool()
	defer cp.CloseIdleConnections()

	tcptest.D2L("127.0.0.1:0", func(l net.Listener) {
		srv := &Server{}
		srv.ReadTimeout = 2e9
		srv.WriteTimeout = 2e9
		srv.Handler = HandlerFunc(func(w ResponseWriter, r *Request) {
			Error(w, "error", 500)
		})
		err := srv.Serve(l)
		as.Error(err)
	}, func(addr net.Addr) {
		client := &Client{
			Dialer:        cp,
			Host:          "*",
			Addr:          addr.String(),
			ReadDeadline:  2e9,
			WriteDeadline: 2e9,
		}

		t.Parallel()
		for i := 0; i < 100; i++ {
			name := fmt.Sprint("/", i)
			t.Run(name, func(t *testing.T) {
				res, err := client.Post(name, Header{}, "123")
				as.NotError(err).NotNil(res)
			})
		}
	})
}
