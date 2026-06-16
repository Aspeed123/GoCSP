package runtime

type ProcessContext struct {
    Inputs  map[string]chan any
    Outputs map[string][]chan any
}