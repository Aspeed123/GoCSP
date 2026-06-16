package runtime

import (
	"reflect"
	"context"
	"fmt"

	"gocsp/internal/logger"
	"gocsp/internal/model"
)

func runSink(appCtx context.Context, node model.Node, procCtx *ProcessContext) {
	
	var cases []reflect.SelectCase

	cases = append(cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(appCtx.Done()),
	})

	var ports []string
	
	for _, port := range node.Inputs {
		
		ch := procCtx.Inputs[port]
		
		cases = append(cases, reflect.SelectCase{
			Dir: reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		})
		
		ports = append(ports, port)
	}

	closedPorts := make(map[string]bool)
	
	activeInputs := len(ports)

	defer func() {
        logger.Log(logger.Event{
            Event: "node_stop",
            Node:  node.ID,
        })

        for _, port := range ports {
            if !closedPorts[port] {
                logger.Log(logger.Event{
                    Event: "port_closed",
                    Node:  node.ID,
                    Port:  port,
                })
            }
        }

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
	
	for activeInputs > 0 {
		
		chosen, value, ok := reflect.Select(cases)

		if chosen == 0 {
			break
		}
		
		port := ports[chosen-1]
		
		if !ok {
			closedPorts[port] = true

			cases[chosen].Chan = reflect.Value{}
			
			activeInputs--
			
			continue
		}
		
		logger.Log(logger.Event{
			Event: "receive",
			Node: node.ID,
			Port: port,
			Value: value.Interface(),
		})
		
		fmt.Println(
			"RESULT:",
			value.Interface(),
		)
	}
}