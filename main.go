package main

import (
	"net/http"
	"encoding/json"
	"log"
	"fmt"
	"io"
	"time"
)

var urlMap = map[string][]string{
	"1": {"https://api.hallowedvisions.com"},
}

type JsonError struct {
	msg string
	code string
}

func SendError (w http.ResponseWriter, r *http.Request, status int, jsonError JsonError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	jsonMap := map[string]interface{} {
		"msg": jsonError.msg,
		"code": jsonError.code,
	}

	data, _ := json.Marshal(jsonMap)

	w.Write(data)
}

func RouteProxy (w http.ResponseWriter, r *http.Request)  {
	path := r.URL.String()

	serverId, ok := r.Header["Server-Id"]
	if !ok {
		SendError(w, r, 400, JsonError{"Header Missing", "INVALID_URL"})
		return
	}

	serverUrl, ok := urlMap[serverId[0]]

	if !ok {
		SendError(w, r, 400, JsonError{"Invalid Url", "INVALID_URL"})
		return
	}

	baseUrl := fmt.Sprintf("%s%s", serverUrl[0], path)

	
	req, _ := http.NewRequest(r.Method, baseUrl, r.Body)
	req.Header = r.Header

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		SendError(w, r, 500, JsonError{"Internal Server Error", "INVALID_SERVER"})
		return
	}

	defer resp.Body.Close()

	for header, values := range resp.Header {
		for _, subValue := range values {
			w.Header().Add(header, subValue)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	w.Write([]byte("Success"))

}

type ResponseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *ResponseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)

}

func logginMiddleware (next http.Handler) http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		scheme := "http"

		if r.TLS != nil {
			scheme = "https"
		}


		fmt.Printf("Request at %s:\n", time.Now().Format("02 Jan 2006 03:04PM"))
		fmt.Printf("	Incoming: %s://%s%s\n", scheme, r.Host, r.URL.String())
		fmt.Printf("	%s %s\n", r.Method, r.RequestURI)

		start := time.Now()
		responseRecorder := ResponseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(&responseRecorder, r)

		duration := time.Since(start)
		fmt.Printf("	Status: %d %dms", responseRecorder.statusCode, duration.Milliseconds())
		fmt.Println()
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", RouteProxy)
	fmt.Println("Server Running on 8080")
	fmt.Println("")
	log.Fatal(http.ListenAndServe(":8080", logginMiddleware(mux)))

}

