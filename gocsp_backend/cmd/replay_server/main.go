package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"

	"gocsp/internal/replay"
)

const ReplaySpeed = 2000

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {

	http.HandleFunc("/ws", handleReplay)

	server := &http.Server{
		Addr: ":8080",
	}

	// Запуск сервера
	go func() {

		fmt.Println("Replay server started at :8080")

		err := server.ListenAndServe()

		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Ожидание Ctrl+C
	stop := make(chan os.Signal, 1)

	signal.Notify(
		stop,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-stop

	fmt.Println("\nShutting down server...")

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			5*time.Second,
		)

	defer cancel()

	err := server.Shutdown(ctx)

	if err != nil {
		log.Println(err)
	}

	fmt.Println("Server stopped")
}

func handleReplay(
	w http.ResponseWriter,
	r *http.Request,
) {

	conn, err :=
		upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
		return
	}

	defer func() {
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(
				websocket.CloseNormalClosure,
				"server shutdown",
			),
			time.Now().Add(time.Second),
		)

		_ = conn.Close()
	}()

	events, err :=
		replay.LoadEvents("events.jsonl")

	if err != nil {

		log.Println(err)

		return
	}

	for i, event := range events {

		data, _ := json.Marshal(event)

		err := conn.WriteMessage(
			websocket.TextMessage,
			data,
		)

		if err != nil {
			log.Println(err)
			return
		}

		fmt.Println("SENT:", string(data))

		if i < len(events)-1 {

			next := events[i+1]

			delta :=
				next.Time.Sub(event.Time)

			delta =
				delta * time.Duration(ReplaySpeed)

			if delta < 0 {
				delta = 0
			}

			if delta > 2*time.Second {
				delta = 2 * time.Second
			}

			time.Sleep(delta)
		}
	}
}