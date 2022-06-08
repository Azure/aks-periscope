package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest"
)

// KubeCommandRunner replicates some of the functionality provided by the kubectl binary.
// This uses the `Unstructured` package (https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured)
// to work with API resources.
// That decision means we sacrifice strong typing to get more straightforward serialization, fewer package
// dependencies, and better ability to handle resource version changes over time.
type KubeCommandRunner struct {
	kubeconfig *rest.Config
}

func NewKubeCommandRunner(config *rest.Config) *KubeCommandRunner {
	return &KubeCommandRunner{
		kubeconfig: config,
	}
}

// GetTableOutput replicates 'kubectl get [kind] -o [table|wide]'.
func (runner *KubeCommandRunner) GetTableOutput(gvr *schema.GroupVersionResource, namespace string, listOptions *metav1.ListOptions, printOptions *printers.PrintOptions) (string, error) {
	table, err := runner.GetUnstructuredTable(gvr, namespace, listOptions)
	if err != nil {
		return "", fmt.Errorf("error requesting table for %s in %s: %w", gvr.String(), namespace, err)
	}

	output, err := runner.PrintAsTable(table, printOptions)
	if err != nil {
		return "", fmt.Errorf("error printing %s in %s as table: %w", gvr.String(), namespace, err)
	}

	return output, nil
}

// GetJsonListOutput replicates 'kubectl get [kind] -o json'.
func (runner *KubeCommandRunner) GetJsonListOutput(gvr *schema.GroupVersionResource, namespace string, listOptions *metav1.ListOptions) (string, error) {
	list, err := runner.GetUnstructuredList(gvr, namespace, listOptions)
	if err != nil {
		return "", fmt.Errorf("error requesting all %s in %s: %w", gvr.String(), namespace, err)
	}

	output, err := runner.PrintAsJson(list)
	if err != nil {
		return "", fmt.Errorf("error printing %s in %s as JSON: %w", gvr.String(), namespace, err)
	}

	return output, nil
}

// GetYamlListOutput replicates 'kubectl get [kind] -o yaml'.
func (runner *KubeCommandRunner) GetYamlListOutput(gvr *schema.GroupVersionResource, namespace string, listOptions *metav1.ListOptions) (string, error) {
	list, err := runner.GetUnstructuredList(gvr, namespace, listOptions)
	if err != nil {
		return "", fmt.Errorf("error requesting all %s in %s: %w", gvr.String(), namespace, err)
	}

	output, err := runner.PrintAsYaml(list)
	if err != nil {
		return "", fmt.Errorf("error printing %s in %s as YAML: %w", gvr.String(), namespace, err)
	}

	return output, nil
}

// GetJsonObjectOutput replicates 'kubectl get [kind] [name] -o json'.
func (runner *KubeCommandRunner) GetJsonObjectOutput(gvr *schema.GroupVersionResource, namespace, name string) (string, error) {
	obj, err := runner.GetUnstructuredItem(gvr, namespace, name)
	if err != nil {
		return "", fmt.Errorf("error requesting %s %s in %s: %w", gvr.String(), name, namespace, err)
	}

	output, err := runner.PrintAsJson(obj)
	if err != nil {
		return "", fmt.Errorf("error printing %s %s in %s as JSON: %w", gvr.String(), name, namespace, err)
	}

	return output, nil
}

// GetYamlObjectOutput replicates 'kubectl get [kind] [name] -o yaml'.
func (runner *KubeCommandRunner) GetYamlObjectOutput(gvr *schema.GroupVersionResource, namespace, name string) (string, error) {
	obj, err := runner.GetUnstructuredItem(gvr, namespace, name)
	if err != nil {
		return "", fmt.Errorf("error requesting %s %s in %s: %w", gvr.String(), name, namespace, err)
	}

	output, err := runner.PrintAsYaml(obj)
	if err != nil {
		return "", fmt.Errorf("error printing %s %s in %s as YAML: %w", gvr.String(), name, namespace, err)
	}

	return output, nil
}

