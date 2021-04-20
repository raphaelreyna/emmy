package cmd

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/raphaelreyna/emmy/internal/print"
	comptop "github.com/raphaelreyna/go-comptop"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(analyzeCmd)
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze the topological properties of Docker networks.",
	RunE:  analyze,
}

func analyze(cmd *cobra.Command, args []string) error {
	// Create a Docker client
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	if err = _analyze(cmd.Context(), cli); err != nil {
		cmd.SilenceUsage = true
	}

	return err
}

func _analyze(ctx context.Context, cli *client.Client) error {
	opts := types.NetworkListOptions{}

	networks, err := cli.NetworkList(ctx, opts)
	if err != nil {
		return err
	}

	// Enumerate the containers we encounter
	id := comptop.Index(0)
	verts := map[string]comptop.Index{}
	idx := func(s string) comptop.Index {
		if v, exists := verts[s]; exists {
			return v
		}

		verts[s] = id
		id++

		return verts[s]
	}

	// Compute bases from network info
	bases := []comptop.Base{}
	for _, network := range networks {
		network, err = cli.NetworkInspect(ctx, network.ID, types.NetworkInspectOptions{})
		if err != nil {
			return err
		}

		if len(network.Containers) == 0 {
			continue
		}

		b := comptop.Base{}
		for _, container := range network.Containers {
			cidx := idx(container.Name)
			b = append(b, cidx)
		}

		bases = append(bases, b)
	}

	// Is there enough to keep going?
	if len(bases) < 1 {
		return nil
	}

	// Invert verts map
	vertsInverse := map[comptop.Index]string{}
	for k, v := range verts {
		vertsInverse[v] = k
	}

	// Create the sinmplicial complex
	c := comptop.Complex{}
	c.NewSimplices(bases...)

	// Compute and print out info
	print.Complex(&c, vertsInverse)

	return nil
}
