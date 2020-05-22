package main

import (
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"os"
)

const (
	defaultWatchNamespace = v1.NamespaceAll
)

type Options struct {
	Debug              bool
	ZoneName           string
	ClusterName        string
	WatchNamespace     string
	CloudflareApiToken string
}

func (options *Options) BindFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&options.ZoneName, "zone-name", "z", "", "* cloudflare zone name (example.com)")
	fs.StringVarP(&options.ClusterName, "cluster-name", "c", "", "* A unique string in case use the same zone")
	fs.StringVar(&options.WatchNamespace, "watch-namespace", defaultWatchNamespace,
		`Namespace the controller watches for updates to Kubernetes objects.
This includes Ingresses, Services and all configuration resources. All
namespaces are watched if this parameter is left empty.`)

	fs.BoolVar(&options.Debug, "debug", false, "debugging")
}

func (options *Options) BindEnv() {
	if s, ok := os.LookupEnv("CF_TOKEN"); ok {
		options.CloudflareApiToken = s
	}
}

func getOptions() *Options {
	options := new(Options)

	fs := pflag.NewFlagSet("", pflag.ExitOnError)
	fs.SortFlags = false

	options.BindFlags(fs)
	_ = fs.Parse(os.Args)

	options.BindEnv()

	return options
}
