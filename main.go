package main

import (
	"net/http"
	"encoding/json"
	"log"
	"fmt"
	"io"
	"time"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"os"
)


type ProjectInformation struct {
	name string
	urls []string
}
var urlMap = map[string]ProjectInformation{}

func initUrlMap(urlMap map[string]ProjectInformation){
    urlMap["HV001"] = ProjectInformation{"Sahntek", []string{os.Getenv("SAHNTEK")}}
	urlMap["HV002"] = ProjectInformation{"Hallowed Visions", []string{os.Getenv("HV")}}
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

	serverUrlStruct, ok := urlMap[serverId[0]]

	if !ok {
		SendError(w, r, 400, JsonError{"Invalid Url", "INVALID_URL"})
		return
	}

	baseUrl := fmt.Sprintf("%s%s", serverUrlStruct.urls[0], path)

	req, _ := http.NewRequest(r.Method, baseUrl, r.Body)
	req.Header = r.Header

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		SendError(w, r, 500, JsonError{"Internal Server Error", "INVALID_SERVER"})
		return
	}

	defer resp.Body.Close()

	for header, values := range resp.Header {
		if header == "Access-Control-Allow-Origin" || header == "Access-Control-Allow-Credentials"{
            continue
        }
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


		fmt.Printf("Request @ %s:\n", time.Now().Format("02 Jan 2006 03:04PM"))
		fmt.Printf("	Request Url: %s://%s%s\n", scheme, r.Host, r.URL.String())
		fmt.Printf("	From: %s\n", r.Header.Get("Origin"))
		fmt.Printf("	%s %s\n", r.Method, r.RequestURI)

		start := time.Now()
		responseRecorder := ResponseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(&responseRecorder, r)

		projectId, ok := r.Header["Server-Id"]
		if !ok {
			projectId = []string{"Unknown"}
		}

		serverUrlStruct, ok := urlMap[projectId[0]]
		if !ok {
			serverUrlStruct = ProjectInformation{"Unkown", []string{}}
		}
		
		duration := time.Since(start)
		fmt.Printf("	Project Name: %s\n", serverUrlStruct.name)
		fmt.Printf("	Status: %d %dms", responseRecorder.statusCode, duration.Milliseconds())
		fmt.Println()
	})
}

var allowedOrigins = []string{"https://sahntek.hallowedvisions.com", "https://hallowedvisions.com"}

var allowedHeaders = []string{    
	"Content-Type", 
    "Authorization", 
    "Cookie", 
    "csrfToken",
	"Server-Id",
}
func main() {
	_ = godotenv.Load()
	initUrlMap(urlMap)

	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
		AllowedHeaders: allowedHeaders,
		AllowCredentials: true,
	})
	mux := http.NewServeMux()
	mux.HandleFunc("/", RouteProxy)
	fmt.Printf("Server Running on %s\n", os.Getenv("PORT"))
	fmt.Println("")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), c.Handler(logginMiddleware(mux))))

}

