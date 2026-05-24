package runtime

import (
	"context"

	"gocsp/internal/logger"
	"gocsp/internal/model"
)

func runSwitch(appCtx context.Context, node model.Node, procCtx *ProcessContext) {
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

            select {
            case <-appCtx.Done():
                return
            case condVal, condOk := <-condChan:
                if !condOk {
                    return
                }

                isTrue, _ := condVal.(bool)

                targetOutputs := falseOutputs
                if isTrue {
                    targetOutputs = trueOutputs
                }

                for _, ch := range targetOutputs {
                    select {
                    case ch <- data:
                        logger.Log(logger.Event{
                            Event: "send",
                            Node:  node.ID,
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