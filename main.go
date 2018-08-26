package main

import (
	"log"
	"net/http"
	"net/http/httputil"

	"os"

	"io/ioutil"

	"encoding/json"

	"errors"
	"strconv"

	"./utils"
)

type Port struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func main() {
	databaseAddress := ""
	if os.Getenv("DATABASE_ADDRESS") != "" {
		databaseAddress = os.Getenv("DATABASE_ADDRESS")
	}

	// connect database
	err := utils.Open(os.Getenv("DATABASE_USERNAME"), os.Getenv("DATABASE_PASSWORD"), databaseAddress, os.Getenv("DATABASE_NAME"))
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	defer utils.Close()

	// load port data
	bytes, err := ioutil.ReadFile("./conf/data.json")
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	// decode json file
	var Ports []Port
	if err := json.Unmarshal(bytes, &Ports); err != nil {
		log.Fatal(err.Error())
		return
	}

	if len(Ports) == 0 {
		return
	}

	// create reverse proxy
	errorChannel := make(chan error)
	for _, port := range Ports {
		go func() {
			// check port
			if num, err := strconv.Atoi(port.From); err != nil || num < 0 || num > 65535 {
				errorChannel <- errors.New("json error")
			}
			if num, err := strconv.Atoi(port.To); err != nil || num < 0 || num > 65535 {
				errorChannel <- errors.New("json error")
			}

			// request
			director := func(request *http.Request) {
				request.URL.Scheme = "http"
				request.URL.Host = ":" + port.To
				if request.Header.Get("transparent-proxy") == "true" {
					errorChannel <- errors.New("loop detected")
				}
				request.Header.Set("transparent-proxy", "true")

				// certification
				id := ""
				cookie, err := request.Cookie("session")
				if err == nil && cookie.Value != "" {
					id, err = utils.CheckSession(cookie.Value)
					if err != nil {
						id = ""
					}
				}

				request.Header.Set("id", id)
			}

			// server
			rp := &httputil.ReverseProxy{Director: director}
			server := http.Server{Addr: ":" + port.From, Handler: rp}
			errorChannel <- server.ListenAndServe()
		}()
	}

	// catch goroutines error
	for {
		err = <-errorChannel
		if err != nil {
			log.Fatal(err.Error())
			return
		}
	}
}
