package main

import (
	bytes2 "bytes"
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
	Host string `json:"host"`
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
		go func(from, host, to string) {
			// check port
			if num, err := strconv.Atoi(port.From); err != nil || num < 0 || num > 65535 {
				errorChannel <- errors.New("json error")
			}
			if num, err := strconv.Atoi(port.To); err != nil || num < 0 || num > 65535 {
				errorChannel <- errors.New("json error")
			}

			// request
			director := func(request *http.Request) {
				url := *request.URL
				url.Scheme = "http"
				url.Host = host + ":" + to

				var buf []byte
				if request.Body != nil {
					buf, err = ioutil.ReadAll(request.Body)
					if err != nil {
						errorChannel <- err
					}
				} else {
					buf = make([]byte, 0)
				}

				req, err := http.NewRequest(request.Method, url.String(), bytes2.NewBuffer(buf))

				if err != nil {
					errorChannel <- err
				}
				req.Header = request.Header

				if req.Header.Get("transparent-proxy") == "true" {
					errorChannel <- errors.New("loop detected")
				}
				req.Header.Set("transparent-proxy", "true")

				// certification
				id := ""
				cookie, err := request.Cookie("session")
				if err == nil && cookie.Value != "" {
					id, err = utils.CheckSession(cookie.Value)
					if err != nil {
						id = ""
					}
				}

				req.Header.Set("id", id)
				*request = *req
			}

			// server
			rp := &httputil.ReverseProxy{Director: director}
			server := http.Server{Addr: ":" + from, Handler: rp}
			errorChannel <- server.ListenAndServe()
		}(port.From, port.Host, port.To)
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
