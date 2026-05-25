package logger

import (
    "encoding/json"
    "fmt"
    "time"
    "sync/atomic"
    "os"
    "sync"
)

var eventCounter int64

var logFile *os.File

var logMutex sync.Mutex

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
    go func() {
        for event := range logChannel {
            data, _ := json.Marshal(event)

            fmt.Println(string(data))

            if logFile != nil {
                logFile.Write(data)
                logFile.Write([]byte("\n"))
            }
        }
    }()
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