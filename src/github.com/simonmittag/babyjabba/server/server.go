package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/rs/zerolog/log"
)

//Version is the server version
var Version string = "unknown"

//ID is a unique server ID
var ID string = "unknown"

//Server struct defines runis the runtime environment for a config.
type Server struct {
	Config
}

//Runtime has access server config
var Runtime *Server

//BootStrap starts up the server from a ServerConfig
func BootStrap() {
	config := new(Config).
		parse("./babyjabba.json").
		reAliasResources().
		addDefaultPolicy()
	Runtime = &Server{Config: *config}
	Runtime.assignHandlers().
		startListening()
}

func (server Server) startListening() {
	log.Info().Msgf("BabyJabba listening on port %d...", server.Port)
	err := http.ListenAndServe(":"+strconv.Itoa(server.Port), nil)
	if err != nil {
		log.Fatal().Err(err).Msgf("unable to start HTTP(S) server on port %d, exiting...", server.Port)
		panic(err.Error())
	}
}

func (server Server) assignHandlers() Server {
	for _, route := range server.Routes {
		if route.Alias == AboutJabba {
			http.HandleFunc(route.Path, serverInformationHandler)
			log.Debug().Msgf("assigned internal server information handler to path %s", route.Path)
		}
	}
	http.HandleFunc("/", proxyHandler)
	log.Debug().Msgf("assigned proxy handler to path %s", "/")
	return server
}

func writeStandardResponseHeaders(response http.ResponseWriter, request *http.Request, statusCode int) {
	response.Header().Set("Server", "BabyJabba "+Version)
	response.Header().Set("Content-Encoding", "identity")
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Cache-control:", "no-store, no-cache, must-revalidate, proxy-revalidate")
	//for TLS response, we set HSTS header
	if Runtime.Mode == "TLS" {
		response.Header().Set("Strict-Transport-Security", "max-age=31536000")
	}
	response.Header().Set("X-server-id", ID)
	response.Header().Set("X-xss-protection", "1;mode=block")
	response.Header().Set("X-content-type-options", "nosniff")
	response.Header().Set("X-frame-options", "sameorigin")
	//copy the X-REQUEST-ID from the request
	response.Header().Set(X_REQUEST_ID, request.Header.Get(X_REQUEST_ID))

	//status code must be last, no headers may be written after this one.
	response.WriteHeader(statusCode)
}

func sendStatusCodeAsJSON(response http.ResponseWriter, request *http.Request, statusCode int) {
	if statusCode >= 299 {
		log.Warn().Int("downstreamResponseCode", statusCode).
			Str("path", request.URL.Path).
			Str(X_REQUEST_ID, request.Header.Get(X_REQUEST_ID)).
			Msgf("request not served")
	}
	writeStandardResponseHeaders(response, request, statusCode)
	response.Write([]byte(fmt.Sprintf("{ \"code\":\"%d\" }", statusCode)))
}
