# Progetto 7 - Gateway API + Autenticazione (note locali, versione ordinata)

## 0) Come usare queste note

- Questa cartella e' una working area temporanea della fork.
- Per avvio locale CrownLabs (VM dentro CrownLabs), segui anche:
  - `/home/crownlabs/zz-local-project7-notes/progetto7-crownlabs-locale-guida-operativa.md`
- Percorso consigliato: segui prima la sezione 3 (happy path).
- Usa la sezione 4 solo se trovi errori.
- Quando riallinei la fork pubblica puoi eliminare:
  - `zz-local-project7-notes/`

---

## 1) Obiettivo e vincoli

Obiettivo:

- Migrare l'autenticazione L7 da Ingress/NGINX a Gateway API/Envoy.
- Evitare regressioni applicative durante la transizione.

Vincoli funzionali da preservare:

- Scope `Standard`: auth attiva.
- Scope `Exam`: auth disattivata.
- Scope `Exercise`: auth disattivata.

Fuori scope in questa fase:

- TCPRoute/SSH.

---

## 2) Snapshot stato attuale (aggiornare a mano)

- [x] Cilium installato da values locali.
- [x] Gateway API CRD applicati.
- [x] Envoy Gateway controller avviato.
- [ ] CrownLabs locale (operator + automation) avviato in modalita' integrata.
- [ ] GatewayClass/Gateway/HTTPRoute di test applicati.
- [ ] Test auth scope-aware completati.

---

## 3) Runbook essenziale (happy path)

Questa e' la sequenza minima e ordinata. Evita passaggi extra finche' non serve.

### 3.1 Safety check contesto

```bash
kubectl config current-context
kubectl cluster-info
```

Il contesto deve puntare al cluster di laboratorio.

### 3.2 Prerequisiti tool

```bash
command -v kubectl
command -v helm || (curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 && chmod 700 get_helm.sh && ./get_helm.sh)
```

### 3.3 Cilium (se non gia' pronto)

Values file locale: `~/kubernetes-config/p7-cilium-values.yaml`

Contenuto minimo valido:

```yaml
kubeProxyReplacement: true
routingMode: tunnel
tunnelProtocol: vxlan
mtu: 1400
bpf:
  masquerade: false
enableIPv4Masquerade: true
```

Apply:

```bash
APISERVER_IP=$(kubectl get endpoints kubernetes -n default -o jsonpath='{.subsets[0].addresses[0].ip}')
APISERVER_PORT=$(kubectl get endpoints kubernetes -n default -o jsonpath='{.subsets[0].ports[0].port}')

helm repo add cilium https://helm.cilium.io/
helm repo update
helm template cilium cilium/cilium --version 1.18.8 \
  --namespace kube-system \
  -f ~/kubernetes-config/p7-cilium-values.yaml \
  --set k8sServiceHost=${APISERVER_IP} \
  --set k8sServicePort=${APISERVER_PORT} \
  > ~/kubernetes-config/p7-cilium-install.yaml

kubectl apply -f ~/kubernetes-config/p7-cilium-install.yaml
kubectl -n kube-system rollout status ds/cilium --timeout=10m
kubectl -n kube-system get pods -l k8s-app=cilium
```

### 3.4 Gateway API + Envoy Gateway

