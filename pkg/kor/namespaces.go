package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

type GenericResource struct {
	NamespacedName types.NamespacedName
	GVR            schema.GroupVersionResource
}

func getGVR(name string, splitGV []string) (*schema.GroupVersionResource, error) {
	switch NumberOfGVPartsFound := len(splitGV); NumberOfGVPartsFound {
	case 1:
		return &schema.GroupVersionResource{
			Version:  splitGV[0],
			Resource: name,
		}, nil
	case 2:
		return &schema.GroupVersionResource{
			Group:    splitGV[0],
			Version:  splitGV[1],
			Resource: name,
		}, nil
	default:
		return nil, fmt.Errorf("gv is wrong length slice: %d", NumberOfGVPartsFound)
	}
}

func ignoreResourceType(resource string, ignoreResourceTypes []string) bool {
	for _, ignoreType := range ignoreResourceTypes {
		if resource == ignoreType {
			return true
		}
	}
	return false
}

func ignorePredefinedResource(gr GenericResource) bool {
	// Specific list of resources to ignore - resources created in all namespaced by default
	if gr.GVR.Resource == "configmaps" && gr.GVR.Version == "v1" && gr.NamespacedName.Name == "kube-root-ca.crt" {
		return true
	}
	if gr.GVR.Resource == "serviceaccounts" && gr.GVR.Version == "v1" && gr.NamespacedName.Name == "default" {
		return true
	}
	if gr.GVR.Resource == "events" {
		return true
	}
	return false
}

func isNamespaceNotEmpty(
	gvr *schema.GroupVersionResource,
	unstructuredList *unstructured.UnstructuredList,
	filterOpts *filters.Options,
) bool {
	for _, unstructuredObj := range unstructuredList.Items {
		gr := GenericResource{
			GVR: *gvr,
			NamespacedName: types.NamespacedName{
				Namespace: unstructuredObj.GetNamespace(),
				Name:      unstructuredObj.GetName(),
			},
		}
		// Ignore default cluster resources
		if ignorePredefinedResource(gr) {
			continue
		}
		// User specified resource type ignore list
		if ignoreResourceType(gr.GVR.Resource, filterOpts.IgnoreResourceTypes) {
			continue
		}
		return true
	}
	return false
}

func isErrorOrNamespaceContainsResources(
	ctx context.Context,
	clientset kubernetes.Interface,
	dynamicClient dynamic.Interface,
	namespace string,
	filterOpts *filters.Options,
) (bool, error) {
	apiResourceLists, err := clientset.Discovery().ServerPreferredNamespacedResources()
	if err != nil {
		return true, err
	}

	// Iterate over all API resources and list instances of each in the specified namespace
	for _, apiResourceList := range apiResourceLists {
		for _, apiResource := range apiResourceList.APIResources {
			gv := strings.Split(apiResourceList.GroupVersion, "/")
			gvr, err := getGVR(apiResource.Name, gv)
			if err != nil {
				return true, err
			}

			unstructuredList, err := dynamicClient.Resource(*gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				continue
			}

			if isNamespaceNotEmpty(gvr, unstructuredList, filterOpts) {
				return true, nil
			}
		}
	}
	return false, nil
}

func processNamespaces(
	ctx context.Context,
	clientset kubernetes.Interface,
	dynamicClient dynamic.Interface,
	filterOpts *filters.Options,
) ([]ResourceInfo, error) {
	var unusedNamespaces []ResourceInfo

	namespaces, err := clientset.CoreV1().Namespaces().List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list namespaces")
	}

	for _, namespace := range namespaces.Items {
		if pass := filters.ApplyFilters(
			&namespace, filterOpts,
			filters.SystemNamespaceFilter,
			filters.ExcludeNamespacesFilter,
			filters.IncludeNamespacesFilter,
			filters.KorLabelFilter,
			filters.LabelFilter,
			filters.AgeFilter,
		); pass {
			continue
		}

		// skipping default resources here
		resourceFound, err := isErrorOrNamespaceContainsResources(
			ctx,
			clientset,
			dynamicClient,
			namespace.Name,
			filterOpts,
		)
		if err != nil {
			return unusedNamespaces, err
		}

		// construct list of unused namespaces here following a set of rules
		if !resourceFound {
			unusedNamespaces = append(unusedNamespaces, ResourceInfo{namespace.Name, "unused namespace"})
		}
	}
	return unusedNamespaces, nil
}

func GetUnusedNamespaces(
	ctx context.Context,
	filterOpts *filters.Options,
	clientset kubernetes.Interface,
	dynamicClient dynamic.Interface,
	outputFormat string,
	opts common.Opts,
) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)

	if len(filterOpts.IncludeNamespaces) > 0 && len(filterOpts.ExcludeNamespaces) > 0 {
		fmt.Fprintf(os.Stderr, "Exclude namespaces can't be used together with include namespaces. Ignoring --exclude-namespace(-e) flag\n")
		filterOpts.ExcludeNamespaces = nil
	}

	diff, err := processNamespaces(ctx, clientset, dynamicClient, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process namespaces: %v\n", err)
	}

	if len(diff) > 0 {
		// We consider cluster scope resources in "" (empty string) namespace, as it is common in k8s
		if resources[""] == nil {
			resources[""] = make(map[string][]ResourceInfo)
		}
		resources[""]["Namespaces"] = diff
	}

	if opts.DeleteFlag {
		if diff, err = DeleteResource(
			diff,
			clientset,
			"",
			"Namespace",
			opts.NoInteractive,
		); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete namespace %s : %v\n", diff, err)
		}
	}

	var outputBuffer bytes.Buffer
	var jsonResponse []byte
	switch outputFormat {
	case "table":
		outputBuffer = FormatOutput(resources, opts)
	case "json", "yaml":
		var err error
		if jsonResponse, err = json.MarshalIndent(resources, "", "  "); err != nil {
			return "", err
		}
	}

	unusedNamespaces, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedNamespaces, nil
}