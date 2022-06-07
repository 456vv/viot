package viot

import (
	"strconv"
	"time"
)

type dataWriter struct {
	res    *responseWrite // 上级
	resp   *Response
	body   interface{} // body 数据
	fulfil atomicBool  // 判断是否调用了.done(), 避免重复调用
}

// 生成响应
func (T *dataWriter) generateResponse() {
	// 写入标头
	T.res.default200Status()
	T.resp = &Response{
		Status:  T.res.status,
		Header:  T.res.Header(),
		Body:    T.body,
		Close:   false,
		Request: T.res.req,
		nonce:   T.res.req.nonce,
	}

	keepAlivesEnabled := T.res.conn.server.doKeepAlives() // 服务支持长连接
	if !T.res.req.ProtoAtLeast(1, 1) || T.res.req.wantsClose() || !keepAlivesEnabled {
		// 客户端协议1.0 或 设置了Connection : close。则设置关闭
		T.resp.Header.Set("Connection", "close")
		T.resp.Close = true
	} else if keepAlivesEnabled && (T.res.req.Method == "HEAD" || T.res.bodyAllowed()) {
		// 客户端支持长连接，要求是长连接并状态码正确
		// 如果没有设置Connection，则设置为keep-alive
		if _, ok := T.resp.Header["Connection"]; !ok {
			T.resp.Header.Set("Connection", "keep-alive")
		}
	}

	// 关闭连接
	T.res.closeAfterReply = (T.resp.Header.Get("Connection") != "keep-alive")

	if _, ok := T.resp.Header["Timestamp"]; !ok {
		unix := time.Now().UTC().Unix()
		sUnix := strconv.FormatInt(unix, 10)
		T.resp.Header.Set("Timestamp", sUnix)
	}
}

// body内容写入
func (T *dataWriter) setBody(data interface{}) error {
	T.body = data
	return nil
}

// 无body内容写入
func (T *dataWriter) done() error {
	if T.fulfil.setTrue() {
		return ErrDoned
	}

	T.generateResponse()
	lineBytes, err := T.res.conn.parser.Unresponse(T.resp)
	if err != nil {
		return err
	}
	return T.res.conn.writeLineByte(lineBytes)
}
