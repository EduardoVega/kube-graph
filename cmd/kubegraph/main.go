package main

import (
	"flag"
	"os"

	"github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"fmt"

	"github.com/EduardoVega/kubegraph/graph"
	"github.com/EduardoVega/kubegraph/internal"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // support for cloud providers auth
)

// Options defines the graph command options
type Options struct {
	ConfigFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
	Client       dynamic.Interface
	Namespace    string
	Name         string
	Kind         string
	DotGraph     bool
	PrintVersion bool
}

func init() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	_ = pflag.Set("logtostderr", "true")

	// Hide flags from --help
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		if f.Name != "v" {
			pflag.Lookup(f.Name).Hidden = true
		}
	})
}

func main() {
	defer klog.Flush()
	c := NewCmd(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}

// NewOptions returns an Options struct
func NewOptions(iostreams genericclioptions.IOStreams) *Options {
	return &Options{
		ConfigFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   iostreams,
	}
}

// NewCmd returns a new graph command
func NewCmd(iostreams genericclioptions.IOStreams) *cobra.Command {
	o := NewOptions(iostreams)

	c := &cobra.Command{
		Use:   "[KIND] [NAME] [flags]",
		Short: "Print a tree or dot graph to visualize the relationship between kubernetes objects",
		Example: `
# Print a tree graph that shows all kubernetes objects that are related to the service service-foo
kubegraph service service-foo

kubectl graph service service-foo

# Print a DOT graph that shows all kubernetes objects that are related to the ingress ingress-bar
kubegraph ingress ingress-bar --dot 

kubectl graph ingress ingress-bar --dot
`,
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	c.Flags().BoolVar(&o.DotGraph, "dot", o.DotGraph, "If true, a DOT graph will be printed to stdout")
	c.Flags().BoolVar(&o.PrintVersion, "version", o.PrintVersion, "Print kubegraph the version")
	o.ConfigFlags.AddFlags(c.Flags())

	return c
}

// Complete adds values to Option attributes
func (o *Options) Complete(cmd *cobra.Command, args []string) error {
	klog.V(1).Infoln("add information to Options struct")

	// Get Type and Name
	if len(args) >= 2 {
		o.Kind = args[0]
		o.Name = args[1]
	}

	// Get namespace
	namespace, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return err
	}

	if namespace == "" {
		clientConfig := o.ConfigFlags.ToRawKubeConfigLoader()

		namespace, _, err = clientConfig.Namespace()
		if err != nil {
			return err
		}
	}

	o.Namespace = namespace

	// Get Client
	restConfig, err := o.ConfigFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	o.Client = dynClient

	return nil
}

// Validate ensures all expected data is available
func (o *Options) Validate(args []string) error {
	klog.V(1).Infoln("validate arguments")

	if len(args) != 2 && !o.PrintVersion {
		return fmt.Errorf("requires valid 'kind' and 'name' arguments")
	}

	return nil
}

// Run creates a new resource and starts the data gathering to build the graph
func (o *Options) Run() error {
	klog.V(1).Infoln("execute the build function of the Builder")

	if o.PrintVersion {
		fmt.Printf("Version:\t%s\nBranch:\t\t%s\nCommit:\t\t%s\nGo Version:\t%s\nOS/Arch:\t%s\nDate:\t\t%s\n", internal.Version, internal.Branch, internal.Commit, internal.GoVersion, internal.OSArch, internal.Date)
	} else {
		b := graph.NewBuilder(o.Client, o.Out, o.DotGraph, o.Namespace, o.Kind, o.Name)

		err := b.Build()
		if err != nil {
			return err
		}

	}

	return nil
}
