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
	"strconv"
	"syscall"

	. "github.com/dienakakim/r6mapbanbackend/includes"
)

const DATA_FILENAME = "data.gob"

// Allowed maps
const (
	BANK              = "Bank"
	BORDER            = "Border"
	CHALET            = "Chalet"
	CLUBHOUSE         = "Clubhouse"
	COASTLINE         = "Coastline"
	CONSULATE         = "Consulate"
	KAFE              = "Kafe"
	KANAL             = "Kanal"
	OREGON            = "Oregon"
	OUTBACK           = "Outback"
	THEMEPARK         = "Theme Park"
	VILLA             = "Villa"
	FAVELA            = "Favela"
	FORTRESS          = "Fortress"
	HEREFORDBASE      = "Hereford Base"
	HOUSE             = "House"
	PRESIDENTIALPLANE = "Presidential Plane"
	SKYSCRAPER        = "Skyscraper"
	TOWER             = "Tower"
	YACHT             = "Yacht"
)

func main() {
	port := flag.Int("port", 4000, "port to serve")
	flag.Parse()

	// Session map
	var sessionMap map[string]Session
	var mapPool map[string]bool

	// Check if DATA_FILENAME exists
	dataGob, err := os.Open(DATA_FILENAME)
	if err != nil {
		if os.IsNotExist(err) {
			// No such file found, create new session map and map pool
			log.Printf("\"%s\" not found; created new", DATA_FILENAME)
			sessionMap = make(map[string]Session)
			mapPool = make(map[string]bool)
			maps := []string{BANK, BORDER, CHALET, CLUBHOUSE, COASTLINE, CONSULATE, FAVELA, FORTRESS, HEREFORDBASE, HOUSE, KAFE, KANAL, OREGON, OUTBACK, PRESIDENTIALPLANE, SKYSCRAPER, THEMEPARK, TOWER, VILLA, YACHT}
			for _, m := range maps {
				mapPool[m] = true
			}
		} else {
			// File is found, but cannot be opened
			log.Fatalf("File \"%s\" cannot be opened", DATA_FILENAME)
		}
	} else {
		// File opened, decode gob into sessionMap
		dataGobDecoder := gob.NewDecoder(dataGob)
		err := dataGobDecoder.Decode(&sessionMap)
		if err != nil {
			log.Println("Session map cannot be decoded")
		}
		err = dataGobDecoder.Decode(&mapPool)
		if err != nil {
			log.Println("Map pool cannot be decoded")
		}
		dataGob.Close()
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
				// Save session map and map pool into DATA_FILENAME
				file, err := os.Create(DATA_FILENAME)
				defer file.Close()
				if err != nil {
					log.Panic("Sessions cannot be marshaled!!!")
				}
				dataGobEncoder := gob.NewEncoder(file)
				dataGobEncoder.Encode(sessionMap)
				dataGobEncoder.Encode(mapPool)
				// Data saved, can now exit
				done <- true
				log.Println("Data marshaled")
			}
		}
	}()

	// Associate handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/", http.NotFound)
	mux.HandleFunc("/process", handlerBuilder(sessionMap, mapPool))

	log.Printf("Starting server on port %d", *port)
	go func() {
		err = http.ListenAndServe(":"+strconv.Itoa(*port), mux)
		if err != nil {
			log.Fatal("Server cannot be started, terminating")
		}
	}()
	log.Printf("Server started. Send SIGINT to exit")
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

