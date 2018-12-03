package viot

import(
	"context"
)
type RoundTripper interface {                                                               // 执行一个单一的HTTP事务
    RoundTrip(*Request) (*Response, error)                                                          // 单一的HTTP请求
	RoundTripContext(ctx context.Context, req *Request) (resp *Response, err error)					// 单一的HTTP请求(上下文)
}




