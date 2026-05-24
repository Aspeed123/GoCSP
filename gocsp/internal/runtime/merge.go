package runtime

import (
    "reflect"

    "gocsp/internal/logger"
    "gocsp/internal/model"
)

func runMerge(
    node model.Node,
    ctx *ProcessContext,
) {

    outputs := ctx.Outputs["out"]

    var cases []reflect.SelectCase

    var portNames []string

    for portName, ch := range ctx.Inputs {

        cases = append(cases, reflect.SelectCase{
            Dir:  reflect.SelectRecv,
            Chan: reflect.ValueOf(ch),
        })

        portNames = append(portNames, portName)
    }

    activeInputs := len(cases)

    for activeInputs > 0 {

        chosen, value, ok := reflect.Select(cases)

        if !ok {

            cases[chosen].Chan = reflect.Value{}

            activeInputs--

            continue
        }

        port := portNames[chosen]

        logger.Log(logger.Event{
            Event: "receive",
            Node:  node.ID,
            Port:  port,
            Value: value.Interface(),
        })

        logger.Log(logger.Event{
            Event: "send",
            Node:  node.ID,
            Port:  "out",
            Value: value.Interface(),
        })

        for _, out := range outputs {
            out <- value.Interface()
        }
    }

    for _, out := range outputs {
        close(out)
    }
}