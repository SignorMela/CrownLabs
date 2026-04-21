// Copyright 2020-2026 Politecnico di Torino
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

package instctrl_test

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	netv1 "k8s.io/api/networking/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	clv1alpha2 "github.com/netgroup-polito/CrownLabs/operators/api/v1alpha2"
	clctx "github.com/netgroup-polito/CrownLabs/operators/pkg/context"
	"github.com/netgroup-polito/CrownLabs/operators/pkg/forge"
	"github.com/netgroup-polito/CrownLabs/operators/pkg/instctrl"
)

var _ = Describe("Instance exposition modes", func() {
	var (
		ctx         context.Context
		instance    clv1alpha2.Instance
		environment clv1alpha2.Environment
		template    clv1alpha2.Template
	)

	const (
		host            = "crownlabs.example.com"
		instanceName    = "kubernetes-0000"
		instanceNS      = "tenant-tester"
		instanceUID     = "dcc6ead1-0040-451b-ba68-787ebfb68640"
		templateName    = "kubernetes"
		environmentName = "control-plane"
		clusterIP       = "1.1.1.1"
	)

	BeforeEach(func() {
		Expect(gatewayv1.AddToScheme(scheme.Scheme)).To(Succeed())

		ctx = ctrl.LoggerInto(context.Background(), logr.Discard())

		environment = clv1alpha2.Environment{Name: environmentName, EnvironmentType: clv1alpha2.ClassContainer, GuiEnabled: true}
		template = clv1alpha2.Template{
			ObjectMeta: metav1.ObjectMeta{Name: templateName, Namespace: templateName},
			Spec: clv1alpha2.TemplateSpec{
				EnvironmentList: []clv1alpha2.Environment{environment},
				Scope:           clv1alpha2.ScopeStandard,
			},
		}
		instance = clv1alpha2.Instance{
			ObjectMeta: metav1.ObjectMeta{Name: instanceName, Namespace: instanceNS, UID: instanceUID},
			Spec: clv1alpha2.InstanceSpec{
				Running:  true,
				Template: clv1alpha2.GenericRef{Name: templateName, Namespace: templateName},
				Tenant:   clv1alpha2.GenericRef{Name: "tester"},
			},
			Status: clv1alpha2.InstanceStatus{
				Environments: []clv1alpha2.InstanceStatusEnv{{Phase: ""}},
			},
		}
	})

	newContext := func() context.Context {
		localCtx, _ := clctx.InstanceInto(ctx, &instance)
		localCtx, _ = clctx.EnvironmentInto(localCtx, &environment)
		localCtx, _ = clctx.TemplateInto(localCtx, &template)
		localCtx = clctx.EnvironmentIndexInto(localCtx, 0)
		return localCtx
	}

	It("Should reconcile HTTPRoute and remove ingress in httproute mode", func() {
		reconciler := instctrl.InstanceReconciler{
			Client: FakeClientWrapped{
				Client:           fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
				serviceClusterIP: clusterIP,
			},
			Scheme:         scheme.Scheme,
			EventsRecorder: record.NewFakeRecorder(32),
			ServiceUrls:    instctrl.ServiceUrls{WebsiteBaseURL: host, InstancesAuthURL: "https://auth.example.com"},
			ExpositionOpts: instctrl.ExpositionOpts{
				Mode:               instctrl.ExpositionModeHTTPRoute,
				GatewayName:        "crownlabs-gw",
				GatewayNamespace:   "envoy-gateway-system",
				GatewaySectionName: "https",
			},
		}

		Expect(reconciler.EnforceInstanceExposition(newContext())).To(Succeed())

		objName := forge.NamespacedNameWithSuffix(&instance, environment.Name)

		route := gatewayv1.HTTPRoute{}
		Expect(reconciler.Get(newContext(), objName, &route)).To(Succeed())
		Expect(route.Spec).To(Equal(forge.HTTPRouteSpec(host, forge.IngressGUICleanPath(&instance, &environment),
			objName.Name, "crownlabs-gw", "envoy-gateway-system", "https", forge.GUIPortNumber, false)))

		ingress := netv1.Ingress{}
		err := reconciler.Get(newContext(), objName, &ingress)
		Expect(kerrors.IsNotFound(err)).To(BeTrue())
	})

	It("Should reconcile ingress and remove stale httproute in ingress mode", func() {
		objName := forge.NamespacedNameWithSuffix(&instance, environment.Name)
		staleRoute := gatewayv1.HTTPRoute{ObjectMeta: forge.NamespacedNameToObjectMeta(objName)}

		reconciler := instctrl.InstanceReconciler{
			Client: FakeClientWrapped{
				Client: fake.NewClientBuilder().
					WithScheme(scheme.Scheme).
					WithObjects(&staleRoute).
					Build(),
				serviceClusterIP: clusterIP,
			},
			Scheme:         scheme.Scheme,
			EventsRecorder: record.NewFakeRecorder(32),
			ServiceUrls:    instctrl.ServiceUrls{WebsiteBaseURL: host, InstancesAuthURL: "https://auth.example.com"},
			ExpositionOpts: instctrl.ExpositionOpts{Mode: instctrl.ExpositionModeIngress},
		}

		Expect(reconciler.EnforceInstanceExposition(newContext())).To(Succeed())

		route := gatewayv1.HTTPRoute{}
		err := reconciler.Get(newContext(), types.NamespacedName{Name: objName.Name, Namespace: objName.Namespace}, &route)
		Expect(kerrors.IsNotFound(err)).To(BeTrue())

		ingress := netv1.Ingress{}
		Expect(reconciler.Get(newContext(), objName, &ingress)).To(Succeed())
	})
})
