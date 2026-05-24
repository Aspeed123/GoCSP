package runtime 

import ( 
	"fmt"
	"sync"
	"strings"
	"reflect"
	"context"


	"gocsp/internal/logger"
	"gocsp/internal/model"
)

func Run(appCtx context.Context, diagram *model.Diagram) {

	contexts := make(map[string]*ProcessContext)

	for _, node := range diagram.Nodes {
		
		contexts[node.ID] = &ProcessContext{
			Inputs: make(map[string]chan any),
			Outputs: make(map[string][]chan any),
		}
	}
	
	for _, edge := range diagram.Edges {

		fromNode, fromPort := splitPort(edge.From)

		toNode, toPort := splitPort(edge.To)
		
		ch := make(chan any, 64)

		if edge.Initial != nil {
			ch <- edge.Initial

			logger.Log(logger.Event{
                Event: "initial_token",
                Node:  toNode,
                Port:  toPort,
                Value: edge.Initial,
            })
		}
		
		contexts[fromNode].Outputs[fromPort] =
			append(
				contexts[fromNode].Outputs[fromPort],
				ch,
			)
			
		contexts[toNode].Inputs[toPort] = ch
	}
	
	var wg sync.WaitGroup
	
	for _, node := range diagram.Nodes {
		
		wg.Add(1)
		
		go func(n model.Node) {
			
			defer wg.Done()
			
			logger.Log(logger.Event{ 
				Event: "goroutine_start",
				Node: n.ID,
			})
			
			switch n.Type {
				
				case "generator":
					runGenerator(appCtx, n, contexts[n.ID])
					
				case "processor":
					runProcessor(appCtx, n, contexts[n.ID])
					
				case "sink":
					runSink(appCtx, n, contexts[n.ID])

				case "switch":
					runSwitch(appCtx, n, contexts[n.ID])
					
			}

		}(node)
	}
	
	wg.Wait()
} 

func runGenerator(
	appCtx context.Context,
	node model.Node,
	ctx *ProcessContext,
) { 
	
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
	
	for port, outputs := range ctx.Outputs {
		
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

	for port, outputs := range ctx.Outputs {
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

func runProcessor(
	appCtx context.Context,
	node model.Node,
	ctx *ProcessContext,
) { 
	runUniversalProcessor(appCtx, node, ctx) 
}

func runSink(
	appCtx context.Context,
	node model.Node,
	ctx *ProcessContext,
) {
	
	var cases []reflect.SelectCase

	cases = append(cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(appCtx.Done()),
	})

	var ports []string
	
	for _, port := range node.Inputs {
		
		ch := ctx.Inputs[port]
		
		cases = append(cases, reflect.SelectCase{
			Dir: reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		})
		
		ports = append(ports, port)
	}
	
	activeInputs := len(ports)
	
	for activeInputs > 0 {
		
		chosen, value, ok := reflect.Select(cases)

		if chosen == 0 {
			break
		}
		
		port := ports[chosen-1]
		
		if !ok {
			logger.Log(logger.Event{
				Event: "port_closed",
				Node: node.ID,
				Port: port,
			})
			
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
	
	logger.Log(logger.Event{
		Event: "node_stop",
		Node: node.ID,
	})
} 

func splitPort(path string) (string, string) {
	
	parts := strings.Split(path, ".")
	
	return parts[0], parts[1]
}