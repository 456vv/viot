package viot

import (
	"context"
	"net"
	"time"

	"github.com/456vv/vconnpool/v2"
)

type Client struct {
	Dialer        vconnpool.Dialer // 拨号
	Host          string           // Host
	Addr          string           // 服务器地址
	WriteDeadline time.Duration    // 写入连接超时
	ReadDeadline  time.Duration    // 读取连接超时
}

// 快速读取
//	url string		网址
//	header Header	标头
//	resp *Response	响应
//	err error		错误
func (T *Client) Get(url string, header Header) (resp *Response, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return T.GetCtx(ctx, url, header)
}

// 快速读取（上下文）
//	ctx context.Context		上下文
//	urlstr string			网址
//	header Header			标头
//	resp *Response			响应
//	err error				错误
func (T *Client) GetCtx(ctx context.Context, urlstr string, header Header) (resp *Response, err error) {
	req, err := NewRequestWithContext(ctx, "GET", urlstr, nil)
	if err != nil {
		return nil, err
	}
	if header != nil {
		req.Header = header.Clone()
	}
	return T.DoCtx(ctx, req)
}

// 自定义请求
//	req *Request			请求
//	resp *Response			响应
//	err error				错误
func (T *Client) Do(req *Request) (resp *Response, err error) {
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	return T.DoCtx(ctx, req)
}

// 自定义请求（上下文）
//	ctx context.Context		上下文
//	req *Request			请求
//	resp *Response			响应
//	err error				错误
func (T *Client) DoCtx(ctx context.Context, req *Request) (resp *Response, err error) {
	done := make(chan bool)
	defer func() {
		if ctx.Err() != nil {
			err = ctx.Err()
			return
		}
		close(done)
	}()

	if req.Host == "" {
		req.Host = T.Host
	}

	nonce := req.nonce
	if nonce == "" {
		// 生成编号
		nonce, err = Nonce()
		if err != nil {
			return nil, err
		}
	}

	// 导出IOT支持的格式
	riot, err := req.RequestConfig(nonce)
	if err != nil {
		return nil, err
	}

	// 转字节串
	rbody, err := riot.Marshal()
	if err != nil {
		return nil, err
	}

	// 创建连接
	addr := T.Addr
	if addr == "" {
		addr = req.Host
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	if T.Dialer == nil {
		T.Dialer = new(net.Dialer)
	}
	netConn, err := T.Dialer.DialContext(ctx, tcpAddr.Network(), tcpAddr.String())
	if err != nil {
		return nil, err
	}

	vc, vcOk := netConn.(vconnpool.Conn) // 判断有没有 Discard 接口

	// 客户端不需要这条连接，不回收到池中
	if req.wantsClose() || req.Close {
		if vcOk {
			vc.Discard()
		}
	}

	// 上下文退出
	go func() {
		defer netConn.Close()
		select {
		case <-ctx.Done():
			netConn.SetDeadline(aLongTimeAgo)
			// 超时的连接不加入到池，因为服务端依然向该连接写入数据。
			// 下次使用该连接将会读取到上次的数据
			if vcOk {
				vc.Discard()
			}
		case <-done:
		}
	}()

	// 写入
	if T.WriteDeadline != 0 {
		if err := netConn.SetWriteDeadline(time.Now().Add(T.WriteDeadline)); err != nil {
			return nil, err
		}
	}

	if _, err = netConn.Write(rbody); err != nil {
		return nil, err
	}

	// 读取并解析响应
	if T.ReadDeadline != 0 {
		if err := netConn.SetReadDeadline(time.Now().Add(T.ReadDeadline)); err != nil {
			return nil, err
		}
	}

	resp, err = ReadResponse(netConn, req)
	// 服务器已经关闭连接，不回收到池中
	if err != nil || resp.Close {
		if vcOk {
			vc.Discard()
		}
	}

	return
}

// 快速提交
//	url string				网址
//	header Header			标头
//	body interface{}		主体
//	resp *Response			响应
//	err error				错误
func (T *Client) Post(url string, header Header, body interface{}) (resp *Response, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return T.PostCtx(ctx, url, header, body)
}

// 快速提交（上下文）
//	ctx context.Context		上下文
//	urlstr string			网址
//	header Header			标头
//	body interface{}		主体
//	resp *Response			响应
//	err error				错误
func (T *Client) PostCtx(ctx context.Context, urlstr string, header Header, body interface{}) (resp *Response, err error) {
	req, err := NewRequestWithContext(ctx, "POST", urlstr, body)
	if err != nil {
		return nil, err
	}
	if header != nil {
		req.Header = header.Clone()
	}
	return T.DoCtx(ctx, req)
}
