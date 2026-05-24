package runtime

import "context"

type Runtime struct {
	Context context.Context
	Cancel  context.CancelFunc
}