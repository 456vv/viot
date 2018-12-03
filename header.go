package viot

//标头
type Header map[string]string

//设置
//	key, value string	键，值
func (h Header) Set(key, value string) {
  	h[key] = value
}
 
//读取
//	key string	键
//	string		值
func (h Header) Get(key string) string {
  	return h[key]
}
 
//删除
//	key string	键
func (h Header) Del(key string) {
  	delete(h, key)
}

//克隆
func (h Header) clone() Header {
	header := make(Header)
	for key, val := range h {
		header.Set(key, val)
	}
	return header
}