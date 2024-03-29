/*
**********************************
***** SAMPLE YAML CONF FILE ******
**********************************

name: example
image: alpine
networks:
  - containers: [0, 1, 2]
    label: big
    image: alpine
  - containers: [2, 3]
    external: true
    enableIPV6: false
    label: small
containers:
  - image: busybox
    containers: [0, 1]
  - bridge: true
    cmd: ["ash"]
    containers:
      - 1
*/
package conf

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	comptop "github.com/raphaelreyna/go-comptop"
)

type NetworkConf struct {
	Containers []int  `yaml:"containers"`
	External   bool   `yaml:"external"`
	EnableIPV6 bool   `yaml:"enableIPV6"`
	Driver     string `yaml:"driver"`
	Label      string `yaml:"label"`
}

func (nc *NetworkConf) AppliesToContainer(idx int) bool {
	for _, i := range nc.Containers {
		if i == idx {
			return true
		}
	}
	return false
}

func (nc *NetworkConf) AppliesToBase(b comptop.Base) bool {
	for _, bb := range b {
		inNetwork := false
		for _, cidx := range nc.Containers {
			if bb == comptop.Index(cidx) {
				inNetwork = true
				break
			}
		}

		if !inNetwork {
			return false
		}
	}

	return true
}

type ContainerConf struct {
	Image      string   `yaml:"image"`
	Bridge     bool     `yaml:"bridge"`
	Containers []int    `yaml:"containers"`
	Networks   []string `yaml:"network"`
	Cmd        []string `yaml:"cmd"`
	Label      string   `yaml:"label"`
}

func (cc *ContainerConf) appliesToContainer(idx int) bool {
	for _, cidx := range cc.Containers {
		if cidx == idx {
			return true
		}
	}

	return false
}

func (cc *ContainerConf) appliesToNetwork(label string) bool {
	for _, l := range cc.Networks {
		if l == label {
			return true
		}
	}

	return false
}

type ComplexConf struct {
	Name            string           `yaml:"name"`
	Image           string           `yaml:"image"`
	NetworkConfs    []*NetworkConf   `yaml:"networks"`
	ContainersConfs []*ContainerConf `yaml:"containers"`

	Bridging map[int]bool `yaml:"-"`
}

func (cc *ComplexConf) ContainerConfig(idx int) *container.Config {
	if cc.Bridging == nil {
		cc.Bridging = map[int]bool{}
	}

	conf := container.Config{
		Image:     cc.Image,
		Tty:       true,
		OpenStdin: true,
	}

	// Look for networks that this container is a part of
	networks := map[*NetworkConf]struct{}{}
	for _, nc := range cc.NetworkConfs {
		if nc.AppliesToContainer(idx) {
			networks[nc] = struct{}{}
			continue
		}
	}

	// Look for container configurations that apply to this idx or any network that its in
	for _, cconf := range cc.ContainersConfs {
		applies := false
		if !cconf.appliesToContainer(idx) {
			for n := range networks {
				if n.Label == "" {
					continue
				}

				if cconf.appliesToNetwork(n.Label) {
					applies = true
					break
				}
			}
		} else {
			applies = true
		}

		if applies {
			if x := cconf.Image; x != "" {
				conf.Image = x
			}

			conf.Cmd = strslice.StrSlice(cconf.Cmd)

			if x := cconf.Label; x != "" {
				conf.Labels = map[string]string{
					"7b-label": x,
				}
			}

			cc.Bridging[idx] = cconf.Bridge
		}
	}

	return &conf
}

func (cc *ComplexConf) Bases() []comptop.Base {
	bases := []comptop.Base{}

	for _, network := range cc.NetworkConfs {
		base := comptop.Base{}
		for _, id := range network.Containers {
			base = append(base, comptop.Index(id))
		}

		bases = append(bases, base)
	}

	return bases
}
