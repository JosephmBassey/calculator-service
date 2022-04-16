package config

import (
	arg "github.com/alexflint/go-arg"
	"github.com/josephmbassey/calculator-service/internals/logger"
	"github.com/pkg/errors"
)

// Config is the config struct
type Config struct {
	logger.LoglevelEnv
	SERVICE_NAME       string `arg:"--service-name,env:SERVICE_NAME"`
	Environment        string `arg:"--environment,env:ENVIRONMENT"`
	ListenHTTP         string `arg:"--listen-http,env:LISTEN_HTTP"`
	ListenGRPC         string `arg:"--listen-grpc,env:LISTEN_GRPC"`
	ListenHTTPLiveness string `arg:"--listen-http-liveness,env:LISTEN_HTTP_LIVENESS"`
}

// New creates a new config struct with sane defaults
func New() (Config, error) {
	c := Config{
		ListenHTTP:         ":8080",
		ListenGRPC:         ":8083",
		ListenHTTPLiveness: ":8084",
	}

	err := errors.Wrap(errors.WithStack(arg.Parse(&c)), "failed to parse config")
	if err != nil {
		return Config{}, err
	}

	return c, nil
}
