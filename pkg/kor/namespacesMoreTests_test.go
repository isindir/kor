package kor

import (
	"context"
	"fmt"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	discoveryfake "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/yonahd/kor/pkg/filters"
)

type fakeHappyDiscovery struct {
	discoveryfake.FakeDiscovery
}

func (c *fakeHappyDiscovery) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	return []*metav1.APIResourceList{
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "deployments",
					Namespaced: true,
					Kind:       "Deployment",
				},
			},
		},
	}, nil
}

type fakeUnhappyDiscovery struct {
	discoveryfake.FakeDiscovery
}

func (c *fakeUnhappyDiscovery) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	return nil, fmt.Errorf("fake error from discovery")
}

type fakeBrokenAPIResourceListDiscovery struct {
	discoveryfake.FakeDiscovery
}

func (c *fakeBrokenAPIResourceListDiscovery) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	return []*metav1.APIResourceList{
		{
			GroupVersion: "fake/broken/apps/v1", // this line causes error
			APIResources: []metav1.APIResource{
				{
					Name:       "deployments",
					Namespaced: true,
					Kind:       "Deployment",
				},
			},
		},
	}, nil
}

type fakeClientset struct {
	kubernetes.Interface
	discovery discovery.DiscoveryInterface
}

func (c *fakeClientset) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}

// Create a test deployment in the namespace
func defineDeployObject(ns, name string) *appsv1.Deployment {
	var replicas int32 = 42
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx",
						},
					},
				},
			},
		},
	}
}

func defineNamespaceObject(nsName string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}
}

func getNamespaceTestSchema(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	if err != nil {
		t.Errorf("Failed to add corev1 to scheme: %v", err)
	}
	err = appsv1.AddToScheme(scheme)
	if err != nil {
		t.Errorf("Failed to add appsv1 to scheme: %v", err)
	}
	return scheme

}

func createHappyDeployFakeClientInterfaces(ctx context.Context, t *testing.T, ns, name string) (kubernetes.Interface, *dynamicfake.FakeDynamicClient) {
	realClientset := fake.NewSimpleClientset()
	fakeDisc := &fakeHappyDiscovery{discoveryfake.FakeDiscovery{Fake: &realClientset.Fake}}
	clientset := &fakeClientset{Interface: realClientset, discovery: fakeDisc}

	scheme := getNamespaceTestSchema(t)
	namespace := defineNamespaceObject(ns)
	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	deployment := defineDeployObject(ns, name)
	_, err = clientset.AppsV1().Deployments("test-namespace").Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test deployment: %v", err)
	}

	listKinds := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}: "DeploymentList",
		{Group: "", Version: "v1", Resource: "namespaces"}:      "NamespaceList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, deployment, namespace)

	return clientset, dynamicClient
}

func createHappyEmptyFakeClientInterfaces(ctx context.Context, t *testing.T, ns, name string) (kubernetes.Interface, *dynamicfake.FakeDynamicClient) {
	realClientset := fake.NewSimpleClientset()
	fakeDisc := &fakeHappyDiscovery{discoveryfake.FakeDiscovery{Fake: &realClientset.Fake}}
	clientset := &fakeClientset{Interface: realClientset, discovery: fakeDisc}

	scheme := getNamespaceTestSchema(t)
	namespace := defineNamespaceObject(ns)
	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	listKinds := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}: "DeploymentList",
		{Group: "", Version: "v1", Resource: "namespaces"}:      "NamespaceList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, namespace)

	return clientset, dynamicClient
}

func createUnhappyDiscoveryFakeClientInterfaces(ctx context.Context, t *testing.T, ns, name string) (kubernetes.Interface, *dynamicfake.FakeDynamicClient) {
	realClientset := fake.NewSimpleClientset()
	fakeDisc := &fakeUnhappyDiscovery{discoveryfake.FakeDiscovery{Fake: &realClientset.Fake}}
	clientset := &fakeClientset{Interface: realClientset, discovery: fakeDisc}

	scheme := getNamespaceTestSchema(t)
	namespace := defineNamespaceObject(ns)
	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	listKinds := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}: "DeploymentList",
		{Group: "", Version: "v1", Resource: "namespaces"}:      "NamespaceList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, namespace)

	return clientset, dynamicClient
}

func createBrokenAPIResourceListDiscoveryFakeClientInterfaces(ctx context.Context, t *testing.T, ns, name string) (kubernetes.Interface, *dynamicfake.FakeDynamicClient) {
	realClientset := fake.NewSimpleClientset()
	fakeDisc := &fakeBrokenAPIResourceListDiscovery{discoveryfake.FakeDiscovery{Fake: &realClientset.Fake}}
	clientset := &fakeClientset{Interface: realClientset, discovery: fakeDisc}

	scheme := getNamespaceTestSchema(t)
	namespace := defineNamespaceObject(ns)
	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	listKinds := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}: "DeploymentList",
		{Group: "", Version: "v1", Resource: "namespaces"}:      "NamespaceList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, namespace)

	return clientset, dynamicClient
}

