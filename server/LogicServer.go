package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strconv"
	"os"
)

type ConfigurationAgent struct {
	Configuration *Configuration
	AgentArray    []*Agent
}

type Configuration struct {
	Name   string `yaml:"Name"`
	Amount int    `yaml:"Amount"`
	Image  string `yaml:"Image"`
}

type Agent struct {
	MapContainerName map[string]*Container
	Port             int
	Active           bool
}

type Container struct {
	Index             int
	ConfigurationName string
	Image             string
}

func containerName(container *Container) string {
	return container.ConfigurationName + strconv.Itoa(container.Index)
}

func (container *Container) checkIndexCorrectness(startIndex int, endIndex int) bool {
	return startIndex <= container.Index
}

func removeConfiguration(configurationName string, containerStartIndex int) (bool, string) {
	allSucceed := true
	errorReturn := ""
	if configurationAgent, ok := mapConfigurationToAgents[configurationName]; ok {

		for i := 0; i < len(configurationAgent.AgentArray); i++ {
			agent := configurationAgent.AgentArray[i]

			for j := containerStartIndex; j <= configurationAgent.Configuration.Amount; j++ {
				containerNameToCheck := configurationName + strconv.Itoa(j)

				if _, ok := agent.MapContainerName[containerNameToCheck]; ok {
					resp := deleteContainer(containerNameToCheck, strconv.Itoa(agent.Port))

					if resp.StatusCode == http.StatusCreated {
						delete(agent.MapContainerName, containerNameToCheck)
						log.Printf("container %s deleted ", containerNameToCheck)
					} else {
						allSucceed = false
						errorReturn = fmt.Sprintf("delete %s container failed", containerNameToCheck)
						log.Printf("delete %s container failed", containerNameToCheck)
					}
				}
			}
		}

		if allSucceed {
			if containerStartIndex == 1 {
				// means the configuration needs to be delete from the main map
				delete(mapConfigurationToAgents, configurationName)
			}

			return allSucceed, errorReturn
		}
		return allSucceed, errorReturn

	}
	return false, "No such configuration in the system"
}

func sortAgentsByContainerAmount(agentArray []*Agent) {
	sort.SliceStable(agentArray, func(i, j int) bool {
		return len(agentArray[i].MapContainerName) < len(agentArray[j].MapContainerName)
	})
}

func checkCreateParamValidity(configuration *Configuration, startIndexContainer int) (bool, string) {
	isValid, errorMessage := checkAmountImageNameValdity(configuration)
	if !isValid {
		return false, errorMessage
	}

	_, ok := mapConfigurationToAgents[configuration.Name]
	if startIndexContainer == 1 {
		if ok {
			//Intended to create new configuration but it already in our system
			return false, "the configuration already exists in the system"
		}
		return true, ""

	} else if ok {
		return true, ""
	}

	// which mean 1 < start index than it must be update call, therefore the configuration must be in the system
	return false, "the configuration doesn't exists in the system"
}

func createConfigurationToAgents(configuration *Configuration, startIndexContainer int) (bool, string) {
	paramValidity, errorMessage := checkCreateParamValidity(configuration, startIndexContainer)

	if !paramValidity {
		return false, errorMessage
	}

	sortAgentsByContainerAmount(agentsArray)

	if len(agentsArray) == 0 {
		return false, "No agents available"
	}

	i := 0
	allErrorMessagesFromAgents := ""
	allContainersSucceed := true
	for i+startIndexContainer <= configuration.Amount {

		agentIndex := i % len(agentsArray)
		containersSucceed, errorCommand := commandToAgentByConfiguration(configuration, agentIndex, startIndexContainer+i)
		allErrorMessagesFromAgents = fmt.Sprintf("%s \n %s", allErrorMessagesFromAgents, errorCommand)

		if !containersSucceed {
			log.Printf(errorCommand)
			containersSucceed = false
		}

		i++
		allContainersSucceed = allContainersSucceed && containersSucceed
	}

	if !allContainersSucceed {
		delete(mapConfigurationToAgents, configuration.Name)
	}

	return allContainersSucceed, allErrorMessagesFromAgents
}

func createAgents() {
	agentsArray = make([]*Agent, 0)
	for i := 0; i < AGENTS_AMOUNTS; i++ {
		createAgent()
	}
}

func createAgent() {
	cmd := exec.Command(AGENT_PATH, PORT)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("start cmd agent with pid=%d started \n", cmd.Process.Pid)
}

