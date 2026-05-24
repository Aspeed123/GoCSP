package runtime

import (
	"reflect"

	"github.com/Knetic/govaluate"

	"gocsp/internal/logger"
	"gocsp/internal/model"
)

func runUniversalProcessor(
	node model.Node,
	ctx *ProcessContext,
) {

	expression, err :=
		govaluate.NewEvaluableExpression(
			node.Operation,
		)

	if err != nil {

		logger.Log(logger.Event{
			Event: "error",
			Node:  node.ID,
			Value: err.Error(),
		})

		return
	}

	var cases []reflect.SelectCase
	var ports []string

	for _, port := range node.Inputs {

		ch := ctx.Inputs[port]

		cases = append(cases,
			reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(ch),
			},
		)

		ports = append(ports, port)
	}

	buffers := make(map[string][]any)
	closedPorts := make(map[string]bool)

	activeInputs := len(ports)

	for activeInputs > 0 {

		chosen, value, ok := reflect.Select(cases)

		port := ports[chosen]

		if !ok {

			closedPorts[port] = true

			cases[chosen].Chan = reflect.Value{}

			activeInputs--

			logger.Log(logger.Event{
				Event: "port_closed",
				Node: node.ID,
				Port: port,
			})
			
			continue
		}


		receivedValue := value.Interface()

		logger.Log(logger.Event{
			Event: "receive",
			Node:  node.ID,
			Port:  port,
			Value: receivedValue,
		})

		buffers[port] =
			append(
				buffers[port],
				receivedValue,
			)

		if allPortsReady(
			buffers,
			ports,
		) {

			parameters := make(map[string]any)

			for _, port := range ports {

				parameters[port] =
					buffers[port][0]

				buffers[port] =
					buffers[port][1:]
			}

			result, err :=
				expression.Evaluate(
					parameters,
				)

			if err != nil {

				logger.Log(logger.Event{
					Event: "error",
					Node:  node.ID,
					Value: err.Error(),
				})

				continue
			}

			if f, ok := result.(float64); ok {

				if f == float64(int(f)) {
					result = int(f)
				}
			}

			for port, outputs := range ctx.Outputs {

				logger.Log(logger.Event{
					Event: "send",
					Node:  node.ID,
					Port:  port,
					Value: result,
				})

				for _, out := range outputs {
					out <- result
				}
			}
		}
	}

	logger.Log(logger.Event{
		Event: "node_stop",
		Node: node.ID,
	})

	for port, outputs := range ctx.Outputs {
		logger.Log(logger.Event{
			Event: "port_closed",
			Node: node.ID,
			Port: port,
		})
		for _, out := range outputs {
			close(out)
		}
	}
}

func allPortsReady(
	buffers map[string][]any,
	ports []string,
) bool {

	for _, port := range ports {

		if len(buffers[port]) == 0 {
			return false
		}
	}

	return true
}