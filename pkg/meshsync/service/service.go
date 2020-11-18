package service

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "MeshSync",
	Short: "Cluster and service mesh specific resource discovery",
	Run: func(cmd *cobra.Command, args []string) {
		kubeconfig, err := cmd.Flags().GetString("kubeconfig")
		if err != nil {
			fmt.Printf("Error : %s", err)
			return
		}
		var config *rest.Config
		if kubeconfig != "" {
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				log.Printf("Couldnot load config: %s", err)
				return
			}
		} else {
			config, err = rest.InClusterConfig()
			if err != nil {
				log.Printf("Couldnot load config: %s", err)
				return
			}
		}

		err = StartDiscovery(config)
		if err != nil {
			log.Printf("Error while discovery: %s", err)
			return
		}

	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringP("kubeconfig", "k", "", "path to kube config file")
}
