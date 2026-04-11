# Progetto 7 - Migrazione pratica auth con Gateway API + Envoy

## Obiettivo

Portare l'autenticazione L7 delle istanze da Ingress NGINX a Gateway API + Envoy, mantenendo:

- Scope Standard: auth attiva.
- Scope Exam: no auth.
- Scope Exercise: no auth.
- Rollback rapido al path legacy.

Prima di questo runbook, leggi i fondamenti in:

- `/home/crownlabs/zz-local-project7-notes/progetto7-gateway-auth-fondamenti-teorici.md`

Per avviare CrownLabs locale in modo affidabile (scenario VM dentro CrownLabs), usa anche:

- `/home/crownlabs/zz-local-project7-notes/progetto7-crownlabs-locale-guida-operativa.md`

## Decisioni operative usate in questo runbook

- Auth target: SecurityPolicy (scelta confermata).
- Canary: per namespace, con label dedicata (assunzione pratica).
- Metriche inattivita' Envoy: da definire dopo discovery nel Prometheus reale (non ancora deciso).
- Tailscale: opzionale. Usarlo solo per test e2e dal laptop (redirect/cookie OIDC); non necessario per debug interno in VM.

## Cartella YAML di supporto

File in `zz-local-project7-notes/gateway-auth-yaml-examples`:

1. `00-gatewayclass-envoy.yaml`
2. `01-gateway-crownlabs.yaml`
3. `02-namespace-canary-label.yaml`
4. `03-httproute-standard-gui.yaml`
5. `04-httproute-standard-standalone-rewrite.yaml`
6. `05-httproute-exam-noauth.yaml`
7. `06-httproute-exercise-noauth.yaml`
8. `07-securitypolicy-standard-oidc.yaml`
9. `08-oidc-client-secret.example.yaml`
10. `09-securitypolicy-standard-per-user-jwt.yaml` (opzionale avanzato)
11. `10-securitypolicy-standard-extauth-oauth2proxy.yaml` (alternativa)
12. `11-referencegrant-extauth-service.yaml` (solo con file 10)

## Fase 0 - Precheck

```bash
kubectl config current-context
kubectl get pods -n envoy-gateway-system
kubectl get gatewayclass
kubectl get crd | grep -E 'gateway.networking.k8s.io|gateway.envoyproxy.io'
```

## Fase 1 - Gateway base e canary namespace

1. Applica GatewayClass e Gateway centrale:

```bash
kubectl apply -f /home/crownlabs/zz-local-project7-notes/gateway-auth-yaml-examples/00-gatewayclass-envoy.yaml
kubectl apply -f /home/crownlabs/zz-local-project7-notes/gateway-auth-yaml-examples/01-gateway-crownlabs.yaml
```

2. Etichetta namespace canary (esempio su `p7-demo`):

```bash
kubectl apply -f /home/crownlabs/zz-local-project7-notes/gateway-auth-yaml-examples/02-namespace-canary-label.yaml
```

3. Verifica:

```bash
kubectl get gateway -n envoy-gateway-system
kubectl get gatewayclass
```

## Fase 2 - HTTPRoute scope-aware (senza auth)

1. Standard (GUI classica):

```bash
kubectl apply -f /home/crownlabs/zz-local-project7-notes/gateway-auth-yaml-examples/03-httproute-standard-gui.yaml
```

2. Standard Standalone con rewrite (se il template richiede rewrite):

```bash
kubectl apply -f /home/crownlabs/zz-local-project7-notes/gateway-auth-yaml-examples/04-httproute-standard-standalone-rewrite.yaml
```

3. Exam/Exercise no-auth:

```bash
kubectl apply -f /home/crownlabs/zz-local-project7-notes/gateway-auth-yaml-examples/05-httproute-exam-noauth.yaml
kubectl apply -f /home/crownlabs/zz-local-project7-notes/gateway-auth-yaml-examples/06-httproute-exercise-noauth.yaml
```

4. Verifica attachment:

```bash
kubectl get httproute -A
kubectl -n p7-demo get httproute standard-gui-route -o yaml | grep -A4 "parents:"
```

## Fase 3 - Auth Standard con SecurityPolicy OIDC

1. Crea Secret OIDC nel namespace della route Standard:

```bash
# Modifica prima il file example con i valori reali
kubectl apply -f /home/crownlabs/zz-local-project7-notes/gateway-auth-yaml-examples/08-oidc-client-secret.example.yaml
```

2. Applica SecurityPolicy OIDC (base):

```bash
kubectl apply -f /home/crownlabs/zz-local-project7-notes/gateway-auth-yaml-examples/07-securitypolicy-standard-oidc.yaml
```

3. Verifica comportamento:

- Route Standard: redirect/login OIDC attivo.
- Route Exam: accesso diretto.
- Route Exercise: accesso diretto.

```bash
kubectl get securitypolicy -A
kubectl describe securitypolicy standard-oidc-policy -n p7-demo
```

## Fase 4 - Per-user (opzionale, dopo stabilizzazione)

Opzione consigliata per iniziare: policy OIDC base (fase 3) + per-user solo dopo canary stabile.

Opzione avanzata con claim JWT (gruppi/utente):

```bash
# Non applicare insieme alla policy base se targetRef e' lo stesso.
kubectl apply -f /home/crownlabs/zz-local-project7-notes/gateway-auth-yaml-examples/09-securitypolicy-standard-per-user-jwt.yaml
```

Nota:

- Il file 09 usa claim JWT (`preferred_username`, `groups`) e richiede allineamento reale con claim Keycloak.
- Verifica claim nel token prima del rollout globale.

## Fase 5 - Alternativa compatibilita' (extAuth su oauth2-proxy)

Se vuoi mantenere oauth2-proxy in linea durante la transizione:

```bash
kubectl apply -f /home/crownlabs/zz-local-project7-notes/gateway-auth-yaml-examples/11-referencegrant-extauth-service.yaml
kubectl apply -f /home/crownlabs/zz-local-project7-notes/gateway-auth-yaml-examples/10-securitypolicy-standard-extauth-oauth2proxy.yaml
```

Usa questa opzione solo se il path oauth2-proxy e i ReferenceGrant sono validati nel cluster.
Se il namespace `crownlabs-instances-auth` non esiste nel tuo ambiente, modifica prima il file `11-referencegrant-extauth-service.yaml`.

## Fase 6 - Inattivita': preparazione query Envoy

Dato che la query non e' ancora decisa, usa questo workflow:

1. Estrarre metriche candidate da Prometheus (Envoy data-plane).
2. Scegliere query equivalente a "last frontend access".
3. Aggiungere nuovi flag `monitoring-envoy-*` in automation.
4. Tenere fallback su NGINX/SSH/WebSSH durante transizione.

Go/No-Go minimo prima di passare oltre:

- Nessuna regressione Standard/Exam/Exercise.
- Nessun incremento anomalo di 4xx/5xx.
- Rollback testato.

## Rollback rapido

1. Disabilitare path Gateway auth (feature flag/controller config) e tornare a Ingress legacy.
2. Lasciare HTTPRoute presenti ma non usate fino a stabilizzazione.
3. Verificare subito login Standard e no-auth su Exam/Exercise.

## Checklist finale pratica

- [ ] Gateway e HTTPRoute canary applicati.
- [ ] SecurityPolicy Standard attiva e funzionante.
- [ ] Exam/Exercise senza auth.
- [ ] Per-user validato (se attivato).
- [ ] Query Envoy inattivita' definita.
- [ ] Rollback verificato con tempi accettabili.
