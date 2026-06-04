package runtime 

import ( 
	"sync"
	"strings"
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

				case "merge":
					runMerge(appCtx, n, contexts[n.ID])
					
			}

		}(node)
	}
	
	wg.Wait()
} 

func splitPort(path string) (string, string) {
	
	parts := strings.Split(path, ".")
	
	return parts[0], parts[1]
}