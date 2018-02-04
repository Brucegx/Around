package main

import (
	"fmt"
	"net/http"
	"encoding/json"
	"log"
	"strconv"
)

const (
	DISTANCE="200km"
)

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Post struct {
	User string `json:"user"`
	Message string `json:"message"`
	Location Location `json:"location"`
}


func main() {
	fmt.Println("started-service")
	http.HandleFunc("/post", handlerPost)
	http.HandleFunc("/search", handlerSearch)
	log.Fatal(http.ListenAndServe(":8081", nil))
}


func handlerPost (w http.ResponseWriter, r *http.Request) {
	fmt.Print("Received a post request")
	var p Post
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&p); err != nil {
		panic(err)
		return
	}
	fmt.Fprint(w, "Post received: %s\n", p.Message)
}

func handlerSearch (w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received one request for search")
	lat := r.URL.Query().Get("lat")
	lon := r.URL.Query().Get("lon")

	lt, _ := strconv.ParseFloat(lat, 64)
	ln, _ := strconv.ParseFloat(lon, 64)
	ran := DISTANCE
	if val := r.URL.Query().Get("range"); val != "" {
		ran = val + "km"
	}


	fmt.Printf("Search received: %s %s %s", lat, lon, ran)

	p:= &Post{
		User:"1234",
		Message:"This is a message",
		Location: Location{
			Lat: lt,
			Lon: ln,
		},
	}
	js, err := json.Marshal(p)
	if err != nil {
		panic(err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