func handlerBuilder(sessionMap map[string]Session, mapPool map[string]bool) func(http.ResponseWriter, *http.Request) {
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
		phase, err := strconv.Atoi(r.Header.Get("MapBan-Phase"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "Expected integer phase value, or phase not set")
			return
		}
		var action string
		if phase == 1 || phase == 2 || phase == 5 || phase == 6 {
			action = "banned"
		} else if phase == 3 || phase == 4 || phase == 7 {
			action = "picked"
		}
		switch phase {
		case 0:
			// Phase 0
			// Expected JSON body:
			// {
			//		orangeTeamName: "...",
			//		blueTeamName: "...",
			//		mapPool: ["...", "...", ...] // length 7+
			// }
			var initSession InitSession
			b := make([]byte, r.ContentLength)
			r.Body.Read(b)
			if err := json.Unmarshal(b, &initSession); err != nil {
				log.Println("Cannot decode json")
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "Malformed JSON body")
				return
			}

			// Check for nil fields
			if initSession.BlueTeamName == nil || initSession.OrangeTeamName == nil || initSession.MapPool == nil {
				w.WriteHeader(http.StatusBadRequest)
				if initSession.OrangeTeamName == nil {
					fmt.Fprintln(w, "Orange team name missing")
				}
				if initSession.BlueTeamName == nil {
					fmt.Fprintln(w, "Blue team name missing")
				}
				if initSession.MapPool == nil {
					fmt.Fprintln(w, "Map pool missing")
				}
				return
			}

			// Check if team names are blank
			if *initSession.OrangeTeamName == "" || *initSession.BlueTeamName == "" {
				w.WriteHeader(http.StatusBadRequest)
				if *initSession.OrangeTeamName == "" {
					fmt.Fprintln(w, "Orange team name cannot be blank")
				}
				if *initSession.OrangeTeamName == "" {
					fmt.Fprintln(w, "Blue team name cannot be missing")
				}
			}

			// Check if at least 7 maps are sent
			if len(initSession.MapPool) < 7 {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "At least 7 maps required, found %d\n", len(initSession.MapPool))
				return
			}

			// Check if submitted map pool are allowed
			for _, m := range initSession.MapPool {
				if !mapPool[m] {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, "Map not allowed: %s\n", m)
					return
				}
			}

			// JSON decoded -- t contains TeamNames
			// Create session
			hostToken := generateToken()
			s := Session{HostToken: hostToken, OrangeTeamToken: generateToken(), BlueTeamToken: generateToken(), OrangeTeamName: *initSession.OrangeTeamName, BlueTeamName: *initSession.BlueTeamName, MapPool: initSession.MapPool, MapsChosen: make([]string, 0, 7)}
			sessionMap[hostToken] = s                                                                         // host
			sessionMap[s.OrangeTeamToken] = Session{HostToken: hostToken, OrangeTeamToken: s.OrangeTeamToken} // orange team
			sessionMap[s.BlueTeamToken] = Session{HostToken: hostToken, BlueTeamToken: s.BlueTeamToken}       // blue team
			log.Printf("Created session \"%s\" with orange team \"%s\" and blue team \"%s\"", hostToken, s.OrangeTeamToken, s.BlueTeamToken)

			// Encode session as JSON response
			sessionJson, err := json.Marshal(s)
			if err != nil {
				// Error writing response, code 500 Internal Server Error
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "Could not create JSON formatted response")
				return
			}
			var buf bytes.Buffer
			json.Indent(&buf, sessionJson, "", "  ")
			fmt.Fprintln(os.Stderr, buf.String())

			// sessionJson is response
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Write(sessionJson)
		case 1:
			fallthrough
		case 3:
			fallthrough
		case 5:
			// Phases 1, 3, 5 -- Orange Team ban/pick
			// Expects token from an Orange Team
			// {
			// 		token: "...",
			//		choice: "..."
			// }
			var mapChoice MapChoice
			b := make([]byte, r.ContentLength)
			r.Body.Read(b)
			if err := json.Unmarshal(b, &mapChoice); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "Cannot parse JSON request body")
				return
			}

			// Check for nil fields
			if mapChoice.Choice == nil || mapChoice.Token == nil {
				w.WriteHeader(http.StatusBadRequest)
				if mapChoice.Choice == nil {
					fmt.Fprintln(w, "Map choice missing")
				}
				if mapChoice.Token == nil {
					fmt.Fprintln(w, "Token missing")
				}
			}

			// Check if token references active session
			session, found := sessionMap[*mapChoice.Token]
			if !found {
				w.WriteHeader(http.StatusInternalServerError) // server fault since sessions must be preserved
				fmt.Fprintln(w, "Session not found")
				return
			}

			// Session found, find host session and
			// check if it is from an orange team
			if *mapChoice.Token != session.OrangeTeamToken {
				w.WriteHeader(http.StatusForbidden) // not orange team
				fmt.Fprintln(w, "Not an orange team")
				return
			}
			session = sessionMap[session.HostToken]

			// Session from orange team confirmed
			// Check if choice is duplicate
			for _, m := range session.MapsChosen {
				if m == *mapChoice.Choice {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, "Duplicate map")
					return
				}
			}

			// Check if map banned is in host session's map pool
			found = false
			for _, m := range session.MapPool {
				if m == *mapChoice.Choice {
					found = true
					break
				}
			}
			if !found {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "Map not found in host map pool: %s\n", *mapChoice.Choice)
				return
			}

			// All verified. Append to list of chosen maps
			log.Printf("Phase %d: \"%s\" %s \"%s\"", phase, session.OrangeTeamName, action, *mapChoice.Choice)
			session.MapsChosen = append(session.MapsChosen, *mapChoice.Choice)
			// Send MapsChosen as response
			resp, err := json.Marshal(session.MapsChosen)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "Cannot encode JSON response")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			sessionMap[session.HostToken] = session
		case 2:
			fallthrough
		case 4:
			fallthrough
		case 6:
			// Phases 2, 4, 6 -- Blue Team ban/pick
			// Expects token from a Blue Team, and non-duplicate choice
			// {
			// 		token: "...",
			//		choice: "..."
			// }
			var mapChoice MapChoice
			b := make([]byte, r.ContentLength)
			r.Body.Read(b)
			if err := json.Unmarshal(b, &mapChoice); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "Cannot parse JSON request body")
				return
			}

			// Check for nil fields
			if mapChoice.Choice == nil || mapChoice.Token == nil {
				w.WriteHeader(http.StatusBadRequest)
				if mapChoice.Choice == nil {
					fmt.Fprintln(w, "Map choice missing")
				}
				if mapChoice.Token == nil {
					fmt.Fprintln(w, "Token missing")
				}
			}

			// Check if token references active session
			session, found := sessionMap[*mapChoice.Token]
			if !found {
				w.WriteHeader(http.StatusInternalServerError) // server fault since sessions must be preserved
				fmt.Fprintln(w, "Session not found")
				return
			}

			// Session found, find host session and
			// check if it is from a blue team
			if *mapChoice.Token != session.BlueTeamToken {
				w.WriteHeader(http.StatusForbidden) // not blue team
				fmt.Fprintln(w, "Not a blue team")
				return
			}
			session = sessionMap[session.HostToken]

			// Session from blue team confirmed
			// Check if choice is duplicate
			for _, m := range session.MapsChosen {
				if m == *mapChoice.Choice {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, "Duplicate map")
					return
				}
			}

			// Check if map banned is in host session's map pool
			found = false
			for _, m := range session.MapPool {
				if m == *mapChoice.Choice {
					found = true
					break
				}
			}
			if !found {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "Map not found in host map pool: %s\n", *mapChoice.Choice)
				return
			}

			// All verified. Append to list of chosen maps
			log.Printf("Phase %d: \"%s\" %s \"%s\"", phase, session.BlueTeamName, action, *mapChoice.Choice)
			session.MapsChosen = append(session.MapsChosen, *mapChoice.Choice)
			// Send MapsChosen as response
			resp, err := json.Marshal(session.MapsChosen)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "Cannot encode JSON response")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			sessionMap[session.HostToken] = session
		case 7:
			// Phase 7 -- Host decider
			// Expects token from the host, and non-duplicate choice
			// {
			// 		token: "...",
			//		choice: "..."
			// }
			var mapChoice MapChoice
			b := make([]byte, r.ContentLength)
			r.Body.Read(b)
			if err := json.Unmarshal(b, &mapChoice); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "Cannot parse JSON request body")
				return
			}

			// Check for nil fields
			if mapChoice.Choice == nil || mapChoice.Token == nil {
				w.WriteHeader(http.StatusBadRequest)
				if mapChoice.Choice == nil {
					fmt.Fprintln(w, "Map choice missing")
				}
				if mapChoice.Token == nil {
					fmt.Fprintln(w, "Token missing")
				}
			}

			// Check if token references active session
			session, found := sessionMap[*mapChoice.Token]
			if !found {
				w.WriteHeader(http.StatusInternalServerError) // server fault since sessions must be preserved
				fmt.Fprintln(w, "Session not found")
				return
			}

			// Session found, check if it is a host session
			if *mapChoice.Token != session.HostToken {
				w.WriteHeader(http.StatusForbidden) // not blue team
				fmt.Fprintln(w, "Not a blue team")
				return
			}
			// `session` is already the host session

			// Session from host confirmed
			// Check if choice is duplicate
			for _, m := range session.MapsChosen {
				if m == *mapChoice.Choice {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, "Duplicate map")
					return
				}
			}

			// Check if map chosen is in map pool
			found = false
			for _, m := range session.MapPool {
				if m == *mapChoice.Choice {
					found = true
					break
				}
			}
			if !found {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "Map not found in host map pool: %s\n", *mapChoice.Choice)
				return
			}

			// All verified. Append to list of chosen maps
			log.Printf("Phase 7: Host %s \"%s\"", action, *mapChoice.Choice)
			session.MapsChosen = append(session.MapsChosen, *mapChoice.Choice)
			// Send 3 picked maps as response
			finalMaps := []string{session.MapsChosen[2], session.MapsChosen[3], session.MapsChosen[6]}
			resp, err := json.Marshal(finalMaps)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "Cannot encode JSON response")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			// Phase 7 is ending phase. Close host, OT, and BT sessions
			delete(sessionMap, session.HostToken)
			delete(sessionMap, session.OrangeTeamToken)
			delete(sessionMap, session.BlueTeamToken)
			log.Printf("Closed sessions \"%s\", \"%s\", and \"%s\". Picked maps are %v", session.HostToken, session.OrangeTeamToken, session.BlueTeamToken, string(resp))
		}
	}
}
