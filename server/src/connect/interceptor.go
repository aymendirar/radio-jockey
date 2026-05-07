package connect

import (
	"context"
	"errors"
	"log/slog"
	"path"
	"reflect"
	"strings"

	"server/src/connect/auth"
	"server/src/proto/protoconnect"

	"connectrpc.com/connect"
)

var AuthenticatedProcedures = []string{
	protoconnect.RadioServiceDeleteSessionAuthProcedure,
}

func authInterceptor(a *auth.Auth) connect.Interceptor {
	set := make(map[string]struct{}, len(AuthenticatedProcedures))
	for _, p := range AuthenticatedProcedures {
		set[p] = struct{}{}
	}
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if _, ok := set[req.Spec().Procedure]; ok {
				token := strings.TrimPrefix(req.Header().Get("authorization"), "Bearer ")
				if err := a.VerifyToken(token); err != nil {
					return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated"))
				}
			}
			return next(ctx, req)
		}
	})
}

func stripInterceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			stripStrings(reflect.ValueOf(req.Any()))
			return next(ctx, req)
		}
	})
}

func stripStrings(v reflect.Value) {
	if !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			stripStrings(v.Elem())
		}
	case reflect.Struct:
		for i := range v.NumField() {
			stripStrings(v.Field(i))
		}
	case reflect.String:
		if v.CanSet() {
			v.SetString(strings.TrimSpace(v.String()))
		}
	}
}

func loggingInterceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			rpc := path.Base(req.Spec().Procedure)
			slog.Info("RPC received", "rpc", rpc, "request", req.Any())
			resp, err := next(ctx, req)
			if err != nil {
				slog.Error("RPC error", "rpc", rpc, "err", err)
			} else {
				slog.Info("RPC completed", "rpc", rpc, "response", resp.Any())
			}
			return resp, err
		}
	})
}
