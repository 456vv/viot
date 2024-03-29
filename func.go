package viot

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http/httpguts"
)

// 在切片中查找
func strSliceContains(ss []string, t string) bool {
	for _, v := range ss {
		if v == t {
			return true
		}
	}
	return false
}

// 查找是否包含所有字段
// 判断 ss 列表中的关键字，是否在 b 存在
func byteFindAll(b []byte, ss []string) bool {
	for _, v := range ss {
		if !bytes.Contains(b, []byte(v)) {
			return false
		}
	}
	return true
}

// 是请求
func isRequest(lineBytes []byte) bool {
	if byteFindAll(lineBytes, reqFeature1) || byteFindAll(lineBytes, reqFeature2) {
		return true
	}
	return false
}

// 是响应
func isResponse(lineBytes []byte) bool {
	return byteFindAll(lineBytes, respFeature1)
}

// 判断方法
func validMethod(method string) bool {
	return len(method) > 0 && strSliceContains(methods, method)
}

// 判断协议
func validNPN(proto string) bool {
	switch proto {
	case "", "IOT/1.1", "IOT/1.0":
		return false
	}
	return true
}

// 解析IOT请求版本
//	vers string			版本字符串
//	major, minor int	大版号，小版号
//	ok bool				true版本正确解析，否则失败
func ParseIOTVersion(vers string) (major, minor int, ok bool) {
	const Big = 1000000 // arbitrary upper bound
	switch vers {
	case "IOT/1.1":
		return 1, 1, true
	case "IOT/1.0":
		return 1, 0, true
	}
	if !strings.HasPrefix(vers, "IOT/") {
		return 0, 0, false
	}
	dot := strings.Index(vers, ".")
	if dot < 0 {
		return 0, 0, false
	}
	major, err := strconv.Atoi(vers[4:dot])
	if err != nil || major < 0 || major > Big {
		return 0, 0, false
	}
	minor, err = strconv.Atoi(vers[dot+1:])
	if err != nil || minor < 0 || minor > Big {
		return 0, 0, false
	}
	return major, minor, true
}

var textprotoReaderPool sync.Pool

// 创建文本格式读取
func newTextprotoReader(br *bufio.Reader) *textproto.Reader {
	if v := textprotoReaderPool.Get(); v != nil {
		tr := v.(*textproto.Reader)
		tr.R = br
		return tr
	}
	return textproto.NewReader(br)
}

// 回收文本格式读取
func putTextprotoReader(r *textproto.Reader) {
	r.R = nil
	textprotoReaderPool.Put(r)
}

// 读取请求数据
//	b io.Reader		需解析的数据，重要提醒：不要包含多个json块，它只能解析一个json块，其它数据块会被丢弃。这会清空你的io.Reader。
//	req *Request	请求
//	err error		错误
func ReadRequest(b io.Reader) (req *Request, err error) {
	return readRequest(b)
}