Baseline da NOTES:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/standard-install.yaml
helm install eg oci://docker.io/envoyproxy/gateway-helm --version v1.1.0 -n envoy-gateway-system --create-namespace
kubectl get pods -n envoy-gateway-system
kubectl get crd | grep gateway
kubectl get gateway
```

Se il pod `envoy-gateway` va in CrashLoop con errore BackendTLSPolicy, applica subito fix compatibilita':

```bash
helm upgrade eg oci://docker.io/envoyproxy/gateway-helm --version v1.7.1 -n envoy-gateway-system --wait --timeout 5m
kubectl -n envoy-gateway-system get pods
kubectl -n envoy-gateway-system logs -l control-plane=envoy-gateway --tail=100
```

### 3.5 Provisioning Gateway (manifests locali)

Comandi previsti da NOTES:

```bash
kubectl apply -f /home/crownlabs/kubernetes-config/vanilla-gateway-class.yaml
kubectl apply -f /home/crownlabs/kubernetes-config/vanilla-gateway.yaml
kubectl apply -f /home/crownlabs/kubernetes-config/vanilla-http-route.yaml
kubectl -n p7-demo get gateway p7-gw
```

Nota: i file sono in `/home/crownlabs/kubernetes-config/`.

### 3.6 Smoke test L7 minimale

```bash
kubectl create namespace p7-demo
kubectl -n p7-demo create deployment whoami --image=traefik/whoami
kubectl -n p7-demo expose deployment whoami --port=80 --target-port=80
kubectl get gateway -A
kubectl get httproute -A
```

Per usare Cilium come LoadBalancer (e avere `Programmed=True`), applica il pool IP dedicato:

```bash
kubectl apply -f /home/crownlabs/kubernetes-config/cilium-lb-pool-p7-gw.yaml
kubectl -n envoy-gateway-system get svc -l gateway.envoyproxy.io/owning-gateway-name=p7-gw,gateway.envoyproxy.io/owning-gateway-namespace=p7-demo -o wide
kubectl -n p7-demo get gateway p7-gw -o jsonpath='{range .status.conditions[*]}{.type}={.status} {.reason}{"\n"}{end}'
curl -sS http://172.23.22.240/ | head -n 5
```

Se `Programmed=False AddressNotAssigned` ma il Service Envoy e' presente, puoi testare via NodePort:

```bash
NODE_IP=$(kubectl get node -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
NODE_PORT=$(kubectl -n envoy-gateway-system get svc -l gateway.envoyproxy.io/owning-gateway-name=p7-gw -o jsonpath='{.items[0].spec.ports[0].nodePort}')
curl -sS "http://${NODE_IP}:${NODE_PORT}/" | head -n 5
```

### 3.7 Network policy trust

```bash
kubectl label namespace envoy-gateway-system crownlabs.polito.it/allow-instance-access=true --overwrite
kubectl get ns envoy-gateway-system --show-labels
```

---

## 4) Troubleshooting rapido (solo casi reali gia' visti)

### 4.1 `error: no objects passed to apply`

Causa tipica: file generato in un path, applicato da un altro path.

Regola: usa sempre lo stesso file tra render e apply.

```bash
> ~/kubernetes-config/p7-cilium-install.yaml
kubectl apply -f ~/kubernetes-config/p7-cilium-install.yaml
```

### 4.2 `yaml: line 2: mapping values are not allowed in this context`

Causa tipica: indentazione errata nel values YAML (chiavi top-level con spazi in piu').

Fix: riallineare le chiavi top-level a colonna 1, con indentazione solo nei blocchi nidificati.

### 4.3 CrashLoop Envoy: `no matches for kind BackendTLSPolicy ... v1alpha3`

Causa: mismatch tra versione chart Envoy e CRD Gateway API disponibili.

Fix pragmatico validato:

```bash
helm upgrade eg oci://docker.io/envoyproxy/gateway-helm --version v1.7.1 -n envoy-gateway-system --wait --timeout 5m
```

Check:

```bash
kubectl -n envoy-gateway-system get pods
kubectl -n envoy-gateway-system logs -l control-plane=envoy-gateway --tail=100
```

### 4.4 Helm bloccato su `pending-install`

```bash
helm status eg -n envoy-gateway-system
helm uninstall eg -n envoy-gateway-system
helm install eg oci://docker.io/envoyproxy/gateway-helm --version v1.7.1 -n envoy-gateway-system --create-namespace --wait --timeout 5m
```

### 4.5 `no accepted gatewayclass` nei log

Il controller e' up, ma non vede una GatewayClass accettata.

Applica il manifesto GatewayClass (`vanilla-gateway-class.yaml`) e ricontrolla.

### 4.6 Heredoc bloccato (prompt `>`)

Se usi `cat <<'EOF'`, la riga finale `EOF` deve essere da sola, senza spazi.

Alternativa robusta: crea i file YAML da editor e poi fai `kubectl apply -f file.yaml`.

### 4.7 Gateway `Programmed=False AddressNotAssigned`

Causa tipica: Service `LoadBalancer` del Gateway senza IP assegnato.

Fix con Cilium LB IPAM:

```bash
kubectl apply -f /home/crownlabs/kubernetes-config/cilium-lb-pool-p7-gw.yaml
kubectl -n envoy-gateway-system get svc -l gateway.envoyproxy.io/owning-gateway-name=p7-gw,gateway.envoyproxy.io/owning-gateway-namespace=p7-demo -o wide
kubectl -n p7-demo get gateway p7-gw -o jsonpath='{range .status.conditions[*]}{.type}={.status} {.reason}{"\n"}{end}'
```

---

## 5) Piano migrazione codice (senza rumore)

Ordine consigliato:

1. Introdurre dual-path con feature flag:
   - path A: Ingress legacy + oauth2-proxy annotations.
   - path B: HTTPRoute + policy auth Envoy.
2. Applicare path B solo a canary (template/namespace limitati).
3. Preservare semantica scope:
   - Standard con auth.
   - Exam/Exercise senza auth.
4. Aggiornare query automazione inattivita' da NGINX a Envoy.
5. Validare rollback immediato verso path A.

---

## 6) Test minimo obbligatorio (Go/No-Go)

- [ ] Standard: redirect OIDC -> callback -> sessione valida.
- [ ] Exam: accesso senza auth.
- [ ] Exercise: accesso senza auth.
- [ ] Logout e refresh token corretti.
- [ ] Error path (issuer down, secret errato, route non agganciata).
- [ ] Nessuna regressione su automazione inattivita'.
- [ ] Rollback testato in locale.

Se uno dei punti sopra fallisce: no-go.

---

## 7) Riferimenti codice essenziali

- `operators/pkg/instctrl/exposition.go`
- `operators/pkg/forge/ingresses.go`
- `operators/pkg/forge/tenant.go`
- `operators/pkg/forge/labels.go`
- `operators/deploy/instance-operator/templates/clusterrole.yaml`
- `operators/cmd/instance-automation/main.go`
- `operators/pkg/instautoctrl/prometheus_utils.go`
- `operators/pkg/instautoctrl/inactivity.go`
- `frontend/src/index.tsx`
- `frontend/src/contexts/AuthContextProvider.tsx`
- `infrastructure/identity-provider/manifests/oauth2-proxy-values.yaml`

---

## 8) Pulizia ambiente locale

```bash
cd /home/crownlabs/CrownLabs/operators
make clean-local
kind delete cluster --name crownlabs-local
```
