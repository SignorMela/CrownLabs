# Gateway Auth YAML Examples

Questi manifest sono stati ripuliti in due aree:

- area attiva: demo essenziale (legacy vs Gateway API)
- area archive: track avanzati sospesi (per-user/extauth)

## Area attiva

- `00-core/`: risorse base Gateway (GatewayClass, Gateway, namespace canary).
- `10-routes/`: HTTPRoute Standard/Exam/Exercise.
- `20-oidc-base/`: secret OIDC + SecurityPolicy base (fase 3).
- `30-local-lab/`: backend demo locale (`whoami`).
- `40-keycloak-local/`: Keycloak locale (richiesto per issuer base OIDC + test JWT 401/200/403).

## Area archive

- `archive/50-advanced-policies/`: policy avanzate storiche (per-user, extauth, reference grant).
- `archive/60-extauth-local/`: bootstrap oauth2-proxy locale e route A/B extauth.
- `archive/70-per-user-ab/`: route/policy A/B per-user.

## Ordine consigliato (base)

1. `00-core/01-gatewayclass-envoy.yaml`
2. `00-core/02-gateway-crownlabs.yaml`
3. `00-core/03-namespace-canary-label.yaml`
4. `30-local-lab/31-demo-backends-p7-demo.yaml` (solo laboratorio locale)
5. `40-keycloak-local/41-namespace-keycloak-local.yaml`
6. `40-keycloak-local/42-keycloak-local-bootstrap.yaml`
7. `10-routes/11-httproute-standard-gui.yaml`
8. `10-routes/13-httproute-exam-noauth.yaml`
9. `10-routes/14-httproute-exercise-noauth.yaml`
10. `20-oidc-base/22-oidc-client-secret.example.yaml`
11. `20-oidc-base/21-securitypolicy-standard-oidc.yaml`

## Track JWT locale

- `40-keycloak-local/41-namespace-keycloak-local.yaml`
- `40-keycloak-local/42-keycloak-local-bootstrap.yaml`
- `40-keycloak-local/44-httproute-standard-gui-localkc-per-user-ab.yaml`
- `40-keycloak-local/45-securitypolicy-standard-per-user-jwt-localkc-ab.yaml`

## Note pratiche

- Adatta namespace, hostnames, route path e backend service names.
- Non applicare insieme policy incompatibili sullo stesso targetRef.
- La policy JWT locale usa `remoteJWKS` puntando a Keycloak locale, cosi' segue automaticamente la rotazione chiavi.
- I manifest in `archive/` non fanno parte della demo pulita corrente.
