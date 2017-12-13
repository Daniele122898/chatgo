package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"fmt"
	"github.com/mitchellh/mapstructure"
)

const(
	MESSAGE_REC = iota 	//0
	JOINED_ROOM 		//1
	CREATED_ROOM		//2
)

var (
	rooms = make(map[string]*Room) 			//map of rooms
	broadcast = make(chan Message, 32)		// broadcast channel

	upgrader = websocket.Upgrader{}
	systemAuthor = Author{Username:"SYSTEM", AvatarUrl:""}
)

type Author struct {
	Username string `json:"username"`
	AvatarUrl string `json:"avatarurl"`
}

type Room struct {
	clients  map[*websocket.Conn]string 	// connected clients
	Name string
}

type RoomAction struct {
	Id string `json:"id"`
	Name string `json:"name"`
}

type Message struct {
	Author Author `json:"author"`
	Message string `json:"message"`
	RoomId string `json:"roomid"`
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
	var client Author
	err = ws.ReadJSON(&client)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}
	fmt.Println("User Registered: ",client.Username)
	//Send list of rooms
	err = ws.WriteJSON(rooms)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}

	//wait for user to join or create a Room
	var data map[string]interface{}
	err = ws.ReadJSON(&data)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}
	opC := data["opcode"].(int)
	//try converting to room action
	delete(data, "opcode")
	var ra RoomAction
	err = mapstructure.Decode(data, &ra)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}
	//Check what the user did
	var room *Room
	switch opC {
	case JOINED_ROOM:
		//Join room
		var ok bool
		room, ok = rooms[ra.Id]
		if !ok {
			log.Println("COULDN'T FIND ROOM IN LIST")
			return
		}
		// Register our new client
		room.clients[ws] =client.Username
	case CREATED_ROOM:
		//Create room
	default:
		log.Println("User tried something before joining room. Close connection")
		return
	}

	//Send welcome message
	welcomeMSG := Message{Author:systemAuthor, Message: client.Username+" joined the chat!", RoomId:ra.Id}
	broadcast <- welcomeMSG

	// Infinite loop to wait and read messages from websocket
	for{
		var msg Message
		// Read in a new message as JSON and map it to the message object
		err = ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(room.clients, ws)
			//broadcast that user left
			broadcast <- Message{Author:systemAuthor, Message: client.Username+ " left the chat!", RoomId:ra.Id}
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
		for client:= range rooms[msg.RoomId].clients {
			err:= client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				username:= rooms[msg.RoomId].clients[client]
				client.Close()
				delete(rooms[msg.RoomId].clients, client)
				//broadcast that user left
				broadcast <- Message{Author:systemAuthor, Message: username+" left the chat!", RoomId: msg.RoomId}
			}
		}
	}
}