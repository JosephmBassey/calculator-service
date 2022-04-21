package grpcdep

import (
	"context"
	"fmt"

	"google.golang.org/grpc/connectivity"
)

type StateProvider interface {
	GetState() connectivity.State
}

type Wrapper struct {
	name string
	conn StateProvider
}

func Wrap(name string, conn StateProvider) *Wrapper {
	return &Wrapper{name, conn}
}

func (w *Wrapper) String() string {
	return w.name
}

func (w *Wrapper) Status(ctx context.Context) error {
	cs := w.conn.GetState()
	if cs == connectivity.Ready {
		return nil
	}
	return fmt.Errorf("connection not 'Ready': %d", cs)
}
