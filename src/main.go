package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/xid"
	"html/template"
)

const(
	MESSAGE_REC = iota 	//0
	ROOM_LIST			//1
	JOINED_ROOM 		//2
	CREATED_ROOM		//3
	LEFT_ROOM			//4
	MESSAGE_HISTORY		//5
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
	messages []*Message
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

type SendData struct{
	OpCode int `json:"opcode"`
	Data interface{} `json:"data"`
}

type RecvData struct{
	OpCode int `json:"opcode"`
	Data map[string]interface{} `json:"data"`
}


func main() {
	// Create a simple file server
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)
	// Configure websocke route
	http.HandleFunc("/ws", handleConnections) //run as go routine
	// Start listening for incoming chat messages
	go handleMessages()
	//Create initial Room
	rooms[xid.New().String()] = &Room{clients:make(map[*websocket.Conn]string), Name:"General"}
	rooms[xid.New().String()] = &Room{clients:make(map[*websocket.Conn]string), Name:"TEST"}
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
	err = ws.WriteJSON(SendData{OpCode:ROOM_LIST, Data: rooms})
	if err != nil {
		log.Printf("error: %v", err)
		return
	}
	handleClientLoop(ws, &client)
}

func handleClientLoop(ws *websocket.Conn, client *Author) {
	var ra RoomAction
	var room *Room
	// Infinite loop to wait and read messages from websocket
	for{
		var dataRec RecvData
		// Read in a new message as JSON and map it to the message object
		err := ws.ReadJSON(&dataRec)
		if err != nil {
			log.Printf("error reading json: %v", err)
			if room != nil {
				if _, ok := room.clients[ws]; ok {
					delete(room.clients, ws)
					//broadcast that user left
					broadcast <- Message{Author: systemAuthor, Message: client.Username + " left the chat!", RoomId: ra.Id}
				}
			}
			break
		}
		opC := int(dataRec.OpCode)
		switch opC {
		case MESSAGE_REC:
			var msgRec Message
			err = mapstructure.Decode(dataRec.Data, &msgRec)
			if err != nil {
				log.Printf("error parsing message: %v", err)
				continue
			}
			// Send the newly received message to the broadcast channel
			broadcast <- msgRec
		case LEFT_ROOM:
			delete(room.clients, ws)
			log.Println(client.Username, " left ", room.Name)
			//broadcast that user left
			broadcast <- Message{Author:systemAuthor, Message: client.Username+ " left the chat!", RoomId:ra.Id}
			room = nil
		case JOINED_ROOM:
			//Join room
			//try converting to room action
			err = mapstructure.Decode(dataRec.Data, &ra)
			if err != nil {
				log.Printf("error decode roomaction: %v", err)
				return
			}
			var ok bool
			room, ok = rooms[ra.Id]
			if !ok {
				log.Println("COULDN'T FIND ROOM IN LIST")
				return
			}
			err = ws.WriteJSON(SendData{OpCode: MESSAGE_HISTORY, Data: room.messages})
			if err != nil {
				log.Printf("error sending message History: %v", err)
				return
			}
			// Register our new client
			room.clients[ws] =client.Username
			log.Println(client.Username, " joined ", room.Name)
			//Send welcome message
			welcomeMSG := Message{Author:systemAuthor, Message: client.Username+" joined the chat!", RoomId:ra.Id}
			broadcast <- welcomeMSG
		case CREATED_ROOM:
			//Create room
			//try converting to room action
			err = mapstructure.Decode(dataRec.Data, &ra)
			if err != nil {
				log.Printf("error decode roomaction: %v", err)
				return
			}
			ra.Id = xid.New().String()
			rooms[ra.Id] = &Room{clients:make(map[*websocket.Conn]string), Name: ra.Name}
			//Send list of rooms
			err = ws.WriteJSON(SendData{OpCode:ROOM_LIST, Data: rooms})
			if err != nil {
				log.Printf("error: %v", err)
				return
			}
		case ROOM_LIST:
			//Client wants a refresh of the rooms
			//Send list of rooms
			err = ws.WriteJSON(SendData{OpCode:ROOM_LIST, Data: rooms})
			if err != nil {
				log.Printf("error: %v", err)
				return
			}
		default:
			log.Fatal("OPCODE DOESNT MATCH")
			continue
		}
	}
}

func handleMessages(){
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		//Get Room
		room, ok := rooms[msg.RoomId]
		if !ok{
			log.Println("error: Didn't find room with roomID")
			continue
		}
		//remove any HTML from message
		msg.Message = template.HTMLEscapeString(msg.Message)
		//room was found and message was successfully received so save to messages
		//don't save messages from system.
		if msg.Author.Username != systemAuthor.Username {
			room.messages = append(room.messages, &msg)
		}
		// Send it out to every client that is currently connected
		for client:= range room.clients {
			err:= client.WriteJSON(SendData{OpCode:MESSAGE_REC, Data:msg})
			if err != nil {
				log.Printf("error writing message: %v", err)
				username:= room.clients[client]
				client.Close()
				delete(room.clients, client)
				//broadcast that user left
				broadcast <- Message{Author:systemAuthor, Message: username+" left the chat!", RoomId: msg.RoomId}
			}
		}
	}
}