func readRequest(b io.Reader) (req *Request, err error) {
	bufr := newBufioReader(b)
	defer func() {
		putBufioReader(bufr)
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	req = new(Request)
	req.bodyr = new(bytes.Buffer)
	//{json}

	var ij pastRequestConfig
	err = json.NewDecoder(io.TeeReader(bufr, req.bodyr)).Decode(&ij)
	if err != nil {
		return nil, fmt.Errorf("incorrect format of request content %v", err)
	}
	if ij.Nonce == "" {
		return nil, fmt.Errorf("the request nonce serial number cannot be \"\"")
	}
	if !validMethod(ij.Method) {
		return nil, fmt.Errorf("request invalid method %q", ij.Method)
	}

	var ok bool
	if req.ProtoMajor, req.ProtoMinor, ok = ParseIOTVersion(ij.Proto); !ok {
		return nil, fmt.Errorf("IOT version with incorrect request format %q", ij.Proto)
	}

	if req.URL, err = url.ParseRequestURI(ij.Path); err != nil {
		return nil, err
	}

	// 释放内存，仅POST提交才支持body
	if ij.Method != "POST" {
		req.bodyr = nil
	}

	req.Header = ij.Header.Clone()
	for hk, hv := range req.Header {
		if !httpguts.ValidHeaderFieldName(hk) {
			return nil, fmt.Errorf("invalid header name %s", hk)
		}
		if !httpguts.ValidHeaderFieldValue(hv) {
			return nil, fmt.Errorf("invalid header value %s", hv)
		}
	}
	if ij.Home != "" && ij.Host == "" {
		// 向下兼容，已经过期
		ij.Host = ij.Home
	}
	req.nonce = ij.Nonce
	req.Method = ij.Method
	req.RequestURI = ij.Path
	req.Proto = ij.Proto
	req.Host = ij.Host
	req.Close = shouldClose(req.ProtoMajor, req.ProtoMinor, req.Header)

	return req, nil
}

// 解析响应
//	b io.Reader		需解析的数据，重要提醒：不要包含多个json块，它只能解析一个json块，其它数据块会被丢弃。这会清空你的io.Reader。
//	res *Response	响应
//	err error		错误
func ReadResponse(r io.Reader, req *Request) (res *Response, err error) {
	res, err = readResponse(r)
	if err != nil {
		return
	}
	if req.nonce != res.nonce {
		return nil, ErrRespNonce
	}
	res.Request = req
	return
}

func readResponse(b io.Reader) (res *Response, err error) {
	bufr := newBufioReader(b)
	defer putBufioReader(bufr)

	res = new(Response)
	//{json}

	var riot ResponseConfig
	err = json.NewDecoder(bufr).Decode(&riot)
	if err != nil {
		return nil, fmt.Errorf("read response content error %v", err)
	}

	if riot.Nonce == "" {
		return nil, fmt.Errorf("the response nonce serial number is \"\"")
	}
	res.Header = riot.Header.Clone()
	for hk, hv := range res.Header {
		if !httpguts.ValidHeaderFieldName(hk) {
			return nil, fmt.Errorf("invalid title name %s", hk)
		}
		if !httpguts.ValidHeaderFieldValue(hv) {
			return nil, fmt.Errorf("invalid header value %s", hv)
		}
	}
	res.nonce = riot.Nonce
	res.Status = riot.Status
	res.Body = riot.Body
	res.Close = shouldClose(1, 1, res.Header)

	return res, nil
}

// 应该关闭，判断请求协议是否支持长连接
func shouldClose(major, minor int, header Header) bool {
	if major < 1 {
		return true
	}
	conv := header["Connection"]
	hasClose := conv == "close"
	if major == 1 && minor == 0 {
		return hasClose || conv != "keep-alive"
	}
	return hasClose
}

var (
	bufioReaderPool   sync.Pool
	bufioWriter2kPool sync.Pool
	bufioWriter4kPool sync.Pool
)

// 提取读取缓冲
func newBufioReader(r io.Reader) *bufio.Reader {
	if v := bufioReaderPool.Get(); v != nil {
		br := v.(*bufio.Reader)
		br.Reset(r)
		return br
	}
	return bufio.NewReader(r)
}

// 回收读取缓冲
func putBufioReader(br *bufio.Reader) {
	br.Reset(nil)
	bufioReaderPool.Put(br)
}

// 分配写入缓冲
func bufioWriterPool(size int) *sync.Pool {
	switch size {
	case 2 << 10:
		return &bufioWriter2kPool
	case 4 << 10:
		return &bufioWriter4kPool
	}
	return nil
}

// 回收写入缓冲
func putBufioWriter(bw *bufio.Writer) {
	bw.Reset(nil)
	if pool := bufioWriterPool(bw.Available()); pool != nil {
		pool.Put(bw)
	}
}

// 提取写入缓冲
func newBufioWriterSize(w io.Writer, size int) *bufio.Writer {
	pool := bufioWriterPool(size)
	if pool != nil {
		if v := pool.Get(); v != nil {
			bw := v.(*bufio.Writer)
			bw.Reset(w)
			return bw
		}
	}
	return bufio.NewWriterSize(w, size)
}

// 判断状态码
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == 204:
		return false
	case status == 304:
		return false
	}
	return true
}

