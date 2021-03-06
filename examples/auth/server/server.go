package main

import (
	"context"
	"errors"

	"time"

	"github.com/lubanproj/gorpc"
	"github.com/lubanproj/gorpc/auth"
	"github.com/lubanproj/gorpc/log"
	"github.com/lubanproj/gorpc/metadata"
	"github.com/lubanproj/gorpc/testdata"
)


func main() {

	// AuthFunc 验证令牌是否有效
	af := func(ctx context.Context) (context.Context, error){
		//取出 metadata
		md := metadata.ServerMetadata(ctx)

		if len(md) == 0 {
			return ctx, errors.New("token nil")
		}
		v := md["authorization"]
		log.Debug("token : ", string(v))
		if string(v) != "Bearer testToken" {
			return ctx, errors.New("token invalid")
		}
		return ctx, nil
	}

	opts := []gorpc.ServerOption{
		gorpc.WithAddress("127.0.0.1:8003"),
		gorpc.WithNetwork("tcp"),
		gorpc.WithSerializationType("msgpack"),
		gorpc.WithTimeout(time.Millisecond * 2000),
		gorpc.WithInterceptor(auth.BuildAuthInterceptor(af)),
	}
	s := gorpc.NewServer(opts ...)
	if err := s.RegisterService("/helloworld.Greeter", new(testdata.Service)); err != nil {
		panic(err)
	}
	s.Serve()
}