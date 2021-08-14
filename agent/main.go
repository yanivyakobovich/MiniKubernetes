package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
	"github.com/mercadolibre/golang-restclient/rest"
)

const BASE_URL = "http://localhost:"

var agentPort int
var cli *client.Client
var portServer int

type Configuration struct {
	Name   string `yaml:"Name"`
	Amount int    `yaml:"Amount"`
	Image  string `yaml:"Image"`
}

type Container struct {
	Index             int
	ConfigurationName string
	Image             string
}

func generateContainerName(container Container) string {
	return fmt.Sprintf("%s%s", container.ConfigurationName, strconv.Itoa(container.Index))
}

func listenOnFreePort() net.Listener {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	return listener
}

func respondWithJSON(responseHTTP http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	responseHTTP.Header().Set("Content-Type", "application/json")
	responseHTTP.WriteHeader(code)
	responseHTTP.Write(response)
}

func respondWithError(responseHTTP http.ResponseWriter, code int, message string) {
	respondWithJSON(responseHTTP, code, message)
}

func runContainerEndPoint(responseHTTP http.ResponseWriter, r *http.Request) {

	var container Container
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&container); err != nil {
		respondWithError(responseHTTP, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	log.Printf("run container with image %s index %d request \n", container.Image, container.Index)

	containerSucceed := runContainer(container.Image, generateContainerName(container))
	if containerSucceed != true {
		respondWithError(responseHTTP, http.StatusBadRequest, "could not create container")
		return
	}

	respondWithJSON(responseHTTP, http.StatusCreated, "container created")
}

func deleteContainerEndPoint(responseHTTP http.ResponseWriter, r *http.Request) {
	var containerName string
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&containerName); err != nil {
		respondWithError(responseHTTP, http.StatusBadRequest, "Invalid request payload")
		return
	}

	defer r.Body.Close()
	if removeContainerByName(containerName) {
		respondWithJSON(responseHTTP, http.StatusCreated, containerName)
		return
	}

	respondWithError(responseHTTP, http.StatusBadRequest, fmt.Sprintf("agent havent succeed to remove container %s", containerName))
}

func sendPort(portAgent string, baseURL string) {
	resp := rest.Post(baseURL+"/agentPort", portAgent)
	if !(resp.StatusCode == http.StatusCreated) {
		log.Fatal(resp.Err)
	}
}

func agentStatusToServerEndPoint(responseHTTP http.ResponseWriter, r *http.Request) {
	responseHTTP.WriteHeader(http.StatusCreated)
}

func copyFileToContainer(containerID string, hostPath, insideContainerFilename string) {
	cmd := exec.Command("docker", "cp", hostPath, fmt.Sprintf("%s:/%s", containerID, insideContainerFilename))
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func removeContainerByName(containerName string) bool {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Println(err)
		return false
	}

	filters := filters.NewArgs()
	filters.Add("name", "^"+containerName+"$")
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		log.Println(err)
		return false
	}

	for _, container := range containers {
		duration := time.Duration(0)
		err = cli.ContainerStop(context.Background(), container.ID, &duration)
		if err != nil {
			log.Println(err)
			return false
		}
		log.Printf("killed: %s\n", container.ID)

		err = cli.ContainerRemove(context.Background(), container.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			log.Println(err)
			return false
		}
		log.Printf("removed: %s\n", container.ID)
	}

	return true
}

func runContainer(imageName string, name string) bool {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Println(err)
		return false
	}

	reader, err := cli.ImagePull(ctx, "docker.io/library/"+imageName, types.ImagePullOptions{})
	if err != nil {
		log.Println(err)
		return false
	}

	io.Copy(os.Stdout, reader)

	resp, err := cli.ContainerCreate(ctx, &container.Config{

		Image: imageName,
		Cmd:   []string{"/bin/sh", "/init.sh"},
		Tty:   false,
	}, nil, nil, nil, name)
	if err != nil {
		log.Println(err)
		return false
	}

	copyFileToContainer(resp.ID, "../agent/init.sh", "init.sh")
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Println(err)
		return false
	}

	return true
}

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

	portServer, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	api := r.PathPrefix("/").Subrouter()

	api.HandleFunc("/runContainer", runContainerEndPoint).Methods(http.MethodPost)
	api.HandleFunc("/deleteContainer", deleteContainerEndPoint).Methods(http.MethodPost)
	api.HandleFunc("/isAgentActive", agentStatusToServerEndPoint).Methods(http.MethodGet)

	portListener := listenOnFreePort()
	agentPort := strconv.Itoa(portListener.Addr().(*net.TCPAddr).Port)

	sendPort(agentPort, fmt.Sprintf("%s%d", BASE_URL, portServer))

	log.Println("agent mode")
	log.Println(fmt.Sprintf("Waiting for connections on %d", portListener.Addr().(*net.TCPAddr).Port))

	log.Fatal(http.Serve(portListener, r))
}
