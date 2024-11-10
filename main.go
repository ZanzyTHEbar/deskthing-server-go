package main

import (
	"deskthing-server/pkg/deskthing"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var wsConn *websocket.Conn

func main() {
	dt := deskthing.GetInstance()
	dt.On(deskthing.IncomingEventStart, func(args ...interface{}) {
		fmt.Println("DeskThing started")
	})

	// Initialize DeskThing with the necessary functions
	err := dt.Start(deskthing.StartData{
		ToServer:  toServer,
		SysEvents: sysEvents,
	})
	if err != nil {
		fmt.Println("Failed to start DeskThing:", err)
		return
	}

	// Connect to the server
	err = connectToServer(dt)
	if err != nil {
		fmt.Println("Failed to connect to server:", err)
		return
	}

	// Register event listeners
	dt.On(deskthing.IncomingEventStart, func(args ...interface{}) {
		fmt.Println("DeskThing started")
	})

	dt.On(deskthing.IncomingEventStop, func(args ...interface{}) {
		fmt.Println("DeskThing stopped")
	})

	// Example: Register an action
	dt.RegisterAction("Print Hello", "printHello", "Prints Hello to the console", "")

	// Handle incoming data from the server (mocked for this example)
	go func() {
		// Simulate receiving data from the server after 2 seconds
		time.Sleep(2 * time.Second)
		incomingData := deskthing.IncomingData{
			Type:    deskthing.IncomingEventData,
			Request: "",
			Payload: map[string]interface{}{
				"message": "Hello from server",
			},
		}
		dt.ToClient(incomingData)
	}()

	// Wait for termination signals (e.g., Ctrl+C)
	waitForShutdown(dt)
}

func toServer(data deskthing.OutgoingData) {
	if wsConn == nil {
		fmt.Println("No server connection established")
		return
	}
	message, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshaling data:", err)
		return
	}
	err = wsConn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		fmt.Println("Error sending data to server:", err)
	}
}

func sysEvents(event string, listener deskthing.DeskthingListener) func() {
	// Implement system event handling as needed
	return func() {}
}

func connectToServer(dt *deskthing.DeskThing) error {
	var err error
	wsConn, _, err = websocket.DefaultDialer.Dial("ws://yourserver.com/socket", nil)
	if err != nil {
		return err
	}

	go func() {
		defer wsConn.Close()
		for {
			_, message, err := wsConn.ReadMessage()
			if err != nil {
				fmt.Println("Error reading message:", err)
				break
			}
			var incomingData deskthing.IncomingData
			err = json.Unmarshal(message, &incomingData)
			if err != nil {
				fmt.Println("Error unmarshaling incoming data:", err)
				continue
			}
			dt.ToClient(incomingData)
		}
	}()

	return nil
}

func waitForShutdown(dt *deskthing.DeskThing) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
	err := dt.Stop()
	if err != nil {
		fmt.Println("Error during shutdown:", err)
	}
	if wsConn != nil {
		wsConn.Close()
	}
}
