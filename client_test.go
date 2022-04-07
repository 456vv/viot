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

func Test_Client(t *testing.T) {

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
		iotConnPool := &vconnpool.ConnPool{
			Dialer: &net.Dialer{
				Timeout:   5 * time.Second,
				DualStack: true,
				KeepAlive: time.Minute,
			},
			IdeConn:    100,
			MaxConn:    0,
			IdeTimeout: 60 * time.Second,
		}
		defer iotConnPool.Close()

		client := Client{
			Dialer:        iotConnPool,
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
