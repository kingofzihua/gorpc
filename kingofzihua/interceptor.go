package main

import (
	"context"
	"fmt"
)

type interceptor func(ctx context.Context, h handler)

type interceptor2 func(ctx context.Context, h handler, ivk invoker) error

type handler func(ctx context.Context)

type invoker func(ctx context.Context, interceptors []interceptor2, h handler) error

// 递归调用
func getInvoker(ctx context.Context, interceptors []interceptor2, cur int, ivk invoker) invoker {
	//停止条件
	if cur == len(interceptors)-1 {
		return ivk
	}

	return func(ctx context.Context, interceptors []interceptor2, h handler) error {
		return interceptors[cur+1](ctx, h, getInvoker(ctx, interceptors, cur+1, ivk))
	}
}

func getChainInterceptor(ctx context.Context, interceptors []interceptor2, ivk invoker) interceptor2 {
	if len(interceptors) == 0 {
		return nil
	}
	if len(interceptors) == 1 {
		return interceptors[0]
	}

	return func(ctx context.Context, h handler, ivk invoker) error {
		return interceptors[0](ctx, h, getInvoker(ctx, interceptors, 0, ivk))
	}
}

func main() {
	var (
		ctx  context.Context
		ceps []interceptor2
		h    = func(ctx context.Context) {
			fmt.Println("handler do somethin ...")
		}
		inter1 = func(ctx context.Context, h handler, ivk invoker) error {
			fmt.Println("interceptor1 start")
			//h(ctx)
			fmt.Println("interceptor1 end")
			return ivk(ctx, ceps, h)
		}
		inter2 = func(ctx context.Context, h handler, ivk invoker) error {
			fmt.Println("interceptor2 start")
			//h(ctx)
			fmt.Println("interceptor2 end")
			return ivk(ctx, ceps, h)
		}
		inter3 = func(ctx context.Context, h handler, ivk invoker) error {
			fmt.Println("interceptor3 start")
			//h(ctx)
			fmt.Println("interceptor3 end")
			return ivk(ctx, ceps, h)
		}
	)
	ceps = append(ceps, inter1, inter2, inter3)

	var ivk = func(ctx context.Context, interceptors []interceptor2, handler2 handler) error {
		fmt.Println("do vava")
		return nil
	}

	cep := getChainInterceptor(ctx, ceps, ivk)

	if err := cep(ctx, h, ivk); err != nil {
		fmt.Println(err)
	}
}
