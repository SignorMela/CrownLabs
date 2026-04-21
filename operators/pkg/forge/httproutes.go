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

package forge

import (
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// HTTPRouteSpec forges the specification of a Kubernetes HTTPRoute resource.
func HTTPRouteSpec(host, path, serviceName, gatewayName, gatewayNamespace, gatewaySectionName string,
	servicePort int32, rewritePrefixPath bool) gatewayv1.HTTPRouteSpec {
	pathMatchType := gatewayv1.PathMatchPathPrefix
	pathValue := path
	backendPort := gatewayv1.PortNumber(servicePort)

	rule := gatewayv1.HTTPRouteRule{
		Matches: []gatewayv1.HTTPRouteMatch{{
			Path: &gatewayv1.HTTPPathMatch{Type: &pathMatchType, Value: &pathValue},
		}},
		BackendRefs: []gatewayv1.HTTPBackendRef{{
			BackendRef: gatewayv1.BackendRef{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Name: gatewayv1.ObjectName(serviceName),
					Port: &backendPort,
				},
			},
		}},
	}

	if rewritePrefixPath {
		rewriteType := gatewayv1.PrefixMatchHTTPPathModifier
		rewritePrefix := "/"
		rule.Filters = []gatewayv1.HTTPRouteFilter{{
			Type: gatewayv1.HTTPRouteFilterURLRewrite,
			URLRewrite: &gatewayv1.HTTPURLRewriteFilter{
				Path: &gatewayv1.HTTPPathModifier{
					Type:               rewriteType,
					ReplacePrefixMatch: &rewritePrefix,
				},
			},
		}}
	}

	spec := gatewayv1.HTTPRouteSpec{
		Hostnames: []gatewayv1.Hostname{gatewayv1.Hostname(host)},
		Rules:     []gatewayv1.HTTPRouteRule{rule},
	}

	if gatewayName != "" {
		parent := gatewayv1.ParentReference{Name: gatewayv1.ObjectName(gatewayName)}
		if gatewayNamespace != "" {
			namespace := gatewayv1.Namespace(gatewayNamespace)
			parent.Namespace = &namespace
		}
		if gatewaySectionName != "" {
			sectionName := gatewayv1.SectionName(gatewaySectionName)
			parent.SectionName = &sectionName
		}

		spec.ParentRefs = []gatewayv1.ParentReference{parent}
	}

	return spec
}
