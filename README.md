## Mini Kubernetes 

### Compilation

1. install the following dependencies:

   ```
   agent/main.go-	"github.com/docker/docker/api/types"
   agent/main.go-	"github.com/docker/docker/api/types/container"
   agent/main.go-	"github.com/docker/docker/api/types/filters"
   agent/main.go-	"github.com/docker/docker/client"
   agent/main.go-	"github.com/gorilla/mux"
   agent/main.go-	"github.com/mercadolibre/golang-restclient/rest"

   cli/main.go-	"github.com/mercadolibre/golang-restclient/rest"
   cli/main.go-	"gopkg.in/yaml.v2"

   server/main.go-	"github.com/gorilla/mux"
   server/main.go-	"github.com/mercadolibre/golang-restclient/rest"
   ```

2. Compile using:

`./compile.sh`

### Server

Start the server by:

`cd ./server; ./server`

The server would listen on port 1234

### CLI

Usage: (you must be in `cli` directory)

`cd cli; ./cli <command>`

The CLI has 6 commands:

1. `create <YAML file path> `: send configuration command to the server
2. `delete <YAML file path>`
3. `update <YAML file path>`
4. `Show env status`
5. `Show env <Name> status`
6. `Show agent status`

Assumptions:
1. The server is listening on port 1234
2. Each container has a terminal at `/bin/sh`
3. On `update <YAML file path>` , if the image is not found at Docker hub, all the running containers will be removed (from all running agents)

## 


