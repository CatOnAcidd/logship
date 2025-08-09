package forward

import "context"

func Start(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