// GetUnstructuredList gets the API response to a List request in Unstructured form.
func (runner *KubeCommandRunner) GetUnstructuredList(gvr *schema.GroupVersionResource, namespace string, options *metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	request, err := runner.getUnstructuredRequest(gvr, false)
	if err != nil {
		return nil, fmt.Errorf("error getting request for JSON: %w", err)
	}

	request = request.NamespaceIfScoped(namespace, namespace != "").VersionedParams(options, metav1.ParameterCodec)

	obj, err := request.Do(context.Background()).Get()
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	return obj.(*unstructured.UnstructuredList), nil
}

// GetUnstructuredTable gets the API response to a List request for a server-generated table, in Unstructured form.
func (runner *KubeCommandRunner) GetUnstructuredTable(gvr *schema.GroupVersionResource, namespace string, options *metav1.ListOptions) (*unstructured.Unstructured, error) {
	request, err := runner.getUnstructuredRequest(gvr, true)
	if err != nil {
		return nil, fmt.Errorf("error getting request for table: %w", err)
	}

	request = request.NamespaceIfScoped(namespace, namespace != "").VersionedParams(options, metav1.ParameterCodec)

	obj, err := request.Do(context.Background()).Get()
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	return obj.(*unstructured.Unstructured), nil
}

// GetUnstructuredItem gets the API response to a Get request in Unstructured form.
func (runner *KubeCommandRunner) GetUnstructuredItem(gvr *schema.GroupVersionResource, namespace string, name string) (*unstructured.Unstructured, error) {
	request, err := runner.getUnstructuredRequest(gvr, false)
	if err != nil {
		return nil, fmt.Errorf("error getting request for JSON: %w", err)
	}

	request = request.NamespaceIfScoped(namespace, namespace != "").VersionedParams(&metav1.GetOptions{}, metav1.ParameterCodec).Name(name)

	obj, err := request.Do(context.Background()).Get()
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	return obj.(*unstructured.Unstructured), nil
}

// PrintAsJson takes the Unstructured representation or one or more resources and serializes them as formatted JSON.
// If the input is a List type, it replaces the Kind/APIVersion with a generic 'List' Kind (as kubectl does).
func (runner *KubeCommandRunner) PrintAsJson(obj runtime.Unstructured) (string, error) {
	return runner.serialize(obj, &printers.JSONPrinter{})
}

// PrintAsYaml takes the Unstructured representation or one or more resources and serializes them as formatted YAML.
// If the input is a List type, it replaces the Kind/APIVersion with a generic 'List' Kind (as kubectl does).
func (runner *KubeCommandRunner) PrintAsYaml(obj runtime.Unstructured) (string, error) {
	return runner.serialize(obj, &printers.YAMLPrinter{})
}

func (runner *KubeCommandRunner) serialize(obj runtime.Unstructured, printer printers.ResourcePrinter) (string, error) {
	list, isList := obj.(*unstructured.UnstructuredList)
	if isList {
		list.SetKind("List")
		list.SetAPIVersion("v1")
		list.SetResourceVersion("")
		list.SetSelfLink("")
	}

	omitManagedFieldsPrinter := printers.OmitManagedFieldsPrinter{Delegate: printer}

	var buf bytes.Buffer
	if err := omitManagedFieldsPrinter.PrintObj(obj, &buf); err != nil {
		return "", fmt.Errorf("error serializing resource: %w", err)
	}

	return buf.String(), nil
}

// PrintAsTable takes the Unstructured representation of a table (as returned from the API), and outputs it as a
// human-readable table.
func (runner *KubeCommandRunner) PrintAsTable(obj runtime.Unstructured, printOptions *printers.PrintOptions) (string, error) {
	table := &metav1.Table{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), table); err != nil {
		return "", fmt.Errorf("error converting unstructured data to table: %w", err)
	}

	printer := printers.NewTablePrinter(*printOptions)

	var buf bytes.Buffer
	if err := printer.PrintObj(table, &buf); err != nil {
		return "", fmt.Errorf("error printing resource as table: %w", err)
	}

	return buf.String(), nil
}

