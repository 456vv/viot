package viot

import (
	"github.com/456vv/verror"
	//"fmt"
	"net"
	"bufio"
	"sync/atomic"
//	"context"
)


//响应写入接口
type ResponseWriter interface {
    Header() Header                                                         // 标头
    Status(int)                                                             // 状态
    SetBody(interface{}) error                                              // 主体
}

//劫持接口
type Hijacker interface {
    Hijack() (net.Conn, *bufio.ReadWriter, error)                           // 劫持
}

//连接关闭通知接口
type CloseNotifier interface {
	CloseNotify() <-chan bool												// 关闭通知
 }
 
 //发射，服务器使用当前连接作为客户端给智能设置发送信息
 type Launcher interface{
    Launch() RoundTripper
}
 
//响应
type responseWrite struct{
	conn             	*conn												// 上级
  	req              	*Request 											// 上级
	closeNotifyCh  		chan bool											// 收到数据，处理还没结束的时候。客户端又发来请求。则取消现有的请求，接受新的请求
	didCloseNotify 		int32 												// atomic (only 0->1 winner should send)

  	wroteStatus      	bool												// 状态写入
	status        		int													// 状态码
		
  	cw 					chunkWriter											// body数据和组装数据
  	
 	header 				Header												// 标头

  	closeAfterReply 	bool												// 服务端设置不关闭连接
  	
  	handlerDone 		atomicBool 											// 判断本次响应是否已经完成
  	
  	isWrite				bool												// 原样数据写入
 }


//原样写入
//	b []byte	字节串
//	int, error	写入数量，错误
 func (T *responseWrite) Write(b []byte) (int, error) {
 	 return T.write(len(b), b, "")
}
 func (T *responseWrite) WriteString(s string) (n int, err error) {
 	 return T.write(len(s), nil, s)
 }
 func (T *responseWrite) write(lenData int, dataB []byte, dataS string) (n int, err error) {
	if T.conn.hijackedv.isTrue() {
  		T.conn.server.logf("viot: 此连接已经劫持，不允许使用此函数Write")
  		return 0, ErrHijacked
  	}
  	
	if lenData == 0 {
		return 0, nil
	}
  	T.isWrite = true
	if dataB != nil {
		return T.conn.bufw.Write(dataB)
	}
	return T.conn.bufw.WriteString(dataS)
 }

//写入状态
//	code int	状态码
func (T *responseWrite) Status(code int) {
	if T.conn.hijackedv.isTrue() {
  		T.conn.server.logf("viot: 此连接已经劫持，不允许使用此函数 .Status()")
  		return
  	}
  	T.wroteStatus = true
  	T.status = code
}

//状态码有效性
func (T *responseWrite) bodyAllowed() bool {
  	return bodyAllowedForStatus(T.status)
}

//默认状态码
func (T *responseWrite) default200Status() {
  	if !T.wroteStatus {
  		T.Status(200)
  	}
}

//写入标头
//	Header	标头
func (T *responseWrite) Header() Header {
	if T.header == nil {
		T.header = make(Header)
	}
  	return T.header
}

//写入数据
//	data interface{}	主体数据
//	error				错误
func (T *responseWrite) SetBody(data interface{}) error {
	if T.conn.hijackedv.isTrue() {
  		T.conn.server.logf("viot: 此连接已经劫持，不允许使用此函数 .SetBody()")
  		return ErrHijacked
  	}
  	
  	//仅在正确的状态码情况下，才能调用此函数
	T.default200Status()
	if !T.bodyAllowed() {
  		return ErrBodyNotAllowed
  	}
  	
  	return T.cw.SetBody(data)
}

//设置关闭通知
func (T *responseWrite) closeNotify() {
	//防止多次调用
	if atomic.CompareAndSwapInt32(&T.didCloseNotify, 0, 1) {
		T.closeNotifyCh <- true
	}
}

//读取关闭通知
//	<-chan bool		关闭事件
func (T *responseWrite) CloseNotify() <-chan bool {
  	if T.handlerDone.isTrue() {
  		panic("viot: 响应处理完成，不允许再调用CloseNotify")
  	}
  	return T.closeNotifyCh
}

//完成
func (T *responseWrite) done() error {
 	T.handlerDone.setTrue() //设置完成标识
 	var err error
 	if !T.isWrite {
 		err = T.cw.done()
 	}
 	T.conn.bufw.Flush()
 	return err
}

//劫持连接
//	rwc net.Conn			原连接
//	buf *bufio.ReadWriter	读取缓冲
//	err error				错误
func (T *responseWrite) Hijack() (rwc net.Conn, buf *bufio.ReadWriter, err error) {
	if T.handlerDone.isTrue() {
		return nil, nil, verror.TrackErrorf("响应处理完成，不允许再调用Hijack")
  	}
  	
  	return T.conn.hijackLocked()
}

//发射
//	tr RoundTripper	转发
func (T *responseWrite) Launch() RoundTripper {
	return T.conn
}















