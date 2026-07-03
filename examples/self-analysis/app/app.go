// Package app imports lib for self-analysis conformance fixtures.
package app

import "github.com/BrendenWalker/lineagis/examples/self-analysis/lib"

// Run invokes lib.Greet for conformance import edges.
func Run(name string) string {
	return lib.Greet(name)
}
