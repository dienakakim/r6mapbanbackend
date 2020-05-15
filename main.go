package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	. "r6mapbanbackend/includes"
	"strconv"
	"syscall"
)

const SESSIONS_FILENAME = "sessions.gob"

func main() {
	port := flag.Int("port", 4000, "port to serve")
	flag.Parse()

	// Session map
	var sessionMap map[string]Session

	// Check if SESSIONS_FILENAME exists
	sessionsGob, err := os.Open(SESSIONS_FILENAME)
	if err != nil {
		if os.IsNotExist(err) {
			// No such file found, create new map
			sessionMap = make(map[string]Session)
		} else {
			// File is found, but cannot be opened
			log.Fatalf("File \"%s\" cannot be opened", SESSIONS_FILENAME)
		}
	} else {
		// File opened, decode gob into sessionMap
		sessionsGobDecoder := gob.NewDecoder(sessionsGob)
		sessionsGobDecoder.Decode(&sessionMap)
		sessionsGob.Close()
	}

	// Set up exit handler
	signals := make(chan os.Signal, 1)
	done := make(chan bool)
	signal.Notify(signals, syscall.SIGINT)
	go func() {
		for {
			sig := <-signals
			switch sig {
			case syscall.SIGINT:
				// User requested termination
				log.Printf("%v received", sig)
				// Save session map into SESSIONS_FILENAME
				file, err := os.Create(SESSIONS_FILENAME)
				defer file.Close()
				if err != nil {
					log.Panic("Sessions cannot be marshaled!!!")
				}
				sessionsGobEncoder := gob.NewEncoder(file)
				sessionsGobEncoder.Encode(sessionMap)
				// Session map saved, can now exit
				done <- true
				log.Printf("Sessions marshaled")
			default:
				log.Printf("%v received; not handled", sig)
			}
		}
	}()

	// Associate handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/", http.NotFound)
	mux.HandleFunc("/process", handlerBuilder(sessionMap))

	log.Printf("Starting server on port %d", *port)
	go func() {
		err = http.ListenAndServe(":"+strconv.Itoa(*port), mux)
		if err != nil {
			log.Fatal("Server cannot be started, terminating")
		}
	}()
	log.Printf("Waiting on signal")
	<-done
	log.Printf("Done.")
}

func generateToken() string {
	var b [18]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}

	return base64.URLEncoding.Strict().EncodeToString(b[:])
}

func handlerBuilder(sessionMap map[string]Session) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Header.Get("Content-Type") != "application/json" {
			// Invalid request -- expected json
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("Invalid request -- expected json, found %s", r.Header.Get("Content-Type"))
			fmt.Fprintln(w, "Invalid request -- expected json")
			return
		}

		// Check for phase value
		switch r.Header.Get("MapBan-Phase") {
		case "":
			// Phase not set
			log.Println("Phase not set")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "Phase not set")
		case "0":
			// Phase 0
			// Expected JSON body:
			// {
			//		orangeTeamName: "...",
			//		blueTeamName: "..."
			// }
			var t TeamNames
			var b bytes.Buffer
			b.ReadFrom(r.Body)
			defer r.Body.Close()
			if err := json.Unmarshal(b.Bytes(), &t); err != nil {
				log.Println("Cannot decode json")
			}

			// JSON decoded -- t contains TeamNames
			// Create session
			hostToken := generateToken()
			s := Session{HostToken: hostToken, OrangeTeamToken: generateToken(), BlueTeamToken: generateToken(), OrangeTeamName: t.OrangeTeamName, BlueTeamName: t.BlueTeamName, MapsChosen: []string{}}
			sessionMap[hostToken] = s

			// Encode session as JSON response
			sessionJson, err := json.Marshal(s)
			if err != nil {
				// Error writing response, code 500 Internal Server Error
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "Could not create JSON formatted response")
				break
			}

			// sessionJson is response
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Write(sessionJson)
		default:
			// Not implemented
			w.WriteHeader(http.StatusNotImplemented)
			fmt.Fprintln(w, "Will be handled soon")
		}
	}
}
