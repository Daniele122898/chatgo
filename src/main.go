package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"fmt"
)

var (
	clients = make(map[*websocket.Conn]string) 	// connected clients
	broadcast = make(chan Message)				// broadcast channel
	upgrader = websocket.Upgrader{}
	systemAuthor = Author{Username:"SYSTEM", AvatarUrl:""}
)

type Author struct {
	Username string `json:"username"`
	AvatarUrl string `json:"avatarurl"`
}

type Message struct {
	Author Author `json:"author"`
	Message string `json:"message"`
}

func main() {
	// Create a simple file server
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)
	// Configure websocke route
	http.HandleFunc("/ws", handleConnections) //run as go routine
	// Start listening for incoming chat messages
	go handleMessages()
	// Start the server on localhost port 8000 and log any errors
	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request){
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Make sure we close the connecton when the function returns
	defer ws.Close()
	//Get Username of user
	var author Author
	err = ws.ReadJSON(&author)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}
	fmt.Println("User Registered: ",author.Username)
	// Register our new client
	clients[ws] = author.Username

	//Send welcome message
	welcomeMSG := Message{Author:systemAuthor, Message: author.Username+" joined the chat!"}
	broadcast <- welcomeMSG

	// Infinite loop to wait and read messages from websocket
	for{
		var msg Message
		// Read in a new message as JSON and map it to the message object
		err = ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			//broadcast that user left
			broadcast <- Message{Author:systemAuthor, Message: author.Username+ " left the chat!"}
			break
		}
		// Send the newly received message to the broadcast channel
		broadcast <- msg
	}
}

func handleMessages(){
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		// Send it out to every client that is currently connected
		for client:= range clients {
			err:= client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				username:= clients[client]
				client.Close()
				delete(clients, client)
				//broadcast that user left
				broadcast <- Message{Author:systemAuthor, Message: username+" left the chat!"}
			}
		}
	}
}