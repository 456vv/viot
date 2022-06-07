package viot

import (
	"testing"
	//	"context"
	//	"net/http"
	"bufio"
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/issue9/assert/v2"
)

func Test_dataWriter_generateResponse(t *testing.T) {
	tests := []struct {
		SetKeepAlivesEnabled       bool
		SetKeepAlivesEnabledString string

		nonce string

		ProtoMajor int
		ProtoMinor int

		closeAfterReply bool
		header          Header
	}{
		{SetKeepAlivesEnabled: true, SetKeepAlivesEnabledString: "keep-alive", nonce: "1", ProtoMajor: 1, ProtoMinor: 1, closeAfterReply: false, header: Header{"Connection": "keep-alive"}},
		{SetKeepAlivesEnabled: true, SetKeepAlivesEnabledString: "close", nonce: "5", ProtoMajor: 1, ProtoMinor: 1, closeAfterReply: true, header: Header{"Connection": "close"}},
		{SetKeepAlivesEnabled: true, SetKeepAlivesEnabledString: "close", nonce: "5", ProtoMajor: 1, ProtoMinor: 0, closeAfterReply: true, header: Header{"Connection": "keep-alive"}},
		{SetKeepAlivesEnabled: true, SetKeepAlivesEnabledString: "close", nonce: "5", ProtoMajor: 1, ProtoMinor: 0, closeAfterReply: true, header: Header{"Connection": "close"}},
		{SetKeepAlivesEnabled: false, SetKeepAlivesEnabledString: "close", nonce: "2", ProtoMajor: 1, ProtoMinor: 0, closeAfterReply: true, header: Header{"Connection": "close"}},
		{SetKeepAlivesEnabled: false, SetKeepAlivesEnabledString: "close", nonce: "2", ProtoMajor: 1, ProtoMinor: 0, closeAfterReply: true, header: Header{"Connection": "keep-alive"}},
		{SetKeepAlivesEnabled: false, SetKeepAlivesEnabledString: "close", nonce: "2", ProtoMajor: 1, ProtoMinor: 1, closeAfterReply: true, header: Header{"Connection": "close"}},
		{SetKeepAlivesEnabled: false, SetKeepAlivesEnabledString: "close", nonce: "2", ProtoMajor: 1, ProtoMinor: 1, closeAfterReply: true, header: Header{"Connection": "keep-alive"}},
	}
	for index, test := range tests {
		cw := &dataWriter{
			res: &responseWrite{
				conn: &conn{
					server: &Server{},
				},
				req: &Request{
					nonce:      test.nonce,
					ProtoMajor: test.ProtoMajor,
					ProtoMinor: test.ProtoMinor,
					Header:     test.header,
				},
			},
		}
		cw.res.conn.server.SetKeepAlivesEnabled(test.SetKeepAlivesEnabled)
		cw.generateResponse()

		if test.SetKeepAlivesEnabled {
			conn := cw.resp.Header.Get("Connection")
			if conn != test.SetKeepAlivesEnabledString {
				t.Fatalf("%v, 预测 %v, 错误 %v", index, test.SetKeepAlivesEnabledString, conn)
			}
		}
		if test.closeAfterReply != cw.res.closeAfterReply {
			t.Fatalf("%v, 预测 %v, 错误 %v", index, test.closeAfterReply, cw.res.closeAfterReply)
		}
		if !reflect.DeepEqual(test.nonce, cw.resp.nonce) {
			t.Fatalf("%v, 预测 %v, 错误 %v", index, test.nonce, cw.resp.nonce)
		}
	}
}

func Test_dataWriter_Body(t *testing.T) {
	as := assert.New(t, true)
	bytesBuffer := bytes.NewBuffer(nil)
	cw := &dataWriter{
		res: &responseWrite{
			conn: &conn{
				server: &Server{},
				bufw:   bufio.NewWriter(bytesBuffer),
				parser: &defaultParse{},
			},
			req: &Request{},
		},
	}
	cw.setBody([1]int{1})
	err := cw.done()
	as.NotError(err)

	var rc ResponseConfig
	err = json.NewDecoder(bytesBuffer).Decode(&rc)
	as.NotError(err)
	as.Equal(rc.Nonce, "")
	as.NotEqual(rc.Header, nil)
	as.Equal(rc.Status, 200)
	as.NotEqual(rc.Body, nil)
}

func Test_dataWriter_done(t *testing.T) {
	as := assert.New(t, true)
	bytesBuffer := bytes.NewBuffer(nil)
	cw := &dataWriter{
		res: &responseWrite{
			conn: &conn{
				server: &Server{},
				bufw:   bufio.NewWriter(bytesBuffer),
				parser: &defaultParse{},
			},
			req: &Request{},
		},
	}

	err := cw.done()
	as.NotError(err)

	var rc ResponseConfig
	err = json.NewDecoder(bytesBuffer).Decode(&rc)
	as.NotError(err)
	as.Equal(rc.Nonce, "")
	as.NotEqual(rc.Header, nil)
	as.Equal(rc.Status, 200)
	as.Equal(rc.Body, nil)
}
