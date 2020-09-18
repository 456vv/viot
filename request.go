package viot

import(
	"github.com/456vv/verror"
	"crypto/tls"
	"net/url"
	"context"
	"bytes"
	//"net/http"
	"encoding/json"
	"strings"
//	"fmt"
)

//iot接收或发送数据格式带BODY
type requestConfigBody struct{
	*RequestConfig
	Body 	interface{} 		`json:"body,omitempty"`
}

//iot接收或发送数据格式
type RequestConfig struct{
	Nonce 	string				`json:"nonce"`//-,omitempty,string,number,boolean
	Proto 	string				`json:"proto"`
	Method 	string				`json:"method"`
	Path 	string				`json:"path"`
	Home	string				`json:"home"`
	Header 	Header				`json:"header"`
	body 	interface{}
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
		return nil, verror.TrackError(err)
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
		return verror.TrackError(err)
	}
	*T 		= *r.RequestConfig
	T.body	= r.Body
	return nil
}

type reqBody struct{
	Body interface{}			`json:"body"`
}

type Request struct {
	nonce 		string														// 编号
	Home		string														// 身份
	Method		string														// 方法
	RequestURI	string														// 请求URL
	URL 		*url.URL													// 路径
    Proto      	string														// 协议
    ProtoMajor 	int															// 协议大版号
    ProtoMinor 	int															// 协议小版号
	Header 		Header														// 标头
	TLS 		*tls.ConnectionState										// TLS
	RemoteAddr 	string														// 远程IP地址
	Close 		bool														// 客户要求一次性连接
	
	bodyw		interface{}													// 写入的Body数据
	datab		*bytes.Buffer												// 请求的数据(缓存让GetBody调用)
	getbodyed	bool														// 判断读取主体
	ctx			context.Context												// 上下文
  	cancelCtx   context.CancelFunc											// 上下文函数

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
	if T.getbodyed {
		return ErrGetBodyed
	}
	T.getbodyed = true
	
	//非 POST 提交，不支持提取BODY数据
	if T.datab == nil {
		return ErrBodyNotAllowed
	}
	
	err := json.NewDecoder(T.datab).Decode(&reqBody{i})
	if err == nil {
		T.datab = nil
	}	
	return verror.TrackError(err)
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
//	ok bool			如果有令牌，返回true
func (T *Request) GetTokenAuth() (token string, ok bool) {
  	auth := T.Header.Get("Authorization")
  	if auth == "" {
  		return
  	}
  	const prefix = "token "
  	if !strings.HasPrefix(auth, prefix) {
  		return
  	}
  	return auth[len(prefix):], true
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
	riot = &RequestConfig{
		Nonce	: nonce,
		Proto	: T.Proto,
		Method	: T.Method,
		Path	: T.RequestURI,
		Home	: T.Home,
		Header	: T.Header.clone(),
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










