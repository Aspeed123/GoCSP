package runtime 

import ( 
	"fmt"
	"sync"
	"strings"
	"reflect"


	"gocsp/internal/logger"
	"gocsp/internal/model"
)

func Run(diagram *model.Diagram) {

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
					runGenerator(n, contexts[n.ID])
					
				case "processor":
					runProcessor(n, contexts[n.ID])
					
				case "sink":
					runSink(n, contexts[n.ID])
					
			}

		}(node)
	}
	
	wg.Wait()
} 

func runGenerator(
	node model.Node,
	ctx *ProcessContext,
) { 
	
	var valuesToSend []any
	
	if len(node.Data) > 0 {
		valuesToSend = node.Data
	} else {
		valuesToSend = []any{1, 2, 3, 4, 5}
	}
	
	for port, outputs := range ctx.Outputs {
		
		for _, val := range valuesToSend {
			
			logger.Log(logger.Event{
				Event: "send",
				Node: node.ID,
				Port: port,
				Value: val,
			})
			
			for _, out := range outputs {
				out <- val
			}
		}
		
		logger.Log(logger.Event{
			Event: "port_closed",
			Node: node.ID,
			Port: port,
		})
		
		for _, out := range outputs {
			close(out)
		}
	}
	
	logger.Log(logger.Event{
		Event: "node_stop",
		Node: node.ID,
	})
}

func runProcessor(
	node model.Node,
	ctx *ProcessContext,
) { 
	runUniversalProcessor(node, ctx) 
}

func runSink(
	node model.Node,
	ctx *ProcessContext,
) {
	
	var cases []reflect.SelectCase
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
		
		port := ports[chosen]
		
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