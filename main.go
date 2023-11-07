package main

import (
	"context"
	"log/slog"
	"net"
	"runtime/debug"

	"github.com/dusted-go/logging/prettylog"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var _ gnmi.GNMIServer = (*GNMIServer)(nil)

type GNMIServer struct {
	// TODO: collector
	// TODO: full server state
	// TODO: subscription state
	// TODO: coalesce queue per subscriber
}

// Capabilities implements gnmi.GNMIServer.
func (*GNMIServer) Capabilities(context.Context, *gnmi.CapabilityRequest) (*gnmi.CapabilityResponse, error) {
	panic("unimplemented")
}

// Get implements gnmi.GNMIServer.
func (*GNMIServer) Get(context.Context, *gnmi.GetRequest) (*gnmi.GetResponse, error) {
	panic("unimplemented")
}

// Set implements gnmi.GNMIServer.
func (*GNMIServer) Set(context.Context, *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	panic("unimplemented")
}

// Subscribe implements gnmi.GNMIServer.
func (*GNMIServer) Subscribe(gnmi.GNMI_SubscribeServer) error {
	panic("unimplemented")
}

// InterceptorLogger adapts slog logger to interceptor logger.
func InterceptorLogger() logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		slog.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

// PanicHandler is a handler which returns an internal error to the client and logs the panic
func PanicHandler() recovery.RecoveryHandlerFunc {
	return func(p any) (err error) {
		slog.Error("recovered from panic", slog.Any("panic", p), slog.String("stack", string(debug.Stack())))
		return status.Errorf(codes.Internal, "Internal Error")
	}
}

func main() {
	logger := slog.New(prettylog.NewHandler(&slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()
	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(InterceptorLogger()),
			recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(PanicHandler())),
		),
		grpc.ChainStreamInterceptor(
			logging.StreamServerInterceptor(InterceptorLogger()),
			recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(PanicHandler())),
		),
	)
	gnmiSrv := &GNMIServer{}

	gnmi.RegisterGNMIServer(grpcSrv, gnmiSrv)
	reflection.Register(grpcSrv)

	lis, err := net.Listen("tcp", "localhost:9090")
	if err != nil {
		slog.Error("failed to listen", slog.Any("err", err))
		return
	}

	go grpcSrv.Serve(lis)
	defer grpcSrv.Stop()

	slog.Info("Started gRPC server", slog.String("addr", lis.Addr().String()))

	<-ctx.Done()
	if err := ctx.Err(); err != nil {
		slog.Error("stopped", slog.Any("err", err))
		return
	}
}
