package runtime

type RuntimeNode struct {
    ID   string
    Type string

    Inputs  map[string]chan any
    Outputs map[string]chan any
}