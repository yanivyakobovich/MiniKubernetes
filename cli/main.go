package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/mercadolibre/golang-restclient/rest"
	"gopkg.in/yaml.v2"
)

const SERVER_URL = "http://localhost:1234"

type ConfigurationAgent struct {
	Configuration Configuration
	AgentArray    []Agent
}

type Agent struct {
	MapContainerName map[string]*Container
	Port             int
	Active           bool
}

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

func printAgentStatus(agents []Agent) {
	if 0 == len(agents) {
		fmt.Printf("No agents in system")
		return
	}

	var agentStatus string
	for i, agent := range agents {
		if agent.Active {
			agentStatus = "active"
		} else {
			agentStatus = "not active"
		}
		fmt.Printf("Agent %d on port: %d is %s \n", i, agent.Port, agentStatus)
	}
}

func printConfigurationWithAgentDivision(configurationAgent ConfigurationAgent) {
	fmt.Printf("Configuration name: %s \n", configurationAgent.Configuration.Name)
	fmt.Printf("Image %s\n", configurationAgent.Configuration.Image)
	fmt.Printf("Amount %d\n", configurationAgent.Configuration.Amount)

	for _, agent := range configurationAgent.AgentArray {
		fmt.Printf("agent on port: %d is responisble to the containers below: \n", agent.Port)
		for containerName, container := range agent.MapContainerName {
			if container.ConfigurationName == configurationAgent.Configuration.Name {
				fmt.Printf("container name %s\n", containerName)
			}
		}
	}
}

func printAllConfigurationStatus(configurationArray []Configuration) {
	if 0 == len(configurationArray) {
		fmt.Println("No configuration in the system")
		return

	}

	for _, configuration := range configurationArray {
		fmt.Printf("configuration name: %s, amount: %d , image: %s \n",
			configuration.Name, configuration.Amount, configuration.Image)
	}
}

func (c *Configuration) getContentFromYAML(fileName string) *Configuration {
	yamlFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

func envStatus() {
	var rb rest.RequestBuilder
	rb.DisableTimeout = true

	resp := rb.Get(SERVER_URL + "/envStatus")
	if resp.Err != nil {
		fmt.Println(resp.Err)
		return
	}

	if resp.Response.StatusCode == http.StatusCreated {
		var configurationArray []Configuration
		err := resp.FillUp(&configurationArray)
		if err != nil {
			log.Fatal(fmt.Sprintf("Json fill up failed. Error: %s", err.Error()))
		}

		printAllConfigurationStatus(configurationArray)
	}

	resp.Body.Close()
}

func create(Info Configuration) {
	var rb rest.RequestBuilder
	rb.DisableTimeout = true

	resp := rb.Post(fmt.Sprintf("%s/create", SERVER_URL), Info)
	if resp.Err != nil {
		fmt.Println(resp.Err)
		return
	}

	stringRespond(resp)
}

func delete(name string) {
	var rb rest.RequestBuilder
	rb.DisableTimeout = true

	resp := rb.Post(fmt.Sprintf("%s/delete", SERVER_URL), name)
	if resp.Err != nil {
		fmt.Println(resp.Err)
		return
	}

	stringRespond(resp)
}

func update(Info Configuration) {
	var rb rest.RequestBuilder
	rb.DisableTimeout = true

	resp := rb.Post(fmt.Sprintf("%s/update", SERVER_URL), Info)
	if resp.Err != nil {
		fmt.Println(resp.Err)
		return
	}

	stringRespond(resp)
}

func envNameStatus(name string) {
	var rb rest.RequestBuilder
	rb.DisableTimeout = true

	resp := rb.Post(SERVER_URL+"/envNameStatus", name)
	if resp.Err != nil {
		fmt.Println(resp.Err)
		return
	}

	if resp.Response.StatusCode == http.StatusCreated {
		var configurationAgent ConfigurationAgent
		err := resp.FillUp(&configurationAgent)
		if err != nil {
			log.Fatal(fmt.Sprintf("Json fill up failed. Error: %s", err.Error()))
		}

		printConfigurationWithAgentDivision(configurationAgent)

	} else {

		var errorMessage string
		err := resp.FillUp(&errorMessage)
		if err != nil {
			log.Fatal(fmt.Sprintf("Json fill up failed. Error: %s", err.Error()))
		}
		fmt.Printf("%s \n", errorMessage)
	}

	resp.Body.Close()
}

func agentsStatus() {
	var rb rest.RequestBuilder
	rb.DisableTimeout = true

	resp := rb.Get(SERVER_URL + "/agentsStatus")
	if resp.Err != nil {
		fmt.Println(resp.Err)
		return
	}

	if resp.Response.StatusCode == http.StatusCreated {
		var agents []Agent
		err := resp.FillUp(&agents)
		if err != nil {
			log.Println(fmt.Sprintf("Json fill up failed. Error: %s", err.Error()))
		}
		printAgentStatus(agents)
	}

	resp.Body.Close()
}

func printHelp() {
	fmt.Println("Please enter valid request, you are only allowed the commands below:")
	fmt.Println("create <YAML file path>")
	fmt.Println("delete <YAML file path>")
	fmt.Println("update <YAML file path>")
	fmt.Println("Show env status")
	fmt.Println("Show env <Name> status")
	fmt.Println("Show agent status")
}

func stringRespond(resp *rest.Response) {
	var message string
	err := resp.FillUp(&message)
	if err != nil {
		log.Println("Json fill up failed. Error: " + err.Error())
		return
	}

	fmt.Println(message)
	resp.Body.Close()
}

func doAction(params []string) {
	if len(params) == 2 {
		switch params[0] {
		case "create":
			yamlStruct := Configuration{}
			yamlStruct.getContentFromYAML(params[1])
			create(yamlStruct)
			return

		case "delete":
			delete(params[1])
			return

		case "update":
			yamlStruct := Configuration{}
			yamlStruct.getContentFromYAML(params[1])
			update(yamlStruct)
			return
		}
	}

	if len(params) == 3 && params[0] == "Show" && params[2] == "status" {
		if params[1] == "env" {
			envStatus()
			return
		}
		if params[1] == "agent" {
			agentsStatus()
			return
		}
	}

	if len(params) == 4 && params[0] == "Show" && params[1] == "env" && params[3] == "status" {
		envNameStatus(params[2])
		return
	}

	printHelp()
}

func main() {
	argsWithoutProg := os.Args[1:]
	doAction(argsWithoutProg)
}
