package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/mercadolibre/golang-restclient/rest"
)

const SERVER_PORT = "127.0.0.1:1234"
const PORT = "1234"
const BASE_URL = "http://localhost:"
const AGENT_PATH = "../agent/agent"
const AGENTS_AMOUNTS = 2
const PATH_MAP = "server/mapConfigurationToAgents.json"
const PATH_AGENTARRAY = "server/agentsArray.json"

var agentsArray []*Agent
var mapConfigurationToAgents map[string]*ConfigurationAgent

func generateAvilablePort() net.Listener {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Using port:", listener.Addr().(*net.TCPAddr).Port)
	return listener

}

func respondWithError(response http.ResponseWriter, code int, message string) {
	log.Println("error happened: " + message)
	respondWithJSON(response, code, "error :"+message)
}

func respondWithJSON(responseHTTP http.ResponseWriter, code int, payload interface{}) {
	responseJSON, _ := json.Marshal(payload)

	responseHTTP.Header().Set("Content-Type", "application/json")
	responseHTTP.WriteHeader(code)
	responseHTTP.Write(responseJSON)
}

func envStatusEndpoint(responseHTTP http.ResponseWriter, r *http.Request) {
	configurationArray := make([]Configuration, 0)
	for _, configurationAgent := range mapConfigurationToAgents {
		configurationArray = append(configurationArray, *configurationAgent.Configuration)
	}

	log.Println("show env status request")
	respondWithJSON(responseHTTP, http.StatusCreated, configurationArray)
}

func agentsStatusEndPoint(responseHTTP http.ResponseWriter, r *http.Request) {
	log.Println("show agents status request")
	respondWithJSON(responseHTTP, http.StatusCreated, agentsArray)
}

func agentPortEndPoint(responseHTTP http.ResponseWriter, r *http.Request) {
	var portAgent string

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&portAgent); err != nil {
		respondWithError(responseHTTP, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	var err error
	var port int

	if port, err = strconv.Atoi(portAgent); err != nil {
		log.Println(err)
		return
	}

	agentAlreadyExisted := false
	for _, agent := range agentsArray {
		if agent.Active == false {
			agent.Active = true
			agent.Port = port
			agentAlreadyExisted = true
			log.Printf("agent %d was replaced\n", port)
		}
	}

	if !agentAlreadyExisted {
		agentToAdd := new(Agent)
		agentToAdd.Port = port
		agentToAdd.MapContainerName = make(map[string]*Container)
		agentToAdd.Active = true

		agentsArray = append(agentsArray, agentToAdd)

		log.Printf("agent %d created\n", port)

	}
	writeDataToJSON()
	respondWithJSON(responseHTTP, http.StatusCreated, portAgent)
}

func envNameStatusEndPoint(responseHTTP http.ResponseWriter, r *http.Request) {
	var configurationName string
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&configurationName); err != nil {
		respondWithError(responseHTTP, http.StatusBadRequest, "Invalid request payload")
		return
	}

	log.Printf("env %s status request \n", configurationName)
	var status *ConfigurationAgent

	if getStatusByConfiguration(configurationName, &status) {
		log.Println(status)
		respondWithJSON(responseHTTP, http.StatusCreated, status)
	} else {
		respondWithError(responseHTTP, http.StatusBadRequest, "The configuration doesn't exists")
	}

	defer r.Body.Close()
}

func deleteEndPoint(responseHTTP http.ResponseWriter, request *http.Request) {

	var configurationNameToDelete string
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&configurationNameToDelete); err != nil {
		respondWithError(responseHTTP, http.StatusBadRequest, "Invalid request payload")
		return
	}

	log.Printf("delete %s request", configurationNameToDelete)
	defer request.Body.Close()

	deleteSucceed, errorOfDelete := removeConfiguration(configurationNameToDelete, 1)
	writeDataToJSON()
	if deleteSucceed {
		messesgeSuccess := fmt.Sprintf("configuration %s been deleted", configurationNameToDelete)
		respondWithJSON(responseHTTP, http.StatusCreated, messesgeSuccess)
	} else {
		respondWithError(responseHTTP, http.StatusBadRequest, errorOfDelete)
	}

}

func createEndpoint(responseHTTP http.ResponseWriter, r *http.Request) {

	var configuration Configuration
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&configuration); err != nil {
		respondWithError(responseHTTP, http.StatusBadRequest, "Invalid request payload")
		return
	}

	containersSucceed, errorMessege := createConfigurationToAgents(&configuration, 1)

	writeDataToJSON()
	if containersSucceed {
		respondWithJSON(responseHTTP, http.StatusCreated, "Containers created")
	} else {
		respondWithError(responseHTTP, http.StatusBadRequest, errorMessege)
	}
	defer r.Body.Close()
}

