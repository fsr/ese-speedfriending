package main

import (
    "encoding/json"
    "fmt"
    "html/template"
    "io/ioutil"
    "log"
    "net/http"
    "sync"
    "time"

    "github.com/segmentio/ksuid"
)

type config struct {
    Freelist free
    PpS      int
    BaseURL  string
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
var freeTemplate *template.Template
var linklistTemplate *template.Template
var freelistTemplate *template.Template

var waitingClient *client

func handleFree(w http.ResponseWriter, r *http.Request) {
    conf.Freelist.M.Lock()
    for i := range conf.Freelist.Urls {
        if conf.Freelist.Urls[i] == r.FormValue("url") {
            http.Redirect(w, r, conf.BaseURL+"/free/"+r.FormValue("url"), http.StatusSeeOther)
            return
        }
    }
    conf.Freelist.Urls = append(conf.Freelist.Urls, r.FormValue("url"))
    log.Printf("%s added to free link list", r.FormValue("url"))
    conf.Freelist.M.Unlock()
    http.Redirect(w, r, conf.BaseURL+"/free/"+r.FormValue("url"), http.StatusSeeOther)
}

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
        link := conf.Freelist.Urls[0]
        conf.Freelist.Urls = conf.Freelist.Urls[1:]

        waitingClient.Link = link
        currentClient.Link = link
        waitingClient = nil
    } else {
        waitingClient = currentClient
    }
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
    c.M.Lock()
    conf.Freelist.M.Lock()

    uuid := ksuid.New().String()
    log.Printf("uuid registered: %s", uuid)
    currentClient := client{ID: uuid, LastAccess: time.Now(), Link: ""}
    c.Uuids[uuid] = &currentClient
    tryToPair(&currentClient)
    fmt.Fprintf(w, uuid)

    conf.Freelist.M.Unlock()
    c.M.Unlock()
}

func handlePoll(w http.ResponseWriter, r *http.Request) {
    c.M.Lock()
    conf.Freelist.M.Lock()

    uuid := r.FormValue("uuid")
    currentClient := c.Uuids[uuid]

    if currentClient == nil {
        log.Printf("uuid %s not registered", uuid)
        conf.Freelist.M.Unlock()
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

    conf.Freelist.M.Unlock()
    c.M.Unlock()
}

type freeData struct {
    URL string
}

func handleFreeTemplate(w http.ResponseWriter, r *http.Request) {
    url := r.URL.Query()["url"][0]
    if url == "" {
        log.Printf("empty free request")
    }
    d := freeData{URL: url}
    freeTemplate.Execute(w, d)
}

func handleLinkList(w http.ResponseWriter, r *http.Request) {
    linklistTemplate.Execute(w, linklist)
}

func handleFreeList(w http.ResponseWriter, r *http.Request) {
    freelistTemplate.Execute(w, conf.Freelist.Urls)
}

func main() {
    //load url list
    data, err := ioutil.ReadFile("config.json")
    if err != nil {
        panic(err)
    }
    json.Unmarshal(data, &conf)
    linklist = conf.Freelist.Urls

    templ, err := template.ParseFiles("templates/free.html")
    if err != nil {
        panic(err)
    }
    freeTemplate = templ

    templ, err = template.ParseFiles("templates/linklist.html")
    if err != nil {
        panic(err)
    }
    linklistTemplate = templ

    templ, err = template.ParseFiles("templates/freelist.html")
    if err != nil {
        panic(err)
    }
    freelistTemplate = templ

    waitingClient = nil
    c.Uuids = make(map[string]*client)

    http.Handle("/", http.FileServer(http.Dir("static")))
    http.HandleFunc("/api/register", handleRegister)
    http.HandleFunc("/api/poll", handlePoll)
    http.HandleFunc("/api/free", handleFree)
    http.HandleFunc("/free", handleFreeTemplate)
    http.HandleFunc("/linklist", handleLinkList)
    http.HandleFunc("/freelist", handleFreeList)
    log.Printf("base url %s (listen on %s)", conf.BaseURL, conf.Port)
    http.ListenAndServe(conf.Port, nil)
}
