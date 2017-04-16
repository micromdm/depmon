package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"time"
)

func main() {
	nagC := make(chan status)
	server := &Server{
		DEPStatusNag:   "Unknown",
		depNaginternal: make(map[status]count),
		nagC:           nagC,
	}
	go server.trackStatus()
	http.HandleFunc("/depnag", server.handleDEPStatus)
	http.HandleFunc("/", server.index)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

const authUsername = "depmon"

type Server struct {
	DEPStatusNag string

	// counter for each status
	depNaginternal map[status]count
	nagC           chan status
}

func (s *Server) UpdateFromNag(depStatus status) {
	go func() { s.nagC <- depStatus }()
}

func (s *Server) trackStatus() {
	ticker := time.NewTicker(1 * time.Minute).C
	for {
		select {
		case depStatus := <-s.nagC:
			s.depNaginternal[depStatus]++
		case <-ticker: // reset
			s.DEPStatusNag = getStatus(s.depNaginternal)
			s.depNaginternal = make(map[status]count)
		}
	}
}

// rank known statuses and return the one with the highest count
func getStatus(statusMap map[status]count) string {
	type kv struct {
		Status status
		Count  count
	}

	type kvs []kv
	is := make(kvs, len(statusMap))
	i := 0
	for k, v := range statusMap {
		is[i] = kv{k, v}
		i++
	}
	sort.Slice(is, func(i, j int) bool { return is[j].Count < is[i].Count })
	return is[0].Status.String()
}

type (
	status uint
	count  uint
)

const (
	available status = 1 << iota
	unavailable
)

func (s status) String() string {
	switch s {
	case available:
		return "Available"
	case unavailable:
		return "Unavailable"
	default:
		return "Unknown"
	}
}

func (s *Server) handleDEPStatus(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch status := string(body); status {
	case "success":
		s.UpdateFromNag(available)
	case "34006": // unavailable
		s.UpdateFromNag(unavailable)
	default:
		log.Printf("unknown DEP status reported: %s\n", status)
	}
}

const indexTemplate = `
<html>
<head>
<title>MicroMDM: DEP Status</title>
</head>
<body>
<div class="container">
	<b>DEP Services Status: {{.DEPStatusNag}}</b>
</div>
</body>
</html>
`

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("").Parse(indexTemplate))
	tmpl.Execute(w, s)

}

func authMW(next http.Handler, authToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, password, ok := r.BasicAuth()
		if !ok || password != authToken {
			w.Header().Set("WWW-Authenticate", `Basic realm="depmon"`)
			http.Error(w, "you need to log in", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}
