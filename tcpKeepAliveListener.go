package viot

import(
	"net"
	"time"
)

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	//设置false会造成数据不是一整条发送，而是断续发送
	//如果不是解析式处理json。读取到的数据是碎片的
	tc.SetNoDelay(true)
	return tc, nil
}
