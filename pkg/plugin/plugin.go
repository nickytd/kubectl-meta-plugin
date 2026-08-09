// Copyright 2026 nickytd
// SPDX-License-Identifier: Apache-2.0

// Package plugin implements the kubectl-meta plugin logic.
package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/yaml"
)

// Run fetches a Kubernetes resource and prints its .metadata block.
// outputFmt must be "yaml" or "json". managedFields are stripped unless withManagedFields is true.
func Run(f util.Factory, streams genericiooptions.IOStreams, outputFmt string, withManagedFields bool, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("resource argument required (TYPE/NAME or NAME)")
	}

	resourceType, resourceName, err := parseArg(args[0])
	if err != nil {
		return err
	}

	ns, _, err := f.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return fmt.Errorf("resolving namespace: %w", err)
	}

	r := f.NewBuilder().
		Unstructured().
		NamespaceParam(ns).DefaultNamespace().
		ResourceTypeOrNameArgs(false, resourceType, resourceName).
		SingleResourceType().
		Do()

	if err := r.Err(); err != nil {
		return fmt.Errorf("fetching %s/%s: %w", resourceType, resourceName, err)
	}

	var result error
	if visitErr := r.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			result = fmt.Errorf("visiting %s/%s: %w", resourceType, resourceName, err)
			return result
		}

		u, ok := info.Object.(*unstructured.Unstructured)
		if !ok {
			result = fmt.Errorf("unexpected object type %T for %s/%s", info.Object, resourceType, resourceName)
			return result
		}

		metadata, found, err := unstructured.NestedMap(u.Object, "metadata")
		if err != nil {
			result = fmt.Errorf("reading metadata of %s/%s: %w", resourceType, resourceName, err)
			return result
		}
		if !found {
			result = fmt.Errorf("no metadata found in %s/%s", resourceType, resourceName)
			return result
		}

		if !withManagedFields {
			delete(metadata, "managedFields")
		}

		result = printMetadata(streams, outputFmt, metadata)
		return result
	}); visitErr != nil && result == nil {
		result = visitErr
	}

	return result
}

// parseArg splits "type/name" or treats a bare name as a pod.
func parseArg(arg string) (resourceType, resourceName string, err error) {
	if arg == "" {
		return "", "", fmt.Errorf("resource argument must not be empty")
	}
	if before, after, ok := strings.Cut(arg, "/"); ok {
		resourceType = before
		resourceName = after
		if resourceType == "" || resourceName == "" {
			return "", "", fmt.Errorf("invalid resource argument %q: expected TYPE/NAME", arg)
		}
		return resourceType, resourceName, nil
	}
	return "pod", arg, nil
}

func printMetadata(streams genericiooptions.IOStreams, outputFmt string, metadata map[string]any) error {
	var out []byte
	var err error

	switch outputFmt {
	case "json":
		out, err = json.MarshalIndent(metadata, "", "  ")
	default: // yaml
		out, err = yaml.Marshal(metadata)
	}
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	_, err = fmt.Fprintln(streams.Out, strings.TrimRight(string(out), "\n"))
	return err
}
