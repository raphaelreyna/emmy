package main

import (
	"context"
	"fmt"
	"log"
	"regexp"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func teardown(ctx context.Context, cli *client.Client, name string) error {
	nn := ""

	// Remove containers
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return err
	}

	regex := regexp.MustCompile(fmt.Sprintf("^/%s_[0-9]+$", name))
	for _, container := range containers {
		inComplex := false
		for _, n := range container.Names {
			if regex.MatchString(n) {
				inComplex = true
				nn = n[1:]
				break
			}
		}
		if !inComplex {
			continue
		}

		err := cli.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
			Force: true,
		})

		if err != nil {
			fmt.Printf("error removing container %s: %v\n", nn, err)
		} else {
			log.Printf("removed container: %s\n", nn)
		}
	}

	// Remove networks
	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return err
	}

	regex = regexp.MustCompile(fmt.Sprintf("^%s_[0-9]+-simplex_[0-9]+$", name))
	for _, network := range networks {
		if regex.MatchString(network.Name) {
			nn = network.Name
		} else {
			continue
		}

		if err := cli.NetworkRemove(ctx, network.ID); err != nil {
			fmt.Printf("error removing network %s: %v\n", nn, err)
		} else {
			log.Printf("removed network: %s\n", nn)
		}
	}

	return nil
}
