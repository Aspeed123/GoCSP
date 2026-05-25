package runtime

import (
	"context"

	"gocsp/internal/logger"
	"gocsp/internal/model"
)

func runSwitch(appCtx context.Context, node model.Node, procCtx *ProcessContext) {
	defer func() {
		logger.Log(logger.Event{
			Event: "node_stop",
			Node:  node.ID,
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
	}()

	inChan := procCtx.Inputs["in"]
	condChan := procCtx.Inputs["cond"]

	trueOutputs := procCtx.Outputs["true"]
	falseOutputs := procCtx.Outputs["false"]

	for {
		select {
		case <-appCtx.Done():
			return
		case data, ok := <-inChan:
			if !ok {
				return
			}

            logger.Log(logger.Event{
				Event: "receive",
				Node:  node.ID,
				Port:  "in",
				Value: data,
			})

			select {
			case <-appCtx.Done():
				return
			case condVal, condOk := <-condChan:
				if !condOk {
					return
				}

                logger.Log(logger.Event{
					Event: "receive",
					Node:  node.ID,
					Port:  "cond",
					Value: condVal,
				})

				isTrue, _ := condVal.(bool)

				targetOutputs := falseOutputs
				port := "false"
				if isTrue {
					targetOutputs = trueOutputs
					port = "true"
				}

				for _, ch := range targetOutputs {
					select {
					case ch <- data:
						logger.Log(logger.Event{
							Event: "send",
							Node:  node.ID,
							Port:  port,
							Value: data,
						})
					case <-appCtx.Done():
						return
					}
				}
			}
		}
	}
}
