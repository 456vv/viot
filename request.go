package viot

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/456vv/vmap/v2"
	"github.com/456vv/vweb/v2/builtin"
)

type pastRequestConfig struct {
	*RequestConfig
	Home string `json:"home,omitempty"`
}

//iot接收或发送数据格式带BODY
type requestConfigBody struct {
	*RequestConfig
	Body interface{} `json:"body,omitempty"`
}

//iot接收或发送数据格式
type RequestConfig struct {
	Nonce  string `json:"nonce"` //-,omitempty,string,number,boolean
	Proto  string `json:"proto"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Host   string `json:"host"`
	Header Header `json:"header"`
	body   interface{}
}

//设置主体
//	i interface{}	主体信息
func (T *RequestConfig) SetBody(i interface{}) {
	T.body = i
}

//读取主体
//	interface{}		主体信息
func (T *RequestConfig) GetBody() interface{} {
	return T.body
}

//编码，字节末尾追加上一个 \\n 字节
//	[]byte			编码后的字节
//	error			错误
func (T *RequestConfig) Marshal() ([]byte, error) {
	b, err := json.Marshal(&requestConfigBody{T, T.body})
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

//解码
//	data []byte		编码后的字节
//	error			错误
func (T *RequestConfig) Unmarshal(data []byte) error {
	var r requestConfigBody
	err := json.Unmarshal(data, &r)
	if err != nil {
		return err
	}
	*T = *r.RequestConfig
	T.body = r.Body
	return nil
}

type reqBody struct {
	Body interface{} `json:"body"`
}

type Request struct {
	nonce      string               // 编号
	Host       string               // 身份
	Method     string               // 方法
	RequestURI string               // 请求URL
	URL        *url.URL             // 路径
	Proto      string               // 协议，默认IOT/1.1
	ProtoMajor int                  // 协议大版号
	ProtoMinor int                  // 协议小版号
	Header     Header               // 标头
	TLS        *tls.ConnectionState // TLS
	RemoteAddr string               // 远程IP地址
	Close      bool                 // 客户要求一次性连接

	bodyw     interface{}        // 写入的Body数据
	bodyr     *bytes.Buffer      // 请求的数据(缓存让GetBody调用)
	ctx       context.Context    // 上下文
	cancelCtx context.CancelFunc // 上下文函数

}

//新的请求
//	method, url string		方法，地址(viot://host/path)
//	body interface{}		数据
//	*Request, error			请求，错误
func NewRequest(method, url string, body interface{}) (*Request, error) {
	return NewRequestWithContext(context.Background(), method, url, body)
}

//新的请求
//	ctx context.Context		上下文
//	method, urlStr string	方法，地址(viot://host/path)
//	body interface{}		数据
//	*Request, error			请求，错误
func NewRequestWithContext(ctx context.Context, method, urlStr string, body interface{}) (*Request, error) {
	if method == "" {
		method = "GET"
	}
	if ctx == nil {
		return nil, errors.New("viot: nil Context")
	}
	if !validMethod(method) {
		return nil, fmt.Errorf("viot: invalid method %q", method)
	}

	nonce, err := Nonce()
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	if body != nil {
		switch v := body.(type) {
		case *bytes.Buffer:
			body = json.RawMessage(v.Bytes())
		case *bytes.Reader:
			b, err := io.ReadAll(v)
			if err != nil {
				return nil, err
			}
			body = json.RawMessage(b)
		case *strings.Reader:
			b, err := io.ReadAll(v)
			if err != nil {
				return nil, err
			}
			body = json.RawMessage(b)
		case *vmap.Map:
			b, err := v.MarshalJSON()
			if err != nil {
				return nil, err
			}
			body = json.RawMessage(b)
		case []byte:
			body = json.RawMessage(v)
		default:
		}
	}
	u.Host = removeEmptyPort(u.Host)
	req := &Request{
		nonce:      nonce,
		Method:     method,
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Proto:      "IOT/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(Header),
		bodyw:      body,
	}
	req.ctx, req.cancelCtx = context.WithCancel(ctx)
	return req, nil
}

//读取编号
//	string	请求编号
func (T *Request) GetNonce() string {
	return T.nonce
}

// 读取主体
//	i interface{}	数据写入这里
//	error			错误
func (T *Request) GetBody(i interface{}) error {
	//这是开发者自行创建的Request，设置SetBody后可以调用GetBody读出
	if T.bodyw != nil {
		if !builtin.Convert(i, T.bodyw) {
			return errors.New("viot:  the type is not supported")
		}
		return nil
	}

	//非 POST 提交，不支持提取BODY数据
	if T.bodyr == nil {
		return ErrGetBodyed
	}

	if err := json.NewDecoder(T.bodyr).Decode(&reqBody{i}); err != nil {
		return err
	}
	T.bodyr = nil
	return nil
}

// 设置主体
//	i interface{}	数据写入这里
//	error			错误
func (T *Request) SetBody(i interface{}) error {
	T.bodyw = i
	return nil
}

//判断版本号
func (T *Request) ProtoAtLeast(major, minor int) bool {
	return T.ProtoMajor > major || T.ProtoMajor == major && T.ProtoMinor >= minor
}

//应该关闭
func (T *Request) wantsClose() bool {
	return hasToken(T.Header.Get("Connection"), "close")
}

//读取上下文
//	context.Context	上下文
func (T *Request) Context() context.Context {
	if T.ctx != nil {
		return T.ctx
	}
	return context.Background()
}

//替换上下文
//	ctx context.Context	上下文
//	*Request			请求
func (T *Request) WithContext(ctx context.Context) *Request {
	if ctx == nil {
		panic("nil context")
	}
	r2 := new(Request)
	*r2 = *T
	r2.ctx = ctx

	if T.URL != nil {
		r2URL := new(url.URL)
		*r2URL = *T.URL
		r2.URL = r2URL
	}

	return r2
}

//基本验证
//	username, password string	用户名，密码
//	ok bool						如果有用户名或密码
func (T *Request) GetBasicAuth() (username, password string, ok bool) {
	auth := T.Header.Get("Authorization")
	if auth == "" {
		return
	}
	return parseBasicAuth(auth)
}

//设置基本验证
//	username, password string	用户名，密码
func (T *Request) SetBasicAuth(username, password string) {
	T.Header.Set("Authorization", "Basic "+basicAuth(username, password))
}

//token验证
//	token string	令牌
func (T *Request) GetTokenAuth() string {
	auth := T.Header.Get("Authorization")
	if auth == "" {
		return auth
	}
	const prefix = "token "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}
	return auth[len(prefix):]
}

//设置token验证
//	token string	令牌
func (T *Request) SetTokenAuth(token string) {
	T.Header.Set("Authorization", "token "+token)
}

//请求，发往设备的请求
//	nonce string		编号
//	riot *RequestConfig	发往设备的请求
//	err error			错误
func (T *Request) RequestConfig(nonce string) (riot *RequestConfig, err error) {

	host := T.Host
	path := T.RequestURI
	if T.URL != nil {
		if host == "" {
			host = T.URL.Host
		}
		if path == "" {
			path = T.URL.RequestURI()
		}
	}
	if host == "" {
		return nil, ErrHostInvalid
	}
	if path == "" {
		return nil, ErrURIInvalid
	}

	proto := T.Proto
	if proto == "" {
		if !T.ProtoAtLeast(1, 0) {
			return nil, ErrProtoInvalid
		}
		proto = fmt.Sprintf("IOT/%d.%d", T.ProtoMajor, T.ProtoMinor)
	}

	if T.Method == "" {
		return nil, ErrMethodInvalid
	}

	riot = &RequestConfig{
		Nonce:  nonce,
		Proto:  proto,
		Method: T.Method,
		Path:   path,
		Host:   host,
		Header: T.Header.Clone(),
	}

	if T.Close {
		riot.Header.Set("Connection", "close")
	}

	//如果没有另设置body，试试读取原有的body
	if T.bodyw == nil {
		T.GetBody(&T.bodyw)
	}
	riot.SetBody(&T.bodyw)
	return riot, nil
}