func updateEndpoint(responseHTTP http.ResponseWriter, r *http.Request) {
	var configuration Configuration
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&configuration); err != nil {
		respondWithError(responseHTTP, http.StatusBadRequest, "Invalid request payload")
		return
	}

	fmt.Printf("update %s request", configuration.Name)
	defer r.Body.Close()
	updataSucceed, errorMessage := update(&configuration)

	writeDataToJSON()
	if updataSucceed {
		respondWithJSON(responseHTTP, http.StatusCreated, "Update complete")
	} else {
		respondWithError(responseHTTP, http.StatusBadRequest, errorMessage)
	}
}

//Server functions to Agent

func runContainer(container Container, port string) *rest.Response {
	log.Println("container send to agent request")
	var rb rest.RequestBuilder
	rb.DisableTimeout = true
	resp := rb.Post(fmt.Sprintf("%s%s/runContainer", BASE_URL, port), container)
	return resp
}

func deleteContainer(containerName string, port string) *rest.Response {
	resp := rest.Post(fmt.Sprintf("%s%s/deleteContainer", BASE_URL, port), containerName)
	return resp
}

func writeDataToJSON() {
	file, _ := json.MarshalIndent(mapConfigurationToAgents, "", " ")
	_ = ioutil.WriteFile("mapConfigurationToAgents.json", file, 0644)
	file, _ = json.MarshalIndent(agentsArray, "", " ")
	_ = ioutil.WriteFile("agentsArray.json", file, 0644)
}

// read the data from json files

// func readJSONToStructs(variable interface{}, path string) bool {

// 	absPath, _ := filepath.Abs(fmt.Sprintf("../%s", path))

// 	if _, err := os.Stat(absPath); err == nil {

// 		configFile, err := os.Open(absPath)
// 		if err != nil {
// 			log.Println("opening config file", err.Error())
// 			return false
// 		}

// 		jsonParser := json.NewDecoder(configFile)
// 		if err = jsonParser.Decode(&variable); err != nil {
// 			log.Println("parsing config file", err.Error())
// 			return false
// 		}
// 		if variable == nil {
// 			return false
// 		}

// 		return true
// 	}

// 	return false
// }

// func initalizeParams() {
// 	if !readJSONToStructs(mapConfigurationToAgents, PATH_MAP) {
// 		mapConfigurationToAgents = make(map[string]*ConfigurationAgent)
// 	}

// 	if !readJSONToStructs(agentsArray, PATH_AGENTARRAY) {
// 		createAgent()
// 	}
// }

func main() {
	mapConfigurationToAgents = make(map[string]*ConfigurationAgent)
	createAgents()

	log.SetFlags(log.LstdFlags | log.Llongfile)

	r := mux.NewRouter()
	api := r.PathPrefix("/").Subrouter()

	api.HandleFunc("/envStatus", envStatusEndpoint).Methods(http.MethodGet)
	api.HandleFunc("/agentsStatus", agentsStatusEndPoint).Methods(http.MethodGet)
	api.HandleFunc("/agentPort", agentPortEndPoint).Methods(http.MethodPost)
	api.HandleFunc("/create", createEndpoint).Methods(http.MethodPost)
	api.HandleFunc("/delete", deleteEndPoint).Methods(http.MethodPost)
	api.HandleFunc("/update", updateEndpoint).Methods(http.MethodPost)
	api.HandleFunc("/envNameStatus", envNameStatusEndPoint).Methods(http.MethodPost)

	go func() {
		for true {
			time.Sleep(10 * time.Second)
			for _, agent := range agentsArray {

				resp, err := http.Get(fmt.Sprintf("%s%d/isAgentActive", BASE_URL, agent.Port))
				if err != nil {
					log.Printf("agent with port=%d is not responding\n", agent.Port)
					agent.Active = false
					createAgent()
					continue
				}
				resp.Body.Close()

				if resp.StatusCode == http.StatusCreated {
					agent.Active = true
					log.Printf("agent with port=%d is ALIVE\n", agent.Port)
				}
			}
		}
	}()

	log.Println("Server is waiting for connections on port " + PORT)

	portToListen := fmt.Sprintf(":" + PORT)
	err := http.ListenAndServe(portToListen, r)
	if err != nil {
		log.Fatal(err)
	}
}
