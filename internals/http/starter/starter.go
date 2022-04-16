// Package starter is a multi-listener start helper. When starting multiple
// http/https and grpc/grpcs servers including error handling a main function
// quickly becomes unreadable. This package accepts any number of servers and
// tries to start each one
package starter

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"

	"os"
	"os/signal"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type grpcServer struct {
	srv *grpc.Server
	lis string
	tls *tls.Config
}

type Server struct {
	srv     []*http.Server
	gsrv    []grpcServer
	err     chan error
	started bool
	timeout time.Duration
	Log     log.Logger
}

func New() *Server {
	return &Server{
		srv:     make([]*http.Server, 0, 1),
		gsrv:    make([]grpcServer, 0, 1),
		err:     make(chan error, 1),
		started: false,
		timeout: 2 * time.Second,
	}
}

func (h *Server) WithHTTP(srv ...*http.Server) *Server {
	if h.started {
		return h
	}

	for _, s := range srv {
		if s == nil {
			continue
		}
		h.srv = append(h.srv, s)
	}
	return h
}

func (h *Server) WithGRPC(srv *grpc.Server, lis string, tlsCfg *tls.Config) *Server {
	if h.started {
		return h
	}

	if srv == nil {
		return h
	}

	h.gsrv = append(h.gsrv, grpcServer{srv, lis, tlsCfg})
	return h
}

func (h *Server) Start() error {
	if h.started {
		return fmt.Errorf("already started")
	}
	h.started = true

	if cap(h.err) < len(h.srv)+len(h.gsrv) {
		h.err = make(chan error, len(h.srv)+len(h.gsrv))
	}
	if h.Log == nil {
		h.Log = log.NewNopLogger()
	}

	for _, srv := range h.srv {
		go func(s *http.Server, ec chan error) {
			level.Info(h.Log).Log("msg", "starting http server", "addr", s.Addr)
			if s.TLSConfig != nil {
				ec <- s.ListenAndServeTLS("", "")
				return
			}
			ec <- s.ListenAndServe()
		}(srv, h.err)
	}
	for _, srv := range h.gsrv {
		go func(s grpcServer, ec chan error) {
			level.Info(h.Log).Log("msg", "starting grpc server", "addr", s.lis)
			var listener net.Listener
			var err error
			if s.tls != nil {
				listener, err = tls.Listen("tcp", s.lis, s.tls)
			} else {
				listener, err = net.Listen("tcp", s.lis)
			}
			if err != nil {
				ec <- errors.Wrapf(err, "failed to start gRPC server: %s", s.lis)
				return
			}
			ec <- s.srv.Serve(listener)
		}(srv, h.err)
	}

	select {
	case e := <-h.err:
		return e
	case <-time.After(h.timeout):
		return nil
	}
	return nil
}

func (h *Server) RunUntilInterrupt() error {
	if err := h.Start(); err != nil {
		return err
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Block until a signal is received.
	level.Info(h.Log).Log("msg", "startup complete")
	<-c
	level.Info(h.Log).Log("msg", "shutting down")

	return h.Stop()
}

func (h *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	var g errgroup.Group
	for _, srv := range h.srv {
		srv := srv // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			level.Info(h.Log).Log("msg", "stopping http server", "addr", srv.Addr)
			return srv.Shutdown(ctx)
		})
	}
	for _, srv := range h.gsrv {
		srv := srv
		g.Go(func() error {
			level.Info(h.Log).Log("msg", "stopping grpc server", "addr", srv.lis)
			srv.srv.GracefulStop()
			return nil
		})
	}
	return g.Wait()
}
