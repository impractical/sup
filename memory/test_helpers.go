package memory

import (
	"context"

	"impractical.co/sup"
)

type Factory struct{}

func (f Factory) NewStorer(ctx context.Context) (sup.Storer, error) {
	return NewStorer()
}

func (f Factory) TeardownStorers() error {
	return nil
}
