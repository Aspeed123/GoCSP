package logger

import (
    "encoding/json"
    "time"
    "sync/atomic"
    "os"
    "sync"
)

var eventCounter int64

var logFile *os.File

var wg sync.WaitGroup

var logChannel = make(chan Event, 10000)

type Event struct {
    ID    int64  `json:"id"`
    Time  time.Time `json:"time"`
    Event string `json:"event"`
    Node  string `json:"node"`
    Port  string `json:"port,omitempty"`
    Value any    `json:"value,omitempty"`
}

func Log(event Event) {
    event.Time = time.Now()

    event.ID = atomic.AddInt64(&eventCounter, 1)

    logChannel <- event
}

func Run() {
    wg.Add(1)

    go func() {
        defer wg.Done()

        for event := range logChannel {
            data, _ := json.Marshal(event)

            if logFile != nil {
                logFile.Write(data)
                logFile.Write([]byte("\n"))
            }
        }
    }()
}

func Shutdown() {
	close(logChannel)

	wg.Wait()
}

func InitLogFile(path string) error {

    file, err := os.Create(path)

    if err != nil {
        return err
    }

    logFile = file

    return nil
}

func CloseLogFile() {

    if logFile != nil {
        logFile.Close()
    }
}