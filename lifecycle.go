package fx

import (
	"context"
	"errors"
)

type Hook struct {
	OnStop func(context.Context) error
}

type Lifecycle struct {
	hooks []Hook
}

func (l *Lifecycle) Stop(ctx context.Context) error {
	var errs []error
	for _, h := range l.hooks {
		if err := h.OnStop(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
