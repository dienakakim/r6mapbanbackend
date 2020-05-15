package main

import (
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
	"strconv"
	"syscall"

	. "github.com/dienakakim/r6mapbanbackend/includes"
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
	signal.Notify(signals, os.Interrupt)
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

// generateToken generates a 24-byte-long token for use in uniquely identifying a frontend node.
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
		w.Header().Set("Access-Control-Allow-Methods", http.MethodPost)

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
			b := make([]byte, r.ContentLength)
			r.Body.Read(b)
			if err := json.Unmarshal(b, &t); err != nil {
				log.Println("Cannot decode json")
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "Malformed JSON body")
				return
			}

			// Check for orangeTeamName and blueTeamName
			if t.BlueTeamName == nil || t.OrangeTeamName == nil {
				log.Println("Team names missing")
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "Required team names missing")
				return
			}

			// JSON decoded -- t contains TeamNames
			// Create session
			hostToken := generateToken()
			s := Session{HostToken: hostToken, OrangeTeamToken: generateToken(), BlueTeamToken: generateToken(), OrangeTeamName: *t.OrangeTeamName, BlueTeamName: *t.BlueTeamName, MapsChosen: []string{}}
			sessionMap[hostToken] = s                                     // host
			sessionMap[s.OrangeTeamToken] = Session{HostToken: hostToken} // orange team
			sessionMap[s.BlueTeamToken] = Session{HostToken: hostToken}   // blue team

			// Encode session as JSON response
			sessionJson, err := json.Marshal(s)
			if err != nil {
				// Error writing response, code 500 Internal Server Error
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "Could not create JSON formatted response")
				return
			}

			// sessionJson is response
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Write(sessionJson)
		case "8":
			// Phase 8 (results phase), mapban done
			// Close session
			token := r.URL.Query().Get("token")
			if token == "" {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "Malformed request -- expected token")
				return
			}

			// Look up token in session map
			s, found := sessionMap[token]
			if !found {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "Malformed request -- expected valid token")
				return
			}

			// s found, could either be host, OT, or BT token
			// OT or BT token
			if s.HostToken != token {
				// Error -- only hosts can close sessions
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprintln(w, "Not host -- closing not allowed")
				return
			}

			// Get chosen maps
			mapsChosen := s.MapsChosen

			// Close host, OT, and BT sessions
			delete(sessionMap, s.OrangeTeamToken)
			delete(sessionMap, s.BlueTeamToken)
			delete(sessionMap, s.HostToken)

			// Closing sessions successful
			// Return maps chosen
			result, err := json.Marshal(mapsChosen)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "Cannot encode JSON response")
				return
			}

			// Encoded successfully
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Write(result)
			return
		default:
			// Not implemented
			w.WriteHeader(http.StatusNotImplemented)
			fmt.Fprintln(w, "Will be handled soon")
		}
	}
}
