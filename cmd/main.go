package main

import (
	"net/http"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_kit "github.com/grpc-ecosystem/go-grpc-middleware/logging/kit"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"

	"go.opencensus.io/plugin/ocgrpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/greyfinance/grey-go-libs/http/handler/version"
	"github.com/greyfinance/grey-go-libs/http/httputils"
	"github.com/greyfinance/grey-go-libs/log/levelfilter"

	"github.com/josephmbassey/calculator-service/config"
	"github.com/josephmbassey/calculator-service/internals/http/starter"
	"github.com/josephmbassey/calculator-service/rpc/proto/calculatorpb"
	"github.com/josephmbassey/calculator-service/services/calculatorservice"
)

func main() {

	// =========================================================================
	// Initialize Config, and tools ...
	cfg, err := config.New()
	if err != nil {
		level.Error(log.NewLogfmtLogger(os.Stdout)).Log("msg", "config err", "err", err)
		os.Exit(1)
	}
	logger := initLogger(cfg)

	// =========================================================================
	// Initialize Services
	calculatorSvc, err := calculatorservice.NewService(logger)

	// =========================================================================

	// For Kubernetes to handle graceful shutdown
	livenessServer := httputils.NewServerWithDefaultTimeouts(logger)
	livenessServer.Handler = http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("liveliness probe ok"))
	})
	livenessServer.Addr = cfg.ListenHTTPLiveness

	// gRPC server
	grpcOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_prometheus.UnaryServerInterceptor,
			grpc_kit.UnaryServerInterceptor(logger, grpc_kit.WithLevels(grpc_kit.DefaultClientCodeToLevel)),
		)),
		grpc.StatsHandler(&ocgrpc.ServerHandler{IsPublicEndpoint: false}),
	}
	grpcServer := grpc.NewServer(grpcOpts...)

	calculatorpb.RegisterCalculatorServiceServer(grpcServer, calculatorservice.NewGRPCHandler(calculatorSvc))

	reflection.Register(grpcServer)
	grpc_prometheus.Register(grpcServer)

	// start servers
	servers := starter.New().
		WithGRPC(grpcServer, cfg.ListenGRPC, nil)
	servers.Log = logger
	if err := servers.RunUntilInterrupt(); err != nil {
		level.Error(logger).Log("msg", "failed to start HTTP/gRPC servers", "err", err)
		os.Exit(1)
	}
}

func initLogger(cfg config.Config) log.Logger {
	logger := log.NewLogfmtLogger(os.Stderr)
	if cfg.Environment == "production" || cfg.Environment == "staging" || os.Getenv("LOGFMT") == "json" {
		logger = log.NewJSONLogger(os.Stdout)
	}
	logger = levelfilter.FromEnv(logger)
	logger = log.With(logger,
		"service", cfg.SERVICE_NAME,
		"environment", cfg.Environment,
		"ts", log.DefaultTimestampUTC,
		"caller", log.DefaultCaller,
		"commit", version.Commit,
	)
	return logger
}
