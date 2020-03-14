// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package e2e

import (
	"bytes"
	goctx "context"
	"fmt"
	"github.com/atlasmap/atlasmap-operator/pkg/config"
	"io"
	"testing"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/extensions/v1beta1"

	"github.com/atlasmap/atlasmap-operator/pkg/apis"
	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"github.com/atlasmap/atlasmap-operator/pkg/util"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Minute * 2
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 30
)

func TestAtlasMap(t *testing.T) {
	atlasMapList := &v1alpha1.AtlasMapList{}

	err := framework.AddToFrameworkScheme(apis.AddToScheme, atlasMapList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	// run subtests
	t.Run("atlasmap-group", func(t *testing.T) {
		t.Run("Cluster", AtlasMapCluster)
	})
}

func atlasMapDeploymentTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	crName := "test-atlasmap-deployment"

	exampleAtlasMap := &v1alpha1.AtlasMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName,
			Namespace: namespace,
		},
		Spec: v1alpha1.AtlasMapSpec{
			Replicas: 1,
		},
	}

	defer f.Client.Delete(goctx.TODO(), exampleAtlasMap)

	if err := f.Client.Create(goctx.TODO(), exampleAtlasMap, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval}); err != nil {
		return err
	}

	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, crName, 1, retryInterval, timeout); err != nil {
		return err
	}

	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, exampleAtlasMap); err != nil {
		return err
	}

	if exampleAtlasMap.Status.Image != config.DefaultConfiguration.GetAtlasMapImage() {
		return fmt.Errorf("expected AtlasMap.Status.Image to be %s but was %s", config.DefaultConfiguration.AtlasMapImage, exampleAtlasMap.Status.Image)
	}

	// Verify a service was created
	atlasMapService := &v1.Service{}
	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, atlasMapService); err != nil {
		return err
	}

	isOpenShift, err := util.IsOpenShift(f.KubeConfig)
	if err != nil {
		return err
	}

	var scheme, host string
	if isOpenShift {
		// Verify a route was created
		atlasMapRoute := &routev1.Route{}
		if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, atlasMapRoute); err != nil {
			return err
		}
		scheme = "https"
		host = atlasMapRoute.Spec.Host
	} else {
		// Verify ingress was created
		atlasMapIngress := &v1beta1.Ingress{}
		if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, atlasMapIngress); err != nil {
			return err
		}
		scheme = "http"
		host = atlasMapIngress.Spec.Rules[0].Host
	}

	expectedURL := fmt.Sprintf("%s://%s", scheme, host)
	if exampleAtlasMap.Status.URL != expectedURL {
		return fmt.Errorf("expected AtlasMap.Status.URL to be %s but was %s", expectedURL, exampleAtlasMap.Status.URL)
	}

	return nil
}

func atlasMapScaleTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	crName := "test-atlasmap-scale"

	exampleAtlasMap := &v1alpha1.AtlasMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName,
			Namespace: namespace,
		},
		Spec: v1alpha1.AtlasMapSpec{
			Replicas: 1,
		},
	}

	defer f.Client.Delete(goctx.TODO(), exampleAtlasMap)

	if err := f.Client.Create(goctx.TODO(), exampleAtlasMap, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval}); err != nil {
		return err
	}

	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, crName, 1, retryInterval, timeout); err != nil {
		return err
	}

	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, exampleAtlasMap); err != nil {
		return err
	}

	exampleAtlasMap.Spec.Replicas = 3
	if err := f.Client.Update(goctx.TODO(), exampleAtlasMap); err != nil {
		return err
	}

	// wait for deployment to reach 3 replicas
	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, crName, 3, retryInterval, timeout); err != nil {
		return err
	}

	deployment := &appsv1.Deployment{}
	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Namespace: f.Namespace, Name: crName}, deployment); err != nil {
		return err
	}

	replicas := int32(1)
	deployment.Spec.Replicas = &replicas
	if err := f.Client.Update(goctx.TODO(), deployment); err != nil {
		return err
	}

	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, crName, 1, retryInterval, timeout); err != nil {
		return err
	}

	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Namespace: f.Namespace, Name: crName}, exampleAtlasMap); err != nil {
		return err
	}

	// Verify update of deployment replicas syncs back to AtlasMap replicas
	if replicas != exampleAtlasMap.Spec.Replicas {
		return fmt.Errorf("expected AtlasMap replicas to be %d but got %d", replicas, exampleAtlasMap.Spec.Replicas)
	}

	return nil
}

func atlasMapImageNameTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	crName := "test-atlasmap-custom-image"
	imageName := "docker.io/atlasmap/atlasmap:1.43"

	exampleAtlasMap := &v1alpha1.AtlasMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName,
			Namespace: namespace,
		},
		Spec: v1alpha1.AtlasMapSpec{
			Replicas: 1,
			Version:  "1.43",
		},
	}

	defer f.Client.Delete(goctx.TODO(), exampleAtlasMap)

	if err := f.Client.Create(goctx.TODO(), exampleAtlasMap, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval}); err != nil {
		return err
	}

	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, crName, 1, retryInterval, timeout); err != nil {
		return err
	}

	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, exampleAtlasMap); err != nil {
		return err
	}

	deployment := &appsv1.Deployment{}
	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, deployment); err != nil {
		return err
	}

	container := deployment.Spec.Template.Spec.Containers[0]
	if container.Image != imageName {
		return fmt.Errorf("expected container image to match %s but got %s", imageName, container.Image)
	}

	return nil
}

func atlasMapResourcesTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()

	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	crName := "test-atlasmap-resources"
	limitCPU := "700m"
	limitMemory := "512Mi"
	requestCPU := "500m"
	requestMemory := "256Mi"

	exampleAtlasMap := &v1alpha1.AtlasMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName,
			Namespace: namespace,
		},
		Spec: v1alpha1.AtlasMapSpec{
			Replicas:      1,
			LimitCPU:      limitCPU,
			LimitMemory:   limitMemory,
			RequestCPU:    requestCPU,
			RequestMemory: requestMemory,
		},
	}

	defer f.Client.Delete(goctx.TODO(), exampleAtlasMap)

	if err := f.Client.Create(goctx.TODO(), exampleAtlasMap, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval}); err != nil {
		return err
	}

	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, crName, 1, retryInterval, timeout*2); err != nil {
		return err
	}

	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, exampleAtlasMap); err != nil {
		return err
	}

	deployment := &appsv1.Deployment{}
	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: namespace}, deployment); err != nil {
		return err
	}

	container := deployment.Spec.Template.Spec.Containers[0]

	if container.Resources.Limits.Cpu().String() != limitCPU {
		return fmt.Errorf("expected CPU limit to match %s but got %s", limitCPU, container.Resources.Limits.Cpu().String())
	}

	if container.Resources.Limits.Memory().String() != limitMemory {
		return fmt.Errorf("expected memory limit to match %s but got %s", limitMemory, container.Resources.Limits.Memory().String())
	}

	if container.Resources.Requests.Cpu().String() != requestCPU {
		return fmt.Errorf("expected CPU request to match %s but got %s", requestCPU, container.Resources.Requests.Cpu().String())
	}

	if container.Resources.Requests.Memory().String() != requestMemory {
		return fmt.Errorf("expected memory request to match %s but got %s", requestMemory, container.Resources.Requests.Memory().String())
	}

	return nil
}

func AtlasMapCluster(t *testing.T) {
	t.Parallel()
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()

	if err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval}); err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}

	t.Log("Initialized cluster resources")

	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	f := framework.Global
	err = routev1.Install(framework.Global.Scheme)
	if err != nil {
		t.Fatal(err)
	}

	if !f.LocalOperator {
		if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "atlasmap-operator", 1, retryInterval, timeout); err != nil {
			t.Fatal(err)
		}
	}

	type testFunction func(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error

	tests := []testFunction{
		atlasMapDeploymentTest,
		atlasMapScaleTest,
		atlasMapImageNameTest,
		atlasMapResourcesTest,
	}

	// run tests
	for _, test := range tests {
		if err = test(t, f, ctx); err != nil {
			logs := operatorLogs(f)
			if len(logs) > 0 {
				t.Log("========== AtlasMap Operator Logs ===========")
				t.Log(logs)
				t.Log("=============================================")
			}
			t.Log("============== Tests Failed =================")
			t.Fatal(err)
		}
	}
}

func operatorLogs(f *framework.Framework) string {
	podListOptions := metav1.ListOptions{
		LabelSelector: "name = atlasmap-operator",
	}
	podList, err := f.KubeClient.CoreV1().Pods(f.Namespace).List(podListOptions)
	if err != nil || len(podList.Items) == 0 {
		return ""
	}

	podLogOptions := v1.PodLogOptions{}
	req := f.KubeClient.CoreV1().Pods(f.Namespace).GetLogs(podList.Items[0].Name, &podLogOptions)
	podLogs, err := req.Stream()
	if err != nil {
		return ""
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return ""
	}
	str := buf.String()

	return str
}
