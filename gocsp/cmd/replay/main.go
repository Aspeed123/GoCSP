package main

import (
    "fmt"
    "time"

    "gocsp/internal/replay"
)

const ReplaySpeed = 50

func main() {

    events, err :=
        replay.LoadEvents("events.jsonl")

    if err != nil {
        panic(err)
    }

    fmt.Println("REPLAY START")

    for i, event := range events {

        fmt.Printf(
            "[%d] %s %s %v\n",
            event.ID,
            event.Node,
            event.Event,
            event.Value,
        )

        if i < len(events)-1 {
			next := events[i+1]

			delta := next.Time.Sub(event.Time)
			delta = delta * time.Duration(ReplaySpeed)

			if delta < 0 {
				delta = 0
			}

			if delta > 2 * time.Second {
				delta = 2 * time.Second
			}

			time.Sleep(delta)
		}
    }

    fmt.Println("REPLAY END")
}