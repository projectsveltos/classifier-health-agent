/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/projectsveltos/classifier-agent/internal/test/helpers"
	libsveltosv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
	"github.com/projectsveltos/libsveltos/lib/crd"
	"github.com/projectsveltos/libsveltos/lib/utils"
)

var (
	testEnv *helpers.TestEnvironment
	cancel  context.CancelFunc
	ctx     context.Context
	scheme  *runtime.Scheme
)

var (
	cacheSyncBackoff = wait.Backoff{
		Duration: 100 * time.Millisecond,
		Factor:   1.5,
		Steps:    8,
		Jitter:   0.4,
	}
)

const (
	timeout         = 40 * time.Second
	pollingInterval = 2 * time.Second
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Classification Suite")
}

var _ = BeforeSuite(func() {
	By("bootstrapping test environment")

	ctx, cancel = context.WithCancel(context.TODO())

	var err error
	scheme, err = setupScheme()
	Expect(err).To(BeNil())

	testEnvConfig := helpers.NewTestEnvironmentConfiguration([]string{}, scheme)
	testEnv, err = testEnvConfig.Build(scheme)
	if err != nil {
		panic(err)
	}

	go func() {
		By("Starting the manager")
		err = testEnv.StartManager(ctx)
		if err != nil {
			panic(fmt.Sprintf("Failed to start the envtest manager: %v", err))
		}
	}()

	if synced := testEnv.GetCache().WaitForCacheSync(ctx); !synced {
		time.Sleep(time.Second)
	}

	classifierCRD, err := utils.GetUnstructured(crd.GetClassifierCRDYAML())
	Expect(err).To(BeNil())
	Expect(testEnv.Create(ctx, classifierCRD)).To(Succeed())

	classifierReportCRD, err := utils.GetUnstructured(crd.GetClassifierReportCRDYAML())
	Expect(err).To(BeNil())
	Expect(testEnv.Create(ctx, classifierReportCRD)).To(Succeed())

	Expect(waitForObject(ctx, testEnv.Client, classifierCRD)).To(Succeed())
	Expect(waitForObject(ctx, testEnv.Client, classifierReportCRD)).To(Succeed())

	if synced := testEnv.GetCache().WaitForCacheSync(ctx); !synced {
		time.Sleep(time.Second)
	}
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func randomString() string {
	const length = 10
	return util.RandomString(length)
}

func setupScheme() (*runtime.Scheme, error) {
	s := runtime.NewScheme()
	if err := libsveltosv1alpha1.AddToScheme(s); err != nil {
		return nil, err
	}
	if err := clientgoscheme.AddToScheme(s); err != nil {
		return nil, err
	}
	if err := apiextensionsv1.AddToScheme(s); err != nil {
		return nil, err
	}
	return s, nil
}

// waitForObject waits for the cache to be updated helps in preventing test flakes due to the cache sync delays.
func waitForObject(ctx context.Context, c client.Client, obj client.Object) error {
	// Makes sure the cache is updated with the new object
	objCopy := obj.DeepCopyObject().(client.Object)
	key := client.ObjectKeyFromObject(obj)
	if err := wait.ExponentialBackoff(
		cacheSyncBackoff,
		func() (done bool, err error) {
			if err := c.Get(ctx, key, objCopy); err != nil {
				if apierrors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		}); err != nil {
		return errors.Wrapf(err, "object %s, %s is not being added to the testenv client cache",
			obj.GetObjectKind().GroupVersionKind().String(), key)
	}
	return nil
}

func getClassifierWithKubernetesConstraints() *libsveltosv1alpha1.Classifier {
	return &libsveltosv1alpha1.Classifier{
		ObjectMeta: metav1.ObjectMeta{
			Name: randomString(),
		},
		Spec: libsveltosv1alpha1.ClassifierSpec{
			ClassifierLabels: []libsveltosv1alpha1.ClassifierLabel{
				{Key: randomString(), Value: randomString()},
			},
			KubernetesVersionConstraints: []libsveltosv1alpha1.KubernetesVersionConstraint{
				{
					Comparison: string(libsveltosv1alpha1.OperationEqual),
					Version:    "v1.25.2",
				},
			},
		},
	}
}

func getClassifierWithResourceConstraints() *libsveltosv1alpha1.Classifier {
	return &libsveltosv1alpha1.Classifier{
		ObjectMeta: metav1.ObjectMeta{
			Name: randomString(),
		},
		Spec: libsveltosv1alpha1.ClassifierSpec{
			ClassifierLabels: []libsveltosv1alpha1.ClassifierLabel{
				{Key: randomString(), Value: randomString()},
			},
			DeployedResources: []libsveltosv1alpha1.DeployedResource{
				{
					Group:   randomString(),
					Version: randomString(),
					Kind:    randomString(),
					LabelFilters: []libsveltosv1alpha1.LabelFilter{
						{
							Key:       randomString(),
							Operation: libsveltosv1alpha1.OperationEqual,
							Value:     randomString(),
						},
					},
				},
			},
		},
	}
}

func getControlPlaneNode() *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: randomString(),
			Labels: map[string]string{
				"node-role.kubernetes.io/control-plane": "ok",
			},
		},
		Status: corev1.NodeStatus{
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: "v1.25.0",
			},
		},
	}
}
