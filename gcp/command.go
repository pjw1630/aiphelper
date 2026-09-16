package gcp

import (
	"github.com/jessevdk/go-flags"
)

var options *Options

func AddCommand(p *flags.Parser) {
	options = &Options{}
	_, err := p.AddCommand("gcp", "Initialize GCP", "Initialize GCP config and Steampipe connections", options)
	if err != nil {
		panic(err)
	}
}
