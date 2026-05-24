package replay

import (
    "bufio"
    "encoding/json"
    "os"
    "sort"

    "gocsp/internal/logger"
)

func LoadEvents(path string) ([]logger.Event, error) {

    file, err := os.Open(path)

    if err != nil {
        return nil, err
    }

    defer file.Close()

    var events []logger.Event

    scanner := bufio.NewScanner(file)

    for scanner.Scan() {

        line := scanner.Bytes()

        var event logger.Event

        err := json.Unmarshal(line, &event)

        if err != nil {
            return nil, err
        }

        events = append(events, event)
    }

    sort.Slice(events, func(i, j int) bool {
        return events[i].ID < events[j].ID
    })

    return events, nil
}