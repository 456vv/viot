package viot

import(
	"encoding/json"
	"io"
)


type ResponseConfig struct{
	Nonce 	string						`json:"nonce"`//-,omitempty,string,number,boolean
	Status	int							`json:"status"`
	Header 	Header						`json:"header"`
	Body 	interface{}					`json:"body,omitempty"`
}


//响应
type Response struct{
    Status     	int															// 状态码
	Header 		Header														// 标头
	Body		interface{}													// 主体
	Close		bool														// 服务器关闭连接
	Request		*Request													// 请求
	RemoteAddr	string														// 远程IP
	
	nonce 		string														// 编码
}

//设置响应编号
//	n string	编号
func (T *Response) SetNonce(n string) {
	T.nonce = n
}

//写入到
//	w ResponseWriter	T响应写w
func (T *Response) WriteTo(w ResponseWriter) {
	w.Status(T.Status)
	h := w.Header()
	for k, v := range T.Header {
		h.Set(k, v)
	}
	w.SetBody(T.Body)
}

//写入w
//	w io.Writer		T响应写w
func (T *Response) Write(w io.Writer) error {
	ir := &ResponseConfig{
		Nonce	: T.nonce,
		Status	: T.Status,
		Header	: T.Header.clone(),
		Body	: T.Body,
	}
	return json.NewEncoder(w).Encode(ir)
}

//转IOT支持的格式
//	nonce string		编号
//	riot *ResponseConfig	IOT响应数据格式
//	err error			错误
func (T *Response) ResponseConfig(nonce string) (riot *ResponseConfig, err error) {
	ir := &ResponseConfig{
		Nonce	: nonce,
		Status	: T.Status,
		Header	: T.Header.clone(),
		Body	: T.Body,
	}
	return ir, nil
}