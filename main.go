package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/yaml"
)

const (
	defaultNS = "default"
)

type HelmChartTemplate struct {
	Name string `json:"name"`
	// Info struct {
	// 	FirstDeployed time.Time `json:"first_deployed"`
	// 	LastDeployed  time.Time `json:"last_deployed"`
	// 	Deleted       string    `json:"deleted"`
	// 	Description   string    `json:"description"`
	// 	Status        string    `json:"status"`
	// 	Notes         string    `json:"notes"`
	// } `json:"info"`
	Chart struct {
		Metadata  json.RawMessage `json:"metadata"`
		Templates []struct {
			Name string `json:"name"`
			Data string `json:"data"`
		} `json:"templates"`
		Values json.RawMessage `json:"values"`
		// Schema any             `json:"schema"`
		Files []struct {
			Name string `json:"name"`
			Data string `json:"data"`
		} `json:"files"`
		Dependencies []struct {
			Name       string `json:"name"`
			Version    string `json:"version"`
			Repository string `json:"repository"`
		} `json:"dependencies"`
	} `json:"chart"`
	Manifest string `json:"manifest"`
	// Version   int    `json:"version"`
	Namespace string `json:"namespace"`
}

type cmdOpts struct {
	kubeConfig     string
	clusterContext string
	outDir         string
	latest         bool
}

// https://devops.stackexchange.com/questions/4344/original-helm-chart-gone-how-can-i-find-get-it-from-the-cluster/17642#17642?newreg=b1f82da562c445b086a171eb8397f33b
func main() {
	var opts cmdOpts

	flag.StringVar(&opts.outDir, "o", ".", "Destination folder (output)")
	flag.StringVar(&opts.kubeConfig, "f", filepath.Join(homedir.HomeDir(), ".kube", "config"), "Kubernetes config file")
	flag.StringVar(&opts.clusterContext, "context", "", "Cluster context name")
	flag.BoolVar(&opts.latest, "latest", false, "Latest helm chart only")
	flag.Parse()

	os.MkdirAll(opts.outDir, os.ModePerm)

	if err := rootCommand(opts); err != nil {
		log.Fatal(err)
	}
}

func rootCommand(opts cmdOpts) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		config *rest.Config
		err    error
	)

	if opts.clusterContext != "" {
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: opts.kubeConfig},
			&clientcmd.ConfigOverrides{
				CurrentContext: opts.clusterContext,
			}).ClientConfig()
		if err != nil {
			return err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", opts.kubeConfig)
		if err != nil {
			return err
		}
	}

	// Creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	secrets, err := clientset.CoreV1().Secrets(defaultNS).List(ctx, v1.ListOptions{})
	if err != nil {
		return err
	}

	for len(secrets.Items) > 0 {
		secret := secrets.Items[0]

		if !strings.HasPrefix(string(secret.Type), "helm.sh/release.") {
			secrets.Items = secrets.Items[1:]
			continue
		}

		b := secret.Data["release"]
		if len(b) == 0 {
			secrets.Items = secrets.Items[1:]
			continue
		}
		b, err = base64Decode(b)
		if err != nil {
			return err
		}

		r, err := gzip.NewReader(bytes.NewBuffer(b))
		if err != nil {
			return err
		}
		defer r.Close()

		var chart HelmChartTemplate
		if err := json.NewDecoder(r).Decode(&chart); err != nil {
			return err
		}

		rootDir := filepath.Join(opts.outDir, secret.Name)
		os.MkdirAll(rootDir, os.ModePerm)

		if opts.latest {
			lastIdx := strings.LastIndex(secret.Name, ".")
			version, _ := strconv.Atoi(secret.Name[lastIdx+2:])
			prevVersionDir := filepath.Join(opts.outDir, secret.Name[:lastIdx+2]+strconv.Itoa(version-1))
			if _, err := os.Stat(prevVersionDir); !os.IsNotExist(err) {
				os.RemoveAll(prevVersionDir)
			}
		}

		// Creating `Chart.yaml`
		{
			mb, err := yaml.JSONToYAML(chart.Chart.Metadata)
			if err != nil {
				return err
			}

			f, err := os.OpenFile(filepath.Join(rootDir, "Chart.yaml"), os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return err
			}
			defer f.Close()

			f.Write(mb)
			f.Close()
		}

		// Creating `values.yaml`
		{
			yb, err := yaml.JSONToYAML(chart.Chart.Values)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				return err
			}

			f, err := os.OpenFile(filepath.Join(rootDir, "values.yaml"), os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return err
			}
			defer f.Close()

			f.Write(yb)
			f.Close()
		}

		for _, tmpl := range chart.Chart.Templates {
			os.MkdirAll(filepath.Join(rootDir, filepath.Dir(tmpl.Name)), os.ModePerm)
			b, _ := base64.StdEncoding.DecodeString(tmpl.Data)
			fileName := filepath.Join(rootDir, tmpl.Name)
			f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return err
			}
			defer f.Close()

			f.Write(b)
			f.Close()
		}

		for _, tmpl := range chart.Chart.Files {
			os.MkdirAll(filepath.Join(rootDir, filepath.Dir(tmpl.Name)), os.ModePerm)
			b, _ := base64.StdEncoding.DecodeString(tmpl.Data)
			fileName := filepath.Join(rootDir, tmpl.Name)
			f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return err
			}
			defer f.Close()

			f.Write(b)
			f.Close()
		}

		secrets.Items = secrets.Items[1:]
	}
	return nil
}

func base64Decode(v []byte) ([]byte, error) {
	b64 := make([]byte, base64.StdEncoding.DecodedLen(len(v)))
	n, err := base64.StdEncoding.Decode(b64, v)
	if err != nil {
		return nil, err
	}
	return b64[:n], nil
}
