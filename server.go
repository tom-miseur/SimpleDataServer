package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "0.0.0.0:81", "http service address")
var upgrader = websocket.Upgrader{} // use default options
var connectedClients int            // track connected WS clients

// DataStore contains all of the data in the SimpleDataServer
type DataStore struct {
	mu  sync.Mutex
	Csv map[string][]string
	// kvp map[string]interface{} // arbitrary types?
	Kvp map[string]string // just string for now
}

// IncomingMessage represents a client -> server message
type IncomingMessage struct {
	Command string   // valid commands: [addTop, addBottom, removeTop, removeBottom, get, set]
	Key     string   // the 'key' being manipulated
	Values  []string // the values being added ('addTop', 'addBottom')
	Value   string   // the singular value (upload, set, get)
}

// OutgoingMessage represents a server -> admin client broadcast message
type OutgoingMessage struct {
	Type    string   // valid types: [connectedClients, dataEvent, download]
	Key     string   // the 'key' being manipulated. Always present for 'dataEvent'
	Command string   // the issued command
	Values  []string // the value(s) that were added/removed
	Value   string   // the singular value (download, set, get)
}

type SyncDataMessage struct {
	Type string // valid types: [csv, kvp]
	Data interface{}
}

var dataStore = new(DataStore)
var adminClients = make(map[*websocket.Conn]bool)
var broadcast = make(chan *OutgoingMessage)

/*
TODO:
.clear(key) // purge all values set against the specified CSV key
// Web API
- synchronize - in case data can get out of step between server and admin clients. Needs testing
*/

func home(w http.ResponseWriter, r *http.Request) {
	fs := http.FileServer(http.Dir("./public"))
	fs.ServeHTTP(w, r)
}

// entry point for admin clients
func startAdmin(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	adminClients[c] = true

	// send whatever is in the dataStore already to the client:
	syncCsv(c)
	syncKvp(c)

	go adminListen(c)
}

func syncCsv(c *websocket.Conn) {
	syncMsg := SyncDataMessage{
		Type: "csvSync",
		Data: dataStore.Csv,
	}

	sendMsg(c, &syncMsg)
}

func syncKvp(c *websocket.Conn) {
	syncMsg := SyncDataMessage{
		Type: "kvpSync",
		Data: dataStore.Kvp,
	}
	sendMsg(c, &syncMsg)
}

func sendMsg(c *websocket.Conn, msg *SyncDataMessage) {
	err := c.WriteJSON(msg)
	if err != nil {
		log.Printf("Websocket error: %s", err)
		c.Close()
	}
}

func broadcaster(msg *OutgoingMessage) {
	broadcast <- msg
}

func broadcastListen() {
	fmt.Println("Server ready: http://localhost:81/")
	for {
		msg := <-broadcast

		for client := range adminClients {
			// fmt.Printf("%+v\n", msg)
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("Websocket error: %s", err)
				client.Close()
				delete(adminClients, client)
			}
		}
	}
}

// entry point for data clients
func connect(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	defer c.Close()

	connectedClients++
	broadcastClientEvent()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			// client closed the connection
			// log.Println("read:", err)

			connectedClients--
			broadcastClientEvent()

			break
		}
		// log.Printf("recv mt: %d, data: %s", mt, message)

		processMessage(c, message)
	}
}

func adminListen(c *websocket.Conn) {
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			// admin connection closed
			c.Close()
			delete(adminClients, c)
			return
		}

		incMsg := IncomingMessage{}
		json.Unmarshal(msg, &incMsg)

		switch incMsg.Command {
		case "download":
			download(c)
		case "upload":
			upload(c, incMsg.Value)
		}
	}
}

func broadcastClientEvent() {
	clients := []string{strconv.Itoa(connectedClients)}
	outMsg := OutgoingMessage{
		Type:   "clientEvent",
		Values: clients,
	}
	go broadcaster(&outMsg)
}

func broadcastDataEvent(key string, cmd string, values []string) {
	outMsg := OutgoingMessage{
		Type:    "dataEvent",
		Command: cmd,
		Key:     key,
		Values:  values,
	}
	go broadcaster(&outMsg)
}

func processMessage(c *websocket.Conn, msg []byte) {
	incMsg := IncomingMessage{}
	json.Unmarshal(msg, &incMsg)
	//fmt.Println("Received command: " + incMsg.Command)

	var ret string

	switch incMsg.Command {
	case "addTop":
		addTop(incMsg)
	case "addBottom":
		addBottom(incMsg)
	case "removeTop":
		ret = removeTop(incMsg)
	case "removeBottom":
		ret = removeBottom(incMsg)
	case "set":
		set(incMsg)
	case "get":
		ret = get(incMsg)
	default:
		ret = "Error: Unknown command"
	}

	// useful debug output:
	// fmt.Println("ret: " + ret)
	// fmt.Println(dataStore)

	if len(ret) > 0 {
		sendResponse(c, ret)
	}
}

