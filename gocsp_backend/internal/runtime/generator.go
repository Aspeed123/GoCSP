package runtime

import (
	"context"

	"gocsp/internal/logger"
	"gocsp/internal/model"
)

func runGenerator(appCtx context.Context, node model.Node, procCtx *ProcessContext) {
	var valuesToSend []any
	
	if len(node.Data) > 0 {
		expanded, err :=
			expandGeneratorData(node.Data)

		if err != nil {

			logger.Log(logger.Event{
				Event: "error",
				Node:  node.ID,
				Value: err.Error(),
			})

			return
		}

		valuesToSend = expanded
		
	} else {
		valuesToSend = []any{1, 2, 3, 4, 5}
	}

outer:
	
	for port, outputs := range procCtx.Outputs {
		
		for _, val := range valuesToSend {

			select {
			case <-appCtx.Done():
				break outer
			default:
			}
			
			logger.Log(logger.Event{
				Event: "send",
				Node: node.ID,
				Port: port,
				Value: val,
			})
			
			for _, out := range outputs {
				select {
				case out <- val:
				case <-appCtx.Done():
					break outer
				}
			}
		}
	}

	<-appCtx.Done()
	
	logger.Log(logger.Event{
		Event: "node_stop",
		Node: node.ID,
	})

	for port, outputs := range procCtx.Outputs {
		logger.Log(logger.Event{
			Event: "port_closed",
			Node:  node.ID,
			Port:  port,
		})

		for _, out := range outputs {
			close(out)
		}
	}
}