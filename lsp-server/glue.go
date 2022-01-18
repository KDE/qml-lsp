package lspserver

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"

	"github.com/sourcegraph/jsonrpc2"
)

type Method func(ctx context.Context, conn jsonrpc2.JSONRPC2, params json.RawMessage) interface{}
type MethodMap map[string]Method
type stdrwc struct{}

func (stdrwc) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (stdrwc) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (stdrwc) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}
func Zu(fn interface{}) func(ctx context.Context, conn jsonrpc2.JSONRPC2, params json.RawMessage) interface{} {
	val := reflect.ValueOf(fn)
	in := val.Type().In(2)
	return func(ctx context.Context, conn jsonrpc2.JSONRPC2, params json.RawMessage) interface{} {
		v := reflect.New(in)
		json.Unmarshal(params, v.Interface())
		ret := val.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(conn), v.Elem()})
		switch len(ret) {
		case 0: // notification
			return nil
		case 2: // lsp
			if !ret[0].IsNil() {
				return ret[0].Interface()
			}
			if !ret[1].IsNil() {
				return ret[1].Interface()
			}
			panic("e")
		default:
			panic("unknown arity of return")
		}
	}
}
func StartServer(a MethodMap) {
	han := jsonrpc2.HandlerWithError(func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
		v, ok := a[req.Method]
		if !ok {
			return nil, errors.New("not found")
		}
		resp := v(ctx, conn, *req.Params)

		return resp, nil
	})
	<-jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}), han).DisconnectNotify()
}
