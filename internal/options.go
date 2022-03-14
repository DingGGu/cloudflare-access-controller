package internal

import (
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"os"
	"time"
)

const (
	defaultWatchNamespace   = v1.NamespaceAll
	defaultSyncPeriodSecond = 60
)

type Options struct {
	Debug              bool
	ZoneName           string
	ClusterName        string
	WatchNamespace     string
	CloudflareApiToken string
	SyncPeriodSecond   time.Duration
}

func (options *Options) BindFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&options.ZoneName, "zone-name", "z", "", "* cloudflare zone name (example.com)")
	fs.StringVarP(&options.ClusterName, "cluster-name", "c", "", "* A unique string in case use the same zone")
	fs.StringVar(&options.WatchNamespace, "watch-namespace", defaultWatchNamespace,
		`Namespace the controller watches for updates to Kubernetes objects.
This includes Ingresses, Services and all configuration resources. All
namespaces are watched if this parameter is left empty.`)
	fs.DurationVar(&options.SyncPeriodSecond, "sync-period", defaultSyncPeriodSecond, `SyncPeriod determines the minimum frequency at which watched resources are reconciled. You have to enter the seconds, recommend 60 seconds or more`)
	options.SyncPeriodSecond = options.SyncPeriodSecond * time.Second

	fs.BoolVar(&options.Debug, "debug", false, "debugging")
}

func (options *Options) BindEnv() {
	if s, ok := os.LookupEnv("CF_API_TOKEN"); ok {
		options.CloudflareApiToken = s
	}
}

func GetOptions() *Options {
	options := new(Options)

	fs := pflag.NewFlagSet("", pflag.ExitOnError)
	fs.SortFlags = false

	options.BindFlags(fs)
	_ = fs.Parse(os.Args)

	options.BindEnv()

	return options
}
