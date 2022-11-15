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

package utils_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2/klogr"

	"github.com/projectsveltos/classifier-agent/pkg/utils"
)

const (
	version25 = "v1.25.2"
	version24 = "v1.24.2"
)

var _ = Describe("Manager", func() {
	var scheme *runtime.Scheme

	BeforeEach(func() {
		var err error
		scheme, err = setupScheme()
		Expect(err).ToNot(HaveOccurred())
	})

	It("IsControlPlaneNode returns true if node is a control plane node", func() {
		node := getNode(version24)
		Expect(utils.IsControlPlaneNode(node)).To(BeTrue())
	})

	It("IsControlPlaneNode returns false if node is not a control plane node", func() {
		node := getNode(version24)
		node.Labels = nil
		Expect(utils.IsControlPlaneNode(node)).To(BeFalse())
	})

	It("GetKubernetesVersion returns cluster Kubernetes version", func() {
		node1 := getNode(version24)
		node2 := getNode(version25)
		node2.Labels = nil
		node3 := getNode(version25)
		node3.Labels = nil

		initObjects := []client.Object{
			node1,
			node2,
			node3,
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjects...).Build()

		version, err := utils.GetKubernetesVersion(context.TODO(), c, klogr.New())
		Expect(err).To(BeNil())
		Expect(version).To(Equal(version24))
	})
})
