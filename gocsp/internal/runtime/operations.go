package runtime

import "fmt"

type OperationFunc func(int) int

var Operations = map[string]OperationFunc{

    "square": func(x int) int {
        return x * x
    },

    "double": func(x int) int {
        return x * 2
    },

    "increment": func(x int) int {
        return x + 1
    },

    "add": func(x int) int {
        return x
    },

    "merge": func(x int) int {
        return x
    },
}

func ExecuteOperation(name string, value int) int {

    op, exists := Operations[name]

    if !exists {
        panic(fmt.Sprintf("unknown operation: %s", name))
    }

    return op(value)
}