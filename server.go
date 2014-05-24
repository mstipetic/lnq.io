package main

import (
//	"fmt"
	"net/http"
	"net/url"
	"github.com/gorilla/mux"
	"github.com/garyburd/redigo/redis"
	"strconv"
	"strings"
	"encoding/json"
)
var redisConn = "localhost:6379"

var redisPool = &redis.Pool{
	MaxIdle:   3,
	MaxActive: 50, // max number of connections
	Dial: func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", redisConn)
		if err != nil {
			panic(err.Error())
		}
		return c, err
	},
}

func isValidURL(urlString string) bool {
	parsedUrl, err := url.Parse(urlString)
	if err != nil || len(parsedUrl.Host) == 0 || !strings.Contains(parsedUrl.Host, ".") {
		return false
	}
	return true
}

func create(w http.ResponseWriter, r *http.Request) {
	//vars := mux.Vars(r)
	//fmt.Println("CREATE")
	//new_url := r.RequestURI
	new_url := r.RequestURI//vars["url"]
	//fmt.Println("FULL REQUEST URI", r.RequestURI)
	new_url = strings.Replace(new_url, "/c/", "", 1)
	println("raw", r.RequestURI)

	//fmt.Println("Received url:", new_url)
	if !strings.Contains(new_url, "://") {
		new_url = strings.Replace(new_url, ":/", "://", 1)
	}
	println("Corrected hash:", new_url)

	if !isValidURL(new_url) {
		new_url = "http://" + new_url
		if !isValidURL(new_url) {
			http.Error(w, "Invalid URL", 400)
			return
		}
	}

	conn := redisPool.Get()
	defer conn.Close()

	link_no, _ := redis.Int(conn.Do("INCR", "global:counter"))
	link_hash := strconv.FormatInt(int64(link_no), 36)

	redis.Strings(conn.Do("SET", link_hash, new_url))

	response := make(map[string]string)
	response["fullUrl"] = new_url
	response["shortUrl"] = "lnq.io/" + link_hash

	response_text, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(response_text))
}


func root(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["hash"]

	if len(hash) == 0 {
		http.ServeFile(w, r, "./files/index.html")
		return
	}

	if hash == "favicon.ico" {
		// TODO favicon
		http.ServeFile(w, r, "./files/favicon.ico")
		return
	}
	//fmt.Println("ROOT")
	hash = r.RequestURI[1:len(r.RequestURI)]

	conn := redisPool.Get()
	defer conn.Close()

	fullUrl, err := redis.String(conn.Do("GET", hash))
	if err != nil || len(fullUrl) == 0 {
		//fmt.Println("ERROR", len(fullUrl))
		http.ServeFile(w, r, "./files/error.html")
		return
	}
	//fmt.Println("FullURL", fullUrl)
	http.Redirect(w, r, fullUrl, http.StatusFound)
}

//set global:counter 10000


func main() {
	r := mux.NewRouter()
	//HandleFunc("/", handler)

	//apiSubrouter := r.PathPrefix("/api").Subrouter()
	r.PathPrefix("/static/").Handler(http.FileServer(http.Dir("./files/")))
	createSubrouter := r.PathPrefix("/c/").Subrouter()
	createSubrouter.HandleFunc("/{url:.*}", create)
	r.HandleFunc("/{hash:.*}", root)

	http.Handle("/", r)
	http.ListenAndServe(":80", nil)

}
