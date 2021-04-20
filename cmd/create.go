package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/go-yaml/yaml"
	spec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/raphaelreyna/emmy/internal/conf"
	comptop "github.com/raphaelreyna/go-comptop"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:        "create [file.yaml]",
	Short:      "Create the network topology described in the passed in file.",
	SuggestFor: []string{"teardown"},
	RunE:       create,
	Args:       cobra.ExactArgs(1),
}

func create(cmd *cobra.Command, args []string) error {
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

	if err = _create(cmd.Context(), cli, &conf); err != nil {
		cmd.SilenceUsage = true
	}

	return err
}

var defaultPlatform *spec.Platform = &spec.Platform{
	Architecture: "amd64",
	OS:           "linux",
}

func _create(ctx context.Context, cli *client.Client, conf *conf.ComplexConf) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	// Create a simplicial complex and fill it
	c := comptop.Complex{}
	c.NewSimplicesWithData(_dataProvider(ctx, cli, conf), conf.Bases()...)

	// Get the ID of the bridge network
	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return err
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
			if network.AppliesToBase(base) {
				nc.Internal = !network.External
				break
			}
		}

		// Create the network
		netName := fmt.Sprintf("%s_%d-simplex_%d", conf.Name, smplx.Dim(), smplx.Index())
		resp, err := cli.NetworkCreate(ctx, netName, nc)
		if err != nil {
			return err
		}

		// Add the containers to the network
		for _, v := range smplx.Faces(0).Slice() {
			cid := v.Data.(string)
			if err := cli.NetworkConnect(ctx, resp.ID, cid, nil); err != nil {
				return err
			}
		}
	}

	// Range over the containers, disconnect the bridge network if needed, and start the container
	containers := c.ChainGroup(0).Simplices()
	for _, smplx := range containers {
		cid := smplx.Data.(string)

		if !conf.Bridging[int(smplx.Index())] {
			if err := cli.NetworkDisconnect(ctx, bridgeID, cid, true); err != nil {
				return err
			}
		}

		if err := cli.ContainerStart(ctx, cid, types.ContainerStartOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func _dataProvider(ctx context.Context, cli *client.Client, cc *conf.ComplexConf) comptop.DataProvider {
	return func(d comptop.Dim, idx comptop.Index, b comptop.Base) interface{} {
		if d != 0 {
			return nil
		}

		resp, err := cli.ContainerCreate(ctx,
			cc.ContainerConfig(int(idx)),
			&container.HostConfig{
				AutoRemove: true,
			}, nil, defaultPlatform,
			fmt.Sprintf("%s_%d", cc.Name, idx),
		)

		if err != nil {
			panic(err)
		}

		return resp.ID
	}
}