func sendResponse(c *websocket.Conn, val string) {
	err := c.WriteMessage(websocket.TextMessage, []byte(val))
	if err != nil {
		log.Println("write:", err)
	}
}

func addTop(incMsg IncomingMessage) {
	dataStore.mu.Lock()

	var csvByKey = dataStore.Csv[incMsg.Key]
	csvByKey = append(incMsg.Values, csvByKey...)
	dataStore.Csv[incMsg.Key] = csvByKey

	broadcastDataEvent(incMsg.Key, "addTop", incMsg.Values)

	dataStore.mu.Unlock()
}

func addBottom(incMsg IncomingMessage) {
	dataStore.mu.Lock()

	var csvByKey = dataStore.Csv[incMsg.Key]
	csvByKey = append(csvByKey, incMsg.Values...)
	dataStore.Csv[incMsg.Key] = csvByKey

	broadcastDataEvent(incMsg.Key, "addBottom", incMsg.Values)

	dataStore.mu.Unlock()
}

func removeTop(incMsg IncomingMessage) string {
	dataStore.mu.Lock()

	var csvByKey = dataStore.Csv[incMsg.Key]
	var val = csvByKey[0]
	dataStore.Csv[incMsg.Key] = remove(csvByKey, 0)

	broadcastDataEvent(incMsg.Key, "removeTop", []string{val})

	dataStore.mu.Unlock()

	return val
}

func removeBottom(incMsg IncomingMessage) string {
	dataStore.mu.Lock()

	var csvByKey = dataStore.Csv[incMsg.Key]
	var val = csvByKey[len(csvByKey)-1]
	dataStore.Csv[incMsg.Key] = remove(csvByKey, len(csvByKey)-1)

	broadcastDataEvent(incMsg.Key, "removeBottom", []string{val})

	dataStore.mu.Unlock()

	return val
}

func remove(slice []string, i int) []string {
	copy(slice[i:], slice[i+1:])
	return slice[:len(slice)-1]
}

func set(incMsg IncomingMessage) {
	dataStore.Kvp[incMsg.Key] = incMsg.Value

	broadcastDataEvent(incMsg.Key, "set", []string{incMsg.Value})
}

func get(incMsg IncomingMessage) string {
	kvpByKey := ""
	// poll indefinitely for a value every second (not ideal)
	for len(kvpByKey) < 1 {
		if kvpByKey, ok := dataStore.Kvp[incMsg.Key]; ok {
			broadcastDataEvent(incMsg.Key, "get", []string{kvpByKey})
			return kvpByKey
		}
		time.Sleep(1 * time.Second)
	}

	// because the above is an infinite loop, we'll never reach here:
	broadcastDataEvent(incMsg.Key, "get", []string{""})
	return ""
}

// download marshals the dataStore to beautified JSON
func download(c *websocket.Conn) {
	data, err := json.Marshal(dataStore)
	if err != nil {
		fmt.Println(err)
	}

	jsonStr := string(data)

	outMsg := OutgoingMessage{
		Type:  "download",
		Value: jsonStr,
	}

	err2 := c.WriteJSON(outMsg)
	if err2 != nil {
		// admin connection closed
		c.Close()
		delete(adminClients, c)
	}
}

// upload replaces the dataStore
func upload(c *websocket.Conn, jsonStr string) {
	err := json.Unmarshal([]byte(jsonStr), dataStore)
	if err != nil {
		log.Printf("Error reading uploaded JSON: %s", err)
	}

	syncCsv(c)
	syncKvp(c)
}

func test() {
	//dataStore.Csv = make(map[string][]string)
	dataStore.Csv["key1"] = []string{"row1", "row2", "row3"}
	dataStore.Csv["key2"] = []string{"row1", "row2", "row3", "row4", "row5"}
	fmt.Println(dataStore.Csv)
}

func main() {
	dataStore.Csv = make(map[string][]string)
	dataStore.Kvp = make(map[string]string)

	//test()
	flag.Parse()
	log.SetFlags(0)
	router := mux.NewRouter()
	router.HandleFunc("/connect", connect).Methods("GET")
	router.HandleFunc("/admin", startAdmin).Methods("GET")
	router.HandleFunc("/", home)

	// static assets
	router.PathPrefix("/public/").Handler(http.StripPrefix("/public/", http.FileServer(http.Dir("./public/"))))

	go broadcastListen()

	log.Fatal(http.ListenAndServe(*addr, router))
}
