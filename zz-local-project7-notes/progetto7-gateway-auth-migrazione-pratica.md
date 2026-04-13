# Progetto 7 - Demo pulita auth: da Ingress legacy a Gateway API

## Obiettivo

Mostrare in modo semplice e ripetibile:

- stato legacy: authentication legata a Ingress nel codice Go degli operator
- stato nuovo: authentication su Gateway API + SecurityPolicy (senza modificare Go)

## Scope attivo della demo

Attivo adesso:

- demo base OIDC su route Standard con Keycloak locale
- verifica rapida con output raw da terminale
- track JWT locale (401/200/403) per confronto tecnico

Archiviato (fuori dal flusso demo corrente):

- per-user A/B
- extauth oauth2-proxy
- policy avanzate sperimentali

Percorso archivio:

- /home/crownlabs/CrownLabs/zz-local-project7-notes/gateway-auth-yaml-examples/archive/
- /home/crownlabs/CrownLabs/zz-local-project7-notes/archive-auth/docs/

## Mappa rapida file attivi

Guida struttura completa:

- /home/crownlabs/CrownLabs/zz-local-project7-notes/00-progetto7-auth-demo-struttura.md

Discorso studio/esposizione (componenti + output):

- /home/crownlabs/CrownLabs/zz-local-project7-notes/09-progetto7-discorso-auth-legacy-vs-keycloak.md

Script demo:

- /home/crownlabs/CrownLabs/zz-local-project7-notes/scripts/auth-tests/README.md

Report finale presentabile:

- /home/crownlabs/CrownLabs/zz-local-project7-notes/07-progetto7-report-migrazione-auth-ingress-gateway.md

## Fase 1 - Evidenza legacy Ingress (Go)

Comandi:

```bash
cd /home/crownlabs/CrownLabs

grep -RInE --include='*.go' "netv1\.Ingress|IngressSpec|IngressAuthenticationAnnotations|nginx\.ingress\.kubernetes\.io/auth" operators/pkg/instctrl operators/pkg/forge
```

File chiave legacy:

- operators/pkg/instctrl/exposition.go
- operators/pkg/forge/ingresses.go

## Fase 2 - Applica demo nuova (Gateway + OIDC)

```bash
kubectl apply -f /home/crownlabs/CrownLabs/zz-local-project7-notes/gateway-auth-yaml-examples/00-core/01-gatewayclass-envoy.yaml
kubectl apply -f /home/crownlabs/CrownLabs/zz-local-project7-notes/gateway-auth-yaml-examples/00-core/02-gateway-crownlabs.yaml
kubectl apply -f /home/crownlabs/CrownLabs/zz-local-project7-notes/gateway-auth-yaml-examples/00-core/03-namespace-canary-label.yaml
kubectl apply -f /home/crownlabs/CrownLabs/zz-local-project7-notes/gateway-auth-yaml-examples/30-local-lab/31-demo-backends-p7-demo.yaml
kubectl apply -f /home/crownlabs/CrownLabs/zz-local-project7-notes/gateway-auth-yaml-examples/40-keycloak-local/41-namespace-keycloak-local.yaml
kubectl apply -f /home/crownlabs/CrownLabs/zz-local-project7-notes/gateway-auth-yaml-examples/40-keycloak-local/42-keycloak-local-bootstrap.yaml
kubectl apply -f /home/crownlabs/CrownLabs/zz-local-project7-notes/gateway-auth-yaml-examples/10-routes/11-httproute-standard-gui.yaml
kubectl apply -f /home/crownlabs/CrownLabs/zz-local-project7-notes/gateway-auth-yaml-examples/10-routes/12-httproute-standard-standalone-rewrite.yaml
kubectl apply -f /home/crownlabs/CrownLabs/zz-local-project7-notes/gateway-auth-yaml-examples/10-routes/13-httproute-exam-noauth.yaml
kubectl apply -f /home/crownlabs/CrownLabs/zz-local-project7-notes/gateway-auth-yaml-examples/10-routes/14-httproute-exercise-noauth.yaml

kubectl apply -f /home/crownlabs/CrownLabs/zz-local-project7-notes/gateway-auth-yaml-examples/20-oidc-base/22-oidc-client-secret.example.yaml
kubectl apply -f /home/crownlabs/CrownLabs/zz-local-project7-notes/gateway-auth-yaml-examples/20-oidc-base/21-securitypolicy-standard-oidc.yaml
```

## Fase 3 - Test demo pulita

Test OIDC base (no token, output raw):

```bash
GW_IP=$(kubectl -n envoy-gateway-system get gateway crownlabs-gw -o jsonpath='{.status.addresses[0].value}')

curl -k -i --resolve crownlabs.polito.it:443:${GW_IP} \
	https://crownlabs.polito.it/instance/cgvt7/vscode
```

Output atteso (grezzo):

- risposta HTTP `302` sulla route Standard
- header `location` verso issuer locale `keycloak-local...svc.cluster.local`

Test standalone (opzionale):

```bash
curl -k -i --resolve crownlabs.polito.it:443:${GW_IP} \
	https://crownlabs.polito.it/instance/cgvt7/standalone
```

Atteso: HTTP `200`.

## Fase 4 - Track opzionale Keycloak locale (se serve demo JWT)

Quando usarlo:

- se vuoi dimostrare in modo deterministico allow/deny via bearer token da CLI
- se vuoi evitare dipendenze dal login browser per il controllo authz

Comando rapido:

```bash
/home/crownlabs/CrownLabs/zz-local-project7-notes/scripts/auth-tests/run-keycloak-local-auth-tests.sh
```

Output atteso (grezzo):

- no token => 401
- token autorizzato => 200
- token non autorizzato => 403

## Nota su Keycloak locale

In questa demo il Keycloak locale e' il provider usato anche per il flusso base OIDC.
Non viene usato il Keycloak reale CrownLabs.

## Go/No-Go demo

Go se:

- evidenza legacy nel codice Go raccolta
- response raw base mostra 302 (standard) e 200 (standalone)
- response raw jwt mostra 401/200/403

No-Go se:

- policy non Accepted
- route Standard non sfida OIDC
- test script non passano
