package main

import (
    "fmt"
    "os"

    "gocsp/internal/parser"
    "gocsp/internal/runtime"
    "gocsp/internal/logger"
)

func main() {

    if len(os.Args) < 2 {
        fmt.Println("usage: gocsp <diagram.json>")
        return
    }

    diagram, err := parser.LoadDiagram(os.Args[1])
    if err != nil {
        panic(err)
    }

    err = logger.InitLogFile("events.jsonl")
    if err != nil {
        panic(err)
    }
    defer logger.CloseLogFile()

    runtime.Run(diagram)
}