package viot

import(
	"github.com/456vv/vmap/v2"
	"time"
	"encoding/json"
)

type dataWriter struct {
  	res 			*responseWrite		// 上级
  	data			*vmap.Map			// 数据结构
  	body			interface{}			// body 数据
  	fulfil			bool				// 判断是否调用了.done(), 避免重复调用
}

//生成响应
func (T *dataWriter) generateResponse() {
 	if T.data != nil {
 		return
 	}
 	
 	T.data 		= vmap.NewMap()
  	keepAlivesEnabled := T.res.conn.server.doKeepAlives() //服务支持长连接
  	isHEAD 		:= T.res.req.Method == "HEAD"
 	setHeader 	:= T.data.GetNewMap("header")
	headerw 	:= T.res.Header()

	if !T.res.req.ProtoAtLeast(1, 1) || T.res.req.wantsClose() || !keepAlivesEnabled {
		//客户端协议1.0 或 设置了Connection : close。则设置关闭
  		headerw.Set("Connection","close")
	}else if keepAlivesEnabled && (isHEAD || T.res.bodyAllowed()) {
		//客户端支持长连接，要求是长连接并状态码正确
		//如果没有设置Connection，则设置为keep-alive
		_, connectionHeaderSet := headerw["Connection"]
  		if !connectionHeaderSet {
  			headerw.Set("Connection","keep-alive")
  		}
	}

	//关闭连接
	T.res.closeAfterReply = (headerw.Get("Connection") != "keep-alive")
	
	if _, ok := headerw["Date"]; !ok {
  		headerw.Set("Date", time.Now().UTC().Format("2006-01-02 15:04:05"))
  	}
  	
  	//写入标头
  	setHeader.ReadFrom(headerw)
	T.res.default200Status()
  	T.data.Set("status", T.res.status)
  	T.data.Set("nonce", T.res.req.nonce)
  	T.data.Set("body", T.body)
}

//body内容写入
func (T *dataWriter) SetBody(data interface{}) error {
	T.body = data
	return nil
}

//无body内容写入
func (T *dataWriter) done() error {
	if T.fulfil {
		return ErrDoned
	}
	T.fulfil = true
	
  	T.generateResponse()//写入
  	
  	lineBytes, err := json.Marshal(T.data)
  	if err != nil {
  		return err
  	}
  	lineBytes = append(lineBytes, '\n')
  	return T.res.conn.writeLineByte(lineBytes)
}

