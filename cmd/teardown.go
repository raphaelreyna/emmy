package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/go-yaml/yaml"
	"github.com/raphaelreyna/emmy/internal/conf"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(teardownCmd)
}

var teardownCmd = &cobra.Command{
	Use:   "teardown [file.yaml]",
	Short: "Teardown the networks and containers described in the passed in file.",
	Aliases: []string{
		"rm", "destroy",
	},
	SuggestFor: []string{"create"},
	RunE:       teardown,
	Args:       cobra.ExactArgs(1),
}

func teardown(cmd *cobra.Command, args []string) error {
	// Create a Docker client
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	// Parse conf file
	fileName := args[len(args)-1]
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0444)
	if err != nil {
		return err
	}
	conf := conf.ComplexConf{}
	if err := yaml.NewDecoder(file).Decode(&conf); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}

	if err = _teardown(cmd.Context(), cli, conf.Name); err != nil {
		cmd.SilenceUsage = true
	}

	return err
}

func _teardown(ctx context.Context, cli *client.Client, name string) error {
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