func (runner *KubeCommandRunner) getUnstructuredRequest(gvr *schema.GroupVersionResource, asTable bool) (*rest.Request, error) {
	// Normally, to get Unstructured data you'd use the 'dynamic' client (https://pkg.go.dev/k8s.io/client-go/dynamic).
	// However, to request table-formatted data from the API, you need to set the Accept header on the request.
	// AFAICT, the dynamic client doesn't support this, so here we build the request directly.
	config := rest.CopyConfig(runner.kubeconfig)
	config.ContentConfig = resource.UnstructuredPlusDefaultContentConfig()
	gv := gvr.GroupVersion()
	config.GroupVersion = &gv
	if len(gv.Group) == 0 {
		config.APIPath = "/api"
	} else {
		config.APIPath = "/apis"
	}

	var restClient resource.RESTClient
	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}

	if asTable {
		restClient = resource.NewClientWithOptions(restClient, func(req *rest.Request) {
			req.SetHeader("Accept", "application/json;as=Table;v=v1;g=meta.k8s.io")
		})
	}

	return restClient.Get().Resource(gvr.Resource), nil
}

// For compatibility between K8s versions, we support retrieval of CRDs with newer and older APIVersions.
// (https://kubernetes.io/docs/reference/using-api/deprecation-guide/#customresourcedefinition-v122)
var crdGvrs = []schema.GroupVersionResource{
	schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"},
	schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1beta1", Resource: "customresourcedefinitions"},
}

// GetCRDUnstructuredList reads all the CRDs in the cluster and returns the result as an UnstructuredList.
func (runner *KubeCommandRunner) GetCRDUnstructuredList() (*unstructured.UnstructuredList, error) {
	for _, gvr := range crdGvrs {
		crds, err := runner.GetUnstructuredList(&gvr, "", &metav1.ListOptions{})
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		// Found
		return crds, nil
	}

	return nil, errors.New("no CRD resource type found")
}

// GetGVRForCRD gets the GroupVersionResource for the specified CRD (where Version is the 'storage'
// version for the resources).
func (runner *KubeCommandRunner) GetGVRForCRD(crdName string) (*schema.GroupVersionResource, error) {
	for _, gvr := range crdGvrs {
		crd, err := runner.GetUnstructuredItem(&gvr, "", crdName)
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		return runner.GetGVRFromCRD(crd)
	}
	return nil, fmt.Errorf("crd %s not found", crdName)
}

// GetGVRFromCRD takes a CRD in Unstructured form and returns the GroupVersionResource for its resources.
func (runner *KubeCommandRunner) GetGVRFromCRD(crd *unstructured.Unstructured) (*schema.GroupVersionResource, error) {
	// The name of a CRD is of the form 'resource.group', so that gives us 2/3 of the GVR.
	name := crd.GetName()
	groupResource := schema.ParseGroupResource(name)

	// For the version, use the 'storage' version. See:
	// https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#writing-reading-and-updating-versioned-customresourcedefinition-objects
	versionList, found, err := unstructured.NestedSlice(crd.Object, "spec", "versions")
	if !found || err != nil {
		return nil, errors.New("spec.versions not found")
	}
	version := ""
	for _, versionItem := range versionList {
		versionObj := versionItem.(map[string]interface{})
		isStorageVersion, found, err := unstructured.NestedBool(versionObj, "storage")
		if !found || err != nil {
			return nil, errors.New("spec.versions[].storage not found")
		}

		if isStorageVersion {
			version, found, err = unstructured.NestedString(versionObj, "name")
			if !found || err != nil {
				return nil, errors.New("spec.versions[].name not found")
			}
			break
		}
	}
	if version == "" {
		return nil, errors.New("no storage version found")
	}

	gvr := groupResource.WithVersion(version)
	return &gvr, nil
}
