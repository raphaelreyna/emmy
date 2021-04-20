package print

import (
	"fmt"
	"strings"

	"github.com/raphaelreyna/go-comptop"
)

func Complex(c *comptop.Complex, nameMap map[comptop.Index]string) {
	ps := c.PrincipleSimplices().Slice()
	fmt.Printf("Container count: %d\nNetwork count: %d\n", len(nameMap), len(ps))
	bn := c.BettiNumbers()
	fmt.Printf("Connected networks count: %d\nEuler characteristic: %d\n", bn[0], c.EulerChar())
	if len(bn) > 1 {
		fmt.Printf("Hole count:")
		for dim, bbn := range bn[1:] {
			fmt.Printf("\n\t- %dD: %d", dim+1, bbn)
		}
	}
	fmt.Println("")

	basis := c.ChainGroup(1).HomologyGroup().MinimalBasis()
	if basis != nil {
		fmt.Println("Minimal paths around 1-dimensional holes:")
		for _, bbasis := range basis {
			visited := map[comptop.Index]struct{}{}
			vs := []string{}
			for _, smplx := range bbasis.Simplices() {
				if _, printed := visited[smplx.Base()[0]]; printed {
					vs = append(vs, nameMap[smplx.Base()[1]])
				} else {
					vs = append(vs, nameMap[smplx.Base()[0]])
					visited[smplx.Base()[0]] = struct{}{}
				}
			}
			vs = append(vs, vs[0])
			fmt.Printf("\t- %s\n", strings.Join(vs, "->"))
		}
	}
}
