package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "sync"
    "time"

    "github.com/segmentio/ksuid"
)

type config struct {
    Servers []string
    PpS      int
    Port     string
}

type free struct {
    M    sync.Mutex
    Urls []string
}

type clients struct {
    M     sync.Mutex
    Uuids map[string]*client
}

type client struct {
    ID         string
    LastAccess time.Time
    Link       string
}

var conf config
var linklist []string
var c clients
var serverIdx int

var waitingClient *client

// check if client at waiting slot is still active. Otherwise delete it.
func cleanUp() {
    if waitingClient == nil {
        return
    }
    now := time.Now()
    timeout, _ := time.ParseDuration("30s")
    if now.Sub(waitingClient.LastAccess) > timeout {
        // delete from client list
        delete(c.Uuids, waitingClient.ID)
        // free waiting slot
        waitingClient = nil
    }
}

func tryToPair(currentClient *client) {
    // delete waiting client if not active any longer
    cleanUp()
    if waitingClient != nil {
        // do not pair client with itself
        if waitingClient.ID == currentClient.ID {
            return
        }

        serverIdx += 1
        idx := serverIdx % len(conf.Servers)
        server := conf.Servers[idx]

        link := fmt.Sprintf("%s/%s", server, currentClient.ID)
        waitingClient.Link = link
        currentClient.Link = link
        waitingClient = nil
    } else {
        waitingClient = currentClient
    }
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
    c.M.Lock()

    uuid := ksuid.New().String()
    log.Printf("uuid registered: %s", uuid)
    currentClient := client{ID: uuid, LastAccess: time.Now(), Link: ""}
    c.Uuids[uuid] = &currentClient
    tryToPair(&currentClient)
    fmt.Fprintf(w, uuid)

    c.M.Unlock()
}

func handlePoll(w http.ResponseWriter, r *http.Request) {
    c.M.Lock()

    uuid := r.FormValue("uuid")
    currentClient := c.Uuids[uuid]

    if currentClient == nil {
        log.Printf("uuid %s not registered", uuid)
        c.M.Unlock()
        return
    }
    // look if paired
    if currentClient.Link != "" {
        fmt.Fprintf(w, currentClient.Link)
        log.Printf("%s assigned to %s", currentClient.Link, uuid)
    } else {
        // find pair
        tryToPair(currentClient)
        // if pairing succesfull, directly print link
        if currentClient.Link != "" {
            fmt.Fprintf(w, currentClient.Link)
            log.Printf("%s assigned to %s", currentClient.Link, uuid)
        } else {
            fmt.Fprintf(w, "wait")
        }
    }

    c.M.Unlock()
}

type freeData struct {
    URL string
}

func main() {
    //load url list
    data, err := ioutil.ReadFile("config.json")
    if err != nil {
        panic(err)
    }
    json.Unmarshal(data, &conf)

    waitingClient = nil
    c.Uuids = make(map[string]*client)

    http.Handle("/", http.FileServer(http.Dir("static")))
    http.HandleFunc("/api/register", handleRegister)
    http.HandleFunc("/api/poll", handlePoll)
    log.Printf("listen on %s", conf.Port)
    http.ListenAndServe(conf.Port, nil)
}