func commandToAgentByConfiguration(configuration *Configuration, indexAgent int, indexContainer int) (bool, string) {

	if indexContainer == 1 {
		_, ok := mapConfigurationToAgents[configuration.Name]
		// creation of the configuration in the map configuration to agents
		if ok == false {
			configurationAgent := new(ConfigurationAgent)
			configurationAgent.AgentArray = make([]*Agent, 0)
			configurationAgent.Configuration = configuration
			mapConfigurationToAgents[configuration.Name] = configurationAgent
		}
	}

	//create the container struct for the agent
	var containerToSend *Container
	containerToSend = new(Container)
	agentPort := strconv.Itoa(agentsArray[indexAgent].Port)
	containerToSend.Index = indexContainer
	containerToSend.ConfigurationName = configuration.Name
	containerToSend.Image = configuration.Image

	resp := runContainer(*containerToSend, agentPort)

	if resp.StatusCode == http.StatusCreated {

		// container created then update the server database
		updateAllDataByContainer(containerToSend, agentsArray[indexAgent])
		log.Printf("container created by agent on port %s", agentPort)
		return true, fmt.Sprintf("container created by agent on port %s", agentPort)
	}

	log.Printf("container failed by agent on port %s", agentPort)

	defer resp.Body.Close()
	return false, fmt.Sprintf("container failed by agent on port %s", agentPort)

}

func updateAllDataByContainer(container *Container, agent *Agent) {
	//update agent
	agent.MapContainerName[containerName(container)] = container

	//update map
	if val, ok := mapConfigurationToAgents[container.ConfigurationName]; ok {

		//configuration exists in map
		i := checkAgentExists(agent, val.AgentArray)

		// the agent is not in the configuration map , need to add the agent
		if i == -1 {
			val.AgentArray = append(val.AgentArray, agent)
		}
	}
}

func getStatusByConfiguration(configurationName string, status **ConfigurationAgent) bool {
	if val, ok := mapConfigurationToAgents[configurationName]; ok {
		*status = val
		return true
	}
	return false
}

func checkAgentExists(agent *Agent, agentArrayInConfigurationMap []*Agent) int {
	for i := 0; i < len(agentArrayInConfigurationMap); i++ {
		if agentsArray[i].Port == agent.Port {
			return i
		}
	}
	return -1
}

//function for debugging
// func printAllDB() {
// 	fmt.Println("new print")
// 	for configurationName, configurationAgent := range mapConfigurationToAgents {
// 		fmt.Printf("configurationName %s \n", configurationName)
// 		if configurationAgent.Configuration != nil {
// 			fmt.Printf("Image %s\n", configurationAgent.Configuration.Image)
// 			fmt.Printf("Amount %d\n", configurationAgent.Configuration.Amount)
// 		}
// 		if 0 < len(configurationAgent.AgentArray) {
// 			for i := 0; i < len(configurationAgent.AgentArray); i++ {
// 				for containerName, container := range configurationAgent.AgentArray[i].MapContainerName {
// 					fmt.Printf("container name %s\n", containerName)
// 					fmt.Println(container)
// 				}
// 			}
// 		}
// 	}
// }

func checkAmountImageNameValdity(configuration *Configuration) (bool, string) {
	if configuration.Image == "" {
		return false, "there is no image in your YAML file"
	}

	if configuration.Name == "" {
		return false, "there is no name in your YAML file"
	}

	if configuration.Amount < 0 {
		return false, "amount must be above zero"
	}

	return true, ""
}

func update(configuration *Configuration) (bool, string) {
	isValid, errorMessage := checkAmountImageNameValdity(configuration)
	if !isValid {
		return false, errorMessage
	}

	if val, ok := mapConfigurationToAgents[configuration.Name]; ok {
		if configuration.Image != val.Configuration.Image {

			// Different image , all the containers that belongs to the old configuration have to delete
			updateSucceed, errorMessage := removeConfiguration(configuration.Name, 1)

			if updateSucceed {
				updateSucceed, errorMessage := createConfigurationToAgents(configuration, 1)
				return updateSucceed, errorMessage
			}

			return updateSucceed, errorMessage
		}

		//same image , need to check the difference in the amount
		if val.Configuration.Amount < configuration.Amount {

			// need to create more containers
			oldAmount := val.Configuration.Amount
			val.Configuration.Amount = configuration.Amount
			updateSucceed, errorMessage := createConfigurationToAgents(val.Configuration, oldAmount+1)
			return updateSucceed, errorMessage
		}

		//need to delete containers
		updateSucceed, errorMessage := removeConfiguration(configuration.Name, configuration.Amount+1)
		val.Configuration.Amount = configuration.Amount
		return updateSucceed, errorMessage

	}

	return false, "No such configuration"
}
