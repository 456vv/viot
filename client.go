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
	Parser        Parser           // 自定义解析接口
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
	if req.Host == "" {
		req.Host = T.Host
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
	conn, err := T.Dialer.DialContext(ctx, tcpAddr.Network(), tcpAddr.String())
	if err != nil {
		return nil, err
	}

	done := make(chan struct{})
	defer func() {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		// 判断有没有 Discard 接口
		// 超时的连接不加入到池，因为服务端依然向该连接写入数据。
		// 下次使用该连接将会读取到上次的数据
		if vc, vcOk := conn.(vconnpool.Conn); vcOk && (err != nil || req.wantsClose() || resp.Close) {
			vc.Discard()
		}
		close(done)
		conn.Close()
	}()

	if T.Parser == nil {
		T.Parser = new(defaultParse)
	}

	// 写入
	breq, err := T.Parser.Unrequest(req)
	if err != nil {
		return nil, err
	}
	if d := T.WriteDeadline; d != 0 {
		if err := conn.SetWriteDeadline(time.Now().Add(d)); err != nil {
			return nil, err
		}
	}
	if _, err = conn.Write(breq); err != nil {
		return nil, err
	}

	// 上下文退出
	go func() {
		select {
		case <-ctx.Done():
			conn.SetDeadline(aLongTimeAgo)
		case <-done:
		}
	}()

	// 读取
	if d := T.ReadDeadline; d != 0 {
		if err := conn.SetReadDeadline(time.Now().Add(d)); err != nil {
			return nil, err
		}
	}
	bres, err := readLineBytes(conn)
	if err != nil {
		return nil, err
	}
	res, err := T.Parser.Response(bres)
	if err != nil {
		return nil, err
	}
	if req.nonce != res.nonce {
		return nil, ErrRespNonce
	}
	res.Request = req
	return res, nil
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
