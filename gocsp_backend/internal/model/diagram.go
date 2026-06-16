package model

type Diagram struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Node struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Operation string `json:"operation,omitempty"`

	Inputs  []string `json:"inputs"`
	Outputs []string `json:"outputs"`
	Data    []any    `json:"data,omitempty"`
}

type Edge struct {
	From 	string `json:"from"`
	To   	string `json:"to"`
	Initial any    `json:"initial,omitempty"`
}
