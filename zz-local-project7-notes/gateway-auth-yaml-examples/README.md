# Gateway Auth YAML Examples

Questi file sono esempi operativi per la migrazione auth Gateway API + Envoy.

Ordine consigliato (base):

1. 00-gatewayclass-envoy.yaml
2. 01-gateway-crownlabs.yaml
3. 02-namespace-canary-label.yaml
4. 03-httproute-standard-gui.yaml
5. 05-httproute-exam-noauth.yaml
6. 06-httproute-exercise-noauth.yaml
7. 08-oidc-client-secret.example.yaml (modificare prima `client-secret`)
8. 07-securitypolicy-standard-oidc.yaml

Opzioni avanzate (non insieme alla base se target uguale):

- 09-securitypolicy-standard-per-user-jwt.yaml
- 10-securitypolicy-standard-extauth-oauth2proxy.yaml + 11-referencegrant-extauth-service.yaml

Note pratiche:

- Adatta namespace, hostnames, route path e backend service names.
- Non applicare insieme 07 e 09 sullo stesso targetRef senza una strategia precisa.
- Il file 10 richiede che il Service oauth2-proxy abbia nome/namespace corretti.
- Il file 11 richiede che il namespace oauth2-proxy esista (es. crownlabs-instances-auth).
