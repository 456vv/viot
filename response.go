package viot

import (
	"encoding/json"
	"io"
)

type ResponseConfig struct {
	Nonce  string      `json:"nonce"` //-,omitempty,string,number,boolean
	Status int         `json:"status"`
	Header Header      `json:"header"`
	Body   interface{} `json:"body,omitempty"`
}

// 编码，字节末尾追加上一个 \\n 字节
//	[]byte			编码后的字节
//	error			错误
func (T *ResponseConfig) Marshal() ([]byte, error) {
	b, err := json.Marshal(T)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// 响应
type Response struct {
	Status  int         // 状态码
	Header  Header      // 标头
	Body    interface{} // 主体
	Close   bool        // 服务器关闭连接
	Request *Request    // 请求

	nonce string // 编码
}

// 设置响应编号
//	n string	编号
func (T *Response) SetNonce(n string) {
	T.nonce = n
}

// 写入到
//	w ResponseWriter	T响应写w
func (T *Response) WriteAt(w ResponseWriter) {
	w.Status(T.Status)
	h := w.Header()
	for k, v := range T.Header {
		h.Set(k, v)
	}
	w.SetBody(T.Body)
}

// 写入w
//	w io.Writer		T响应写w
func (T *Response) WriteTo(w io.Writer) (int64, error) {
	b, err := T.Marshal()
	if err != nil {
		return 0, err
	}
	n, err := w.Write(b)
	return int64(n), err
}

// 封装 ResponseConfig.Marshal
func (T *Response) Marshal() ([]byte, error) {
	if T.Request != nil && T.nonce == "" {
		T.nonce = T.Request.nonce
	}
	rc, err := T.Config(T.nonce)
	if err != nil {
		return nil, err
	}
	return rc.Marshal()
}

// 转IOT支持的格式
//	nonce string		编号
//	riot *ResponseConfig	IOT响应数据格式
//	err error			错误
func (T *Response) Config(nonce string) (rc *ResponseConfig, err error) {
	rc = &ResponseConfig{
		Nonce:  nonce,
		Status: T.Status,
		Header: T.Header.Clone(),
		Body:   T.Body,
	}
	return rc, nil
}
