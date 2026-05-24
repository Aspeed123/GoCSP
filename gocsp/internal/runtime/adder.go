package runtime

import (
    "fmt"

    "gocsp/internal/logger"
    "gocsp/internal/model"
)

func runAdder(
    node model.Node,
    ctx *ProcessContext,
) {

    left := ctx.Inputs["left"]
    right := ctx.Inputs["right"]

    outputs := ctx.Outputs["out"]

    for {

        leftValue, ok := <-left

        if !ok {
            break
        }

        rightValue, ok := <-right

        if !ok {
            break
        }

        a := leftValue.(int)
        b := rightValue.(int)

        result := a + b

        logger.Log(logger.Event{
            Event: "receive",
            Node:  node.ID,
            Port:  "left",
            Value: a,
        })

        logger.Log(logger.Event{
            Event: "receive",
            Node:  node.ID,
            Port:  "right",
            Value: b,
        })

        logger.Log(logger.Event{
            Event: "send",
            Node:  node.ID,
            Port:  "out",
            Value: result,
        })

        for _, out := range outputs {
            out <- result
        }
    }

    for _, out := range outputs {
        close(out)
    }

    fmt.Println("Adder stopped")
}