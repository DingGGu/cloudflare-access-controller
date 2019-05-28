// Copyright © 2019 DingGGu <dingggu@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"github.com/DingGGu/cloudflare-access-controller/internal/controller"
	"github.com/DingGGu/cloudflare-access-controller/internal/provider/cf"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func Execute() {
	var clusterName string
	var kubeConfig string
	var zoneNames []string

	var rootCmd = &cobra.Command{
		Use:   "cloudflare-access-controller",
		Short: "Start cloudflare access controller",
		Long: `Start cloudflare access controller
Annotate Kubernetes ingress
`,
		Run: func(cmd *cobra.Command, args []string) {
			logrus.SetFormatter(&logrus.JSONFormatter{})
			logrus.SetLevel(logrus.DebugLevel)

			if home := homedir.HomeDir(); home != "" && kubeConfig == "" {
				kubeConfig = filepath.Join(home, ".kube", "config")
			}

			config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)

			if err != nil {
				logrus.Error(err)
				os.Exit(1)
			}
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				logrus.Error(err)
				os.Exit(1)
			}

			ctrl := controller.NewController(
				clientset,
				cf.GetCloudFlareClient(os.Getenv("CF_API_KEY"), os.Getenv("CF_API_EMAIL")),
				clusterName,
				"",
				zoneNames,
			)

			ctrl.Run()
		},
	}

	viper.AutomaticEnv()
	rootCmd.Flags().StringVarP(&clusterName, "clusterName", "c", "", "Cluster Name to identify multiple clusters in cloudflare (required)")
	rootCmd.Flags().StringSliceVarP(&zoneNames, "zoneName", "z", []string{}, "Slice of cloudflare ZoneName (required)")
	rootCmd.Flags().StringVarP(&kubeConfig, "kubeconfig", "k", "", "kubernetes config path")
	rootCmd.MarkFlagRequired("clusterName")
	rootCmd.MarkFlagRequired("zoneName")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
