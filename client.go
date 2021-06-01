package viot
	
import(
	"github.com/456vv/vconnpool/v2"
	"time"
	"context"
	"net/url"
	"net"
)

type Client struct{
	Dialer				vconnpool.Dialer	//拨号
	Host				string				//Host
	Addr				string				//服务器地址
	WriteDeadline		time.Duration		//写入连接超时
	ReadDeadline		time.Duration		//读取连接超时
}

//快速读取
//	url string		网址
//	header Header	标头
//	resp *Response	响应
//	err error		错误
func (T *Client) Get(url string, header Header) (resp *Response, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return T.GetCtx(ctx, url, header)
}

//快速读取（上下文）
//	ctx context.Context		上下文
//	urlstr string			网址
//	header Header			标头
//	resp *Response			响应
//	err error				错误
func (T *Client) GetCtx(ctx context.Context, urlstr string, header Header) (resp *Response, err error) {
	u, err := url.Parse(urlstr)
	if err != nil {
		return nil, err
	}
	
	if header == nil {
		header = make(Header)
	}
	
	//构建请求
	req := &Request{
		Proto		: "IOT/1.1",
		Method		: "GET",
		RequestURI	: u.RequestURI(),
		Host		: u.Host,
		Header		: header.Clone(),
	}
	return T.DoCtx(ctx, req)
}

//自定义请求
//	req *Request			请求
//	resp *Response			响应
//	err error				错误
func (T *Client) Do(req *Request) (resp *Response, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return T.DoCtx(ctx, req)
}

//自定义请求（上下文）
//	ctx context.Context		上下文
//	req *Request			请求
//	resp *Response			响应
//	err error				错误
func (T *Client) DoCtx(ctx context.Context, req *Request)(resp *Response, err error){
	defer func(){
		if ctx.Err() != nil {
			err = ctx.Err()
		}
	}()
	
	if req.Host == "" {
		req.Host = T.Host
	}
	//生成编号
	nonce, err := Nonce()
	if err != nil {
		return nil, err
	}
	//导出IOT支持的格式
	riot, err := req.RequestConfig(nonce)
	if err != nil {
		return nil, err
	}
	//转字节串
	rbody, err := riot.Marshal()
	if err != nil {
		return nil, err
	}
	
	//创建连接
	addr := T.Addr
	if addr == "" {
		addr = req.Host
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	netConn, err := T.Dialer.DialContext(ctx, tcpAddr.Network(), tcpAddr.String())
	if err != nil {
		return nil, err
	}
	defer netConn.Close()
	go func(){
		select{
		case <-ctx.Done():
			netConn.Close()
		}
	}()
	
	//写入
	if T.WriteDeadline != 0 {
		if err := netConn.SetWriteDeadline(time.Now().Add(T.WriteDeadline)); err != nil {
			return nil, err
		}
	}
	_, err = netConn.Write(rbody)
	if err != nil {
		return nil, err
	}
	
	//读取并解析响应
	if T.ReadDeadline != 0 {
		if err := netConn.SetReadDeadline(time.Now().Add(T.ReadDeadline)); err != nil {
			return nil, err
		}
	}
	
	return ReadResponse(netConn, req)
}

//快速提交
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

//快速提交（上下文）
//	ctx context.Context		上下文
//	urlstr string			网址
//	header Header			标头
//	body interface{}		主体
//	resp *Response			响应
//	err error				错误
func (T *Client) PostCtx(ctx context.Context, urlstr string, header Header, body interface{}) (resp *Response, err error) {
	u, err := url.Parse(urlstr)
	if err != nil {
		return nil, err
	}
	
	if header == nil {
		header = make(Header)
	}
	
	//构建请求
	req := &Request{
		Proto		: "IOT/1.1",
		Method		: "POST",
		RequestURI	: u.RequestURI(),
		Host		: u.Host,
		Header		: header.Clone(),
	}
	req.SetBody(body)
	return T.DoCtx(ctx, req)
}