// 判断toKen
func hasToken(v, token string) bool {
	if len(token) > len(v) || token == "" {
		return false
	}
	if v == token {
		return true
	}
	for sp := 0; sp <= len(v)-len(token); sp++ {
		// Check that first character is good.
		// The token is ASCII, so checking only a single byte
		// is sufficient. We skip this potential starting
		// position if both the first byte and its potential
		// ASCII uppercase equivalent (b|0x20) don't match.
		// False positives ('^' => '~') are caught by EqualFold.
		if b := v[sp]; b != token[0] && b|0x20 != token[0] {
			continue
		}
		// Check that start pos is on a valid token boundary.
		if sp > 0 && !isTokenBoundary(v[sp-1]) {
			continue
		}
		// Check that end pos is on a valid token boundary.
		if endPos := sp + len(token); endPos != len(v) && !isTokenBoundary(v[endPos]) {
			continue
		}
		if strings.EqualFold(v[sp:sp+len(token)], token) {
			return true
		}
	}
	return false
}

// 是无效符号
func isTokenBoundary(b byte) bool {
	return b == ' ' || b == ',' || b == '\t'
}

// 解析基本验证
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

// 设置基本验证
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// 生成编号
//	nonce string	编号
//	err error		错误
func Nonce() (nonce string, err error) {
	// 创建编号
	bigInt, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	if err != nil {
		return "", fmt.Errorf("create nonce numbering failed %v", err)
	}
	// 提取编号
	d, err := bigInt.MarshalText()
	if err != nil {
		return "", fmt.Errorf("extract nonce numbering failed %v", err)
	}
	return string(d), nil
}

// 是网络读取失败
func isCommonNetReadError(err error) bool {
	if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
		// 网络失败,不要回复
		return true
	}
	if oe, ok := err.(*net.OpError); ok && oe.Op == "read" {
		// 网络失败,不要回复
		return true
	}
	// 读取错误或者被劫持连接
	if err == io.EOF || err == io.ErrUnexpectedEOF || err == ErrHijacked {
		// 读取失败，不要回复
		return true
	}
	return false
}

// 快速设置错误
//	w ResponseWriter	响应
//	err string			错误字符串
//	code int			错误代码
func Error(w ResponseWriter, err string, code int) {
	w.Status(code)
	w.Header().Set("Connection", "close")
	w.SetBody(err)
}

// derogatoryDomain 贬域名
//	host string             host地址
//	f func(string) bool     调用 f 函数，并传入贬域名
func derogatoryDomain(host string, f func(string) bool) {
	// 先全字匹配
	if f(host) {
		return
	}
	// 后通配符匹配
	pos := strings.Index(host, ":")
	var port string
	if pos >= 0 {
		port = host[pos:]
		host = host[:pos]
	}
	labels := strings.Split(host, ".")
	for i := range labels {
		labels[i] = "*"
		candidate := strings.Join(labels, ".") + port
		if f(candidate) {
			break
		}
	}
}

func delay(wait, maxDelay time.Duration) time.Duration {
	if wait == 0 {
		wait = (maxDelay / 100)
	} else {
		wait *= 2
	}
	if wait >= maxDelay {
		wait = maxDelay
	}
	time.Sleep(wait)
	return wait
}

// 给定"host", "host:port"或"[IPv6::address]:port",如果字符串包含端口,则返回true｡
func hasPort(s string) bool { return strings.LastIndex(s, ":") > strings.LastIndex(s, "]") }

// 将":port" 到 "" 以"rfc 3986"第6.2.3节第6.2.3节中的空端口｡
func removeEmptyPort(host string) string {
	if hasPort(host) {
		return strings.TrimSuffix(host, ":")
	}
	return host
}