func createDynamicDeployListForcedErrorFakeClientInterfaces(ctx context.Context, t *testing.T, ns, name string) (kubernetes.Interface, *dynamicfake.FakeDynamicClient) {
	realClientset := fake.NewSimpleClientset()
	fakeDisc := &fakeHappyDiscovery{discoveryfake.FakeDiscovery{Fake: &realClientset.Fake}}
	clientset := &fakeClientset{Interface: realClientset, discovery: fakeDisc}

	scheme := getNamespaceTestSchema(t)
	namespace := defineNamespaceObject(ns)
	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	deployment := defineDeployObject(ns, name)
	_, err = clientset.AppsV1().Deployments("test-namespace").Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test deployment: %v", err)
	}

	listKinds := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}: "DeploymentList",
		{Group: "", Version: "v1", Resource: "namespaces"}:      "NamespaceList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds)
	dynamicClient.PrependReactor("list", "deployments", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, fmt.Errorf("forced error")
	})

	return clientset, dynamicClient
}

type GetFakeClientInterfacesFunc func(ctx context.Context, t *testing.T, ns, name string) (kubernetes.Interface, *dynamicfake.FakeDynamicClient)

func Test_namespaces_IsErrorOrNamespaceContainsResources(t *testing.T) {
	tests := []struct {
		name string

		objName        string
		namespaceName  string
		ctx            context.Context
		getClientsFunc GetFakeClientInterfacesFunc
		filterOpts     *filters.Options

		expectedReturn bool
		expectedError  bool
	}{
		{
			name: "deployment exists, no errors, ignoring secrets and configmaps",

			objName:        "test-object",
			namespaceName:  "test-namespace",
			ctx:            context.TODO(),
			getClientsFunc: createHappyDeployFakeClientInterfaces,
			filterOpts: &filters.Options{
				IgnoreResourceTypes: []string{"configmaps", "secrets"},
			},

			expectedReturn: true,
			expectedError:  false,
		},
		{
			name: "deployment exists, no errors, ignoring deployments",

			objName:        "test-object",
			namespaceName:  "test-namespace",
			ctx:            context.TODO(),
			getClientsFunc: createHappyDeployFakeClientInterfaces,
			filterOpts: &filters.Options{
				IgnoreResourceTypes: []string{"deployments"},
			},

			expectedReturn: false,
			expectedError:  false,
		},
		{
			name: "deployment list is empty, no errors, ignoring secrets",

			objName:        "test-object",
			namespaceName:  "test-namespace",
			ctx:            context.TODO(),
			getClientsFunc: createHappyEmptyFakeClientInterfaces,
			filterOpts: &filters.Options{
				IgnoreResourceTypes: []string{"secrets"},
			},

			expectedReturn: false,
			expectedError:  false,
		},
		{
			name: "deployment list is empty, error in discovery, ignoring secrets",

			objName:        "test-object",
			namespaceName:  "test-namespace",
			ctx:            context.TODO(),
			getClientsFunc: createUnhappyDiscoveryFakeClientInterfaces,
			filterOpts: &filters.Options{
				IgnoreResourceTypes: []string{"secrets"},
			},

			expectedReturn: true,
			expectedError:  true,
		},
		{
			name: "imitate broken APIResourceList, error in discovery, ignoring secrets",

			objName:        "test-object",
			namespaceName:  "test-namespace",
			ctx:            context.TODO(),
			getClientsFunc: createBrokenAPIResourceListDiscoveryFakeClientInterfaces,
			filterOpts: &filters.Options{
				IgnoreResourceTypes: []string{"secrets"},
			},

			expectedReturn: true,
			expectedError:  true,
		},
		{
			name: "imitate failed list deployments call, error in dynamic client, ignoring secrets",

			objName:        "test-object",
			namespaceName:  "test-namespace",
			ctx:            context.TODO(),
			getClientsFunc: createDynamicDeployListForcedErrorFakeClientInterfaces,
			filterOpts: &filters.Options{
				IgnoreResourceTypes: []string{"secrets"},
			},

			expectedReturn: false,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset, dynamicClient := tt.getClientsFunc(tt.ctx, t, tt.namespaceName, tt.objName)
			got, err := isErrorOrNamespaceContainsResources(tt.ctx, clientset, dynamicClient, tt.namespaceName, tt.filterOpts)
			if (err != nil) != tt.expectedError {
				t.Errorf("isErrorOrNamespaceContainsResources() = expected error: %t, got: '%v'", tt.expectedError, err)
			}
			if got != tt.expectedReturn {
				t.Errorf("isErrorOrNamespaceContainsResources() = got %t, want %t", got, tt.expectedReturn)
			}
		})
	}
}