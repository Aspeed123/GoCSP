package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"gocsp/internal/logger"
)

func main() {
    runs := flag.Int("r", 1, "number of runs")
    // diagram := flag.String("f", "", "path to diagram JSON file")
    flag.Parse()

    for i := 0; i < *runs; i++ {
        // RuntimeProcess(*diagram)

        data, _ := os.ReadFile("events.jsonl")

        var start, finish time.Time

        lines := bytes.Split(data, []byte("\n"))

        for _, line := range lines {
            if len(line) == 0 {
                continue
            }

            var e logger.Event
            json.Unmarshal(line, &e)
            start = e.Time
            break
        }

        for _, line := range lines {
            if len(line) == 0 {
                continue
            }

            var e logger.Event
            json.Unmarshal(line, &e)

            if e.Event == "node_stop" {
                break
            }

            finish = e.Time
        }

        fmt.Println("Execution time:", finish.Sub(start))
    }
    
}

func RuntimeProcess(diagram string) {
    cmd := exec.Command("go", "run", "./cmd/gocsp", diagram)

    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Start(); err != nil {
        panic(err)
    }

    time.Sleep(2 * time.Second)

    cmd.Process.Signal(os.Interrupt)
}