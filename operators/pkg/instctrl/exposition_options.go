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

package instctrl

// ExpositionMode controls which routing resource is reconciled for instance GUI exposition.
type ExpositionMode string

const (
	// ExpositionModeIngress reconciles networking.k8s.io/v1 Ingress resources.
	ExpositionModeIngress ExpositionMode = "ingress"
	// ExpositionModeHTTPRoute reconciles gateway.networking.k8s.io/v1 HTTPRoute resources.
	ExpositionModeHTTPRoute ExpositionMode = "httproute"
)

// ExpositionOpts groups the settings used by the instance reconciler for GUI exposition.
type ExpositionOpts struct {
	Mode                   ExpositionMode
	IngressCertificateName string
	GatewayName            string
	GatewayNamespace       string
	GatewaySectionName     string
	GatewayClassName       string
	Compat                 *bool
}

// CompatEnabled returns whether compatibility fallbacks are enabled.
// The default is true when the option is not explicitly configured.
func (o ExpositionOpts) CompatEnabled() bool {
	if o.Compat == nil {
		return true
	}

	return *o.Compat
}
