// Copyright 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the kubectl-meta plugin.
package main

import (
	goflag "flag"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	kcomplete "k8s.io/kubectl/pkg/util/completion"

	"github.com/nickytd/kubectl-meta-plugin/pkg/plugin"
)

var version = "dev"

func main() {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {

	streams := genericiooptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	// meta plugin flags
	var outputFmt string
	var withManagedFields bool

	// single Factory instance shared by RunE and completion
	configFlags := genericclioptions.NewConfigFlags(true)
	f := util.NewFactory(configFlags)

	cmd := &cobra.Command{
		Use:          "kubectl-meta TYPE/NAME [flags]",
		Short:        "Show metadata of any Kubernetes resource",
		Long:         `Displays the .metadata of any Kubernetes resource in YAML (default) or JSON format.`,
		Example:      "kubectl meta pod/my-pod\n  kubectl meta deploy/my-deploy -n kube-system -o json\n  kubectl meta my-pod  # defaults to pod type",
		SilenceUsage: true,
		Version:      version,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if outputFmt != "yaml" && outputFmt != "json" {
				return fmt.Errorf("invalid output format %q: must be yaml or json", outputFmt)
			}
			return plugin.Run(f, streams, outputFmt, withManagedFields, args)
		},
	}

	flags := cmd.Flags()
	configFlags.AddFlags(flags)

	// let's skip irrelevant k8s cli flags
	for _, name := range []string{
		"as", "as-group", "as-uid",
		"cache-dir",
		"certificate-authority", "client-certificate", "client-key",
		"insecure-skip-tls-verify",
		"token", "username", "password",
	} {
		if fl := flags.Lookup(name); fl != nil {
			fl.Hidden = true
		}
	}

	klog.InitFlags(nil)
	cmd.PersistentFlags().AddGoFlagSet(goflag.CommandLine)

	// let's skip irrelevant klog flags
	for _, name := range []string{
		"add_dir_header", "alsologtostderr",
		"log_backtrace_at", "log_dir", "log_file", "log_file_max_size",
		"logtostderr", "one_output",
		"skip_headers", "skip_log_headers", "stderrthreshold",
	} {
		if fl := cmd.PersistentFlags().Lookup(name); fl != nil {
			fl.Hidden = true
		}
	}

	flags.StringVarP(&outputFmt, "output", "o", "yaml", "Output format: yaml or json")
	flags.BoolVar(&withManagedFields, "with-managed-fields", false, "Include managedFields in output")

	// plugin completion function to list the k8s resources when TAB is pressed
	cmd.ValidArgsFunction = typeSlashNameCompletionFunc(f)

	// needed for the actual completion
	cmd.AddCommand(newCompletionCmd())

	return cmd
}

// typeSlashNameCompletionFunc completes the single TYPE/NAME argument.
// First TAB: completes resource types with NoSpace so the shell appends "/" instead of " ".
// After "/": completes resource names inline as TYPE/NAME.
func typeSlashNameCompletionFunc(f util.Factory) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// we are done, no need to complete
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		before, after, ok := strings.Cut(toComplete, "/")
		if !ok {
			// there is no "/" yet — complete the resource type first
			comps := resourceTypeCompletions(f, toComplete)
			return comps, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		}

		// guard against bare "/" with no type prefix
		resourceType := before
		if resourceType == "" {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// completing the name portion of TYPE/NAME
		namePrefix := after
		names := kcomplete.CompGetResource(f, resourceType, namePrefix)
		comps := make([]string, len(names))
		for i, n := range names {
			comps[i] = resourceType + "/" + n
		}
		return comps, cobra.ShellCompDirectiveNoFileComp
	}
}

// resourceTypeCompletions returns resource type names (with trailing "/") that start with prefix.
// When the same short name exists in multiple API groups, all group-qualified variants are offered.
func resourceTypeCompletions(f util.Factory, prefix string) []string {
	dc, err := f.ToDiscoveryClient()
	if err != nil {
		return nil
	}

	lists, err := dc.ServerPreferredResources()
	// partial results are still ok
	if err != nil && lists == nil {
		return nil
	}

	// track how many groups expose each short name
	// ingresses   → networking.k8s.io
	// ingresses   → extensions
	counts := make(map[string]int)
	type entry struct {
		shortName string
		group     string
	}
	var entries []entry

	for _, list := range lists {
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			continue
		}
		for _, r := range list.APIResources {
			if strings.Contains(r.Name, "/") {
				continue // skip subresources
			}
			counts[r.Name]++
			entries = append(entries, entry{shortName: r.Name, group: gv.Group})
		}
	}

	seen := make(map[string]struct{})
	var comps []string
	for _, e := range entries {
		var candidate string
		if counts[e.shortName] > 1 && e.group != "" {
			// ambiguous short name: qualify with group
			candidate = e.shortName + "." + e.group
		} else {
			candidate = e.shortName
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		if strings.HasPrefix(candidate, prefix) {
			comps = append(comps, candidate+"/")
		}
	}
	return comps
}

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion script",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}
}
