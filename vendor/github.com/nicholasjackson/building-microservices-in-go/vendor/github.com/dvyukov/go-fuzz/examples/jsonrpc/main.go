// Copyright 2015 Dmitry Vyukov. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package jsonrpc

import (
	"io"
	"net/rpc"
	"net/rpc/jsonrpc"
	"runtime"
)

func init() {
	rpc.Register(Foo(0))
}

type Foo int

type Args struct {
	I int
	S string
	V []string
	X map[string]string
	N *Nested
}

type Nested struct {
	I int
	S string
}

func (Foo) Bar(a *Args, r *int) error {
	return nil
}

func Fuzz(data []byte) int {
	c := &MyConn{data, false}
	jsonrpc.ServeConn(c)
	if !c.closed {
		panic("conn is not closed")
	}
	if runtime.Gosched(); runtime.NumGoroutine() > 100 {
		panic("goroutine leak")
	}
	return 0
}

type MyConn struct {
	data   []byte
	closed bool
}

func (c *MyConn) Read(b []byte) (n int, err error) {
	if len(c.data) == 0 {
		return 0, io.EOF
	}
	n = copy(b, c.data)
	c.data = c.data[n:]
	return
}

func (c *MyConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (c *MyConn) Close() error {
	c.closed = true
	return nil
}
