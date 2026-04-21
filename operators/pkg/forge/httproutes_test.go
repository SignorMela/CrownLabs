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

package forge_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/netgroup-polito/CrownLabs/operators/pkg/forge"
)

var _ = Describe("HTTPRoutes", func() {
	Describe("The forge.HTTPRouteSpec function", func() {
		const (
			host             = "crownlabs.example.com"
			path             = "/instance/uuid/environment"
			serviceName      = "service-name"
			gatewayName      = "crownlabs-gw"
			gatewayNamespace = "envoy-gateway-system"
			gatewaySection   = "https"
			servicePort      = int32(6080)
		)

		It("Should configure parent refs and backend when rewrite is disabled", func() {
			spec := forge.HTTPRouteSpec(host, path, serviceName, gatewayName, gatewayNamespace, gatewaySection, servicePort, false)

			Expect(spec.Hostnames).To(Equal([]gatewayv1.Hostname{gatewayv1.Hostname(host)}))
			Expect(spec.ParentRefs).To(HaveLen(1))
			Expect(spec.ParentRefs[0].Name).To(Equal(gatewayv1.ObjectName(gatewayName)))
			Expect(*spec.ParentRefs[0].Namespace).To(Equal(gatewayv1.Namespace(gatewayNamespace)))
			Expect(*spec.ParentRefs[0].SectionName).To(Equal(gatewayv1.SectionName(gatewaySection)))

			Expect(spec.Rules).To(HaveLen(1))
			Expect(spec.Rules[0].Matches).To(HaveLen(1))
			Expect(*spec.Rules[0].Matches[0].Path.Type).To(Equal(gatewayv1.PathMatchPathPrefix))
			Expect(*spec.Rules[0].Matches[0].Path.Value).To(Equal(path))
			Expect(spec.Rules[0].BackendRefs).To(HaveLen(1))
			Expect(spec.Rules[0].BackendRefs[0].BackendRef.BackendObjectReference.Name).To(Equal(gatewayv1.ObjectName(serviceName)))
			Expect(*spec.Rules[0].BackendRefs[0].BackendRef.BackendObjectReference.Port).To(Equal(gatewayv1.PortNumber(servicePort)))
			Expect(spec.Rules[0].Filters).To(BeEmpty())
		})

		It("Should configure URL rewrite filter when rewrite is enabled", func() {
			spec := forge.HTTPRouteSpec(host, path, serviceName, "", "", "", servicePort, true)

			Expect(spec.ParentRefs).To(BeEmpty())
			Expect(spec.Rules).To(HaveLen(1))
			Expect(spec.Rules[0].Filters).To(HaveLen(1))
			Expect(spec.Rules[0].Filters[0].Type).To(Equal(gatewayv1.HTTPRouteFilterURLRewrite))
			Expect(spec.Rules[0].Filters[0].URLRewrite).NotTo(BeNil())
			Expect(spec.Rules[0].Filters[0].URLRewrite.Path).NotTo(BeNil())
			Expect(spec.Rules[0].Filters[0].URLRewrite.Path.Type).To(Equal(gatewayv1.PrefixMatchHTTPPathModifier))
			Expect(*spec.Rules[0].Filters[0].URLRewrite.Path.ReplacePrefixMatch).To(Equal("/"))
		})
	})
})
