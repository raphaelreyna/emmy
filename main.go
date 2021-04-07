package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/go-yaml/yaml"
	spec "github.com/opencontainers/image-spec/specs-go/v1"
	comptop "github.com/raphaelreyna/go-comptop"
)

var defaultPlatform *spec.Platform = &spec.Platform{
	Architecture: "amd64",
	OS:           "linux",
}

var teardownMode bool

func init() {
	flag.BoolVar(&teardownMode, "teardown", false, "Teardown the containers and networks created by the passed in yaml file.")
}

func main() {
	flag.Parse()

	// Load configuration file
	file, err := os.Open(os.Args[len(os.Args)-1])
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Decode configuration into ComplexConf struct
	conf := ComplexConf{}
	if err := yaml.NewDecoder(file).Decode(&conf); err != nil {
		panic(err)
	}

	// Create a Docker client
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	// Are we tearing down or creating?
	ctx := context.Background()
	if teardownMode {
		if err := teardown(ctx, cli, conf.Name); err != nil {
			log.Fatalf("error during teardown: %v", err)
		}
		return
	}

	// Create a simplicial complex and fill it
	c := comptop.Complex{}
	c.NewSimplicesWithData(conf.DataProvider(cli), conf.Bases()...)

	// Get the ID of the bridge network
	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}
	bridgeID := ""
	for _, network := range networks {
		if network.Name == "bridge" {
			bridgeID = network.ID
		}
	}

	// Range over the principle simplices, create a network, attach it to the simplex and attach vertices to the network.
	principleSimplices := c.PrincipleSimplices().Slice()
	for _, smplx := range principleSimplices {
		// Skip containers
		if smplx.Dim() == 0 {
			continue
		}

		// Grab the configuration
		base := smplx.Base()
		nc := types.NetworkCreate{
			Attachable: true,
			Options: map[string]string{
				"com.docker.network.container_interface_prefix": "7b",
				"com.docker.network.bridge.name":                fmt.Sprintf("%s%di%v", conf.Name, len(base), smplx.Index()),
			},
		}
		for _, network := range conf.NetworkConfs {
			if network.appliesToBase(base) {
				nc.Internal = !network.External
				break
			}
		}

		// Create the network
		netName := fmt.Sprintf("%s_%d-simplex_%d", conf.Name, smplx.Dim(), smplx.Index())
		resp, err := cli.NetworkCreate(ctx, netName, nc)
		if err != nil {
			panic(err)
		}

		// Add the containers to the network
		for _, v := range smplx.Faces(0).Slice() {
			cid := v.Data.(string)
			if err := cli.NetworkConnect(ctx, resp.ID, cid, nil); err != nil {
				panic(err)
			}

		}
	}

	// Range over the containers, disconnect the bridge network if needed, and start the container
	containers := c.ChainGroup(0).Simplices()
	for _, smplx := range containers {
		cid := smplx.Data.(string)

		if !conf.bridging[int(smplx.Index())] {
			if err := cli.NetworkDisconnect(ctx, bridgeID, cid, true); err != nil {
				panic(err)
			}
		}

		if err := cli.ContainerStart(ctx, cid, types.ContainerStartOptions{}); err != nil {
			panic(err)
		}
	}

	fmt.Printf("created %d networks\n", len(principleSimplices))
	fmt.Printf("started %d containers\n", len(containers))

	bn := c.BettiNumbers()
	fmt.Printf("\ncomponents: %d\n", bn[0])
	if len(bn) > 1 {
		fmt.Println("hole count:")
		for dim, bbn := range bn[1:] {
			fmt.Printf("\t%d-dimensional: %d\n", dim+1, bbn)
		}
		fmt.Println("")
	}

	basis := c.ChainGroup(1).HomologyGroup().MinimalBasis()
	if basis != nil {
		fmt.Println("minimal paths around 1-dimensional holes:")
		for _, bbasis := range basis {
			visited := map[comptop.Index]struct{}{}
			vs := []string{}
			for _, smplx := range bbasis.Simplices() {
				if _, printed := visited[smplx.Base()[0]]; printed {
					vs = append(vs, fmt.Sprintf("%d", smplx.Base()[1]))
				} else {
					vs = append(vs, fmt.Sprintf("%d", smplx.Base()[0]))
					visited[smplx.Base()[0]] = struct{}{}
				}
			}
			vs = append(vs, vs[0])
			fmt.Printf("\t%s\n", strings.Join(vs, "->"))
		}
	}
}
