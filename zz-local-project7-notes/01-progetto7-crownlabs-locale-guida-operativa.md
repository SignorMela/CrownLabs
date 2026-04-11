# Progetto 7 - CrownLabs locale fatto bene (in VM dentro CrownLabs)

## Obiettivo

Avviare CrownLabs in locale in modo affidabile per testare davvero la migrazione auth Gateway API + Envoy, evitando test parziali che non coprono i controller reali.

Scenario considerato:

- stai lavorando in una VM che gira dentro CrownLabs,
- repository CrownLabs gia' clonata,
- vuoi testare Standard/Exam/Exercise e percorso auth senza perdere tempo.

---

## 1) Decisione rapida: cosa avviare davvero

### Modalita' A - Solo lab Gateway (non basta)

- GatewayClass/Gateway/HTTPRoute + whoami.
- Utile per routing L7.
- Non valida i reconcile CrownLabs.

### Modalita' B - Locale integrato (consigliata)

- CRD CrownLabs + operator locali (instance + automation).
- Canary Gateway auth sopra il cluster.
- E' la soglia minima per test credibili.

### Modalita' C - Full locale con auth stack completo (opzionale)

- Aggiungi Keycloak/oauth2-proxy/monitoring completi.
- Utile prima del rollout finale, ma non obbligatoria subito.

Decisione operativa consigliata: parti da Modalita' B.

---

## 2) Prerequisiti nella VM

```bash
kubectl config current-context
kubectl cluster-info
command -v go
command -v make
command -v kubectl
command -v helm
```

Se `helm` manca:

```bash
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 get_helm.sh
./get_helm.sh
```

Nota:

- i target `make` installano automaticamente `controller-gen` se non presente,
- i target locali usano label `crownlabs.polito.it/operator-selector=local`.
- in alcuni ambienti i sample del repository non sono allineati alle CRD correnti (vedi nota in sezione 3).
- messaggio `requires go >= ...; switching to go...` e' normale con toolchain auto-download di Go e non e' un errore.

---

## 3) Bootstrap CrownLabs locale (integrato)

Dalla root degli operatori:

```bash
cd /home/crownlabs/CrownLabs/operators
```

### 3.1 Installa CRD locali

```bash
make install-local
```

Verifica:

```bash
kubectl get crd | grep crownlabs.polito.it
```

### 3.2 Carica dataset minimo compatibile (consigliato)

Applica i manifest minimi compatibili preparati per questo progetto:

```bash
kubectl apply -f /home/crownlabs/zz-local-project7-notes/crownlabs-local-yaml/00-crownlabs-local-minimal-workspace.yaml
kubectl apply -f /home/crownlabs/zz-local-project7-notes/crownlabs-local-yaml/01-crownlabs-local-minimal-tenant.yaml
kubectl apply -f /home/crownlabs/zz-local-project7-notes/crownlabs-local-yaml/02-crownlabs-local-minimal-template.yaml
```

Nota importante su `quota.cpu`:

- nel Workspace usare `cpu: "1"` (stringa quantity), non `cpu: 1`.

### 3.3 Avvia Instance Operator locale (terminale 1)

`run-instance` avvia l'Instance Operator: il controller che riconcilia le `Instance` e crea/aggiorna le risorse Kubernetes dell'ambiente utente (Deployment, Service, esposizione, ecc.).

Percorso safe raccomandato:

```bash
cd /home/crownlabs/CrownLabs/operators
make run-instance
```

Se compare errore su `pcap.h` mancante durante la build, installa il pacchetto corretto:

```bash
sudo apt update
sudo apt install libpcap-dev
```

Attenzione: il pacchetto giusto e' `libpcap-dev` (non `libcap-dev`).

Alternative opzionale se vuoi forzare un domain diverso:

```bash
cd /home/crownlabs/CrownLabs/operators
DOMAIN=crownlabsfake.polito.it make run-instance
```

Perche' questo path:

- evita l'apply automatico dei sample legacy (`samples/`) che puo' fallire su CRD nuove.

### 3.4 Avvia Instance Automation locale (terminale 2)

`run-instance-automation` avvia l'Instance Automation: il controller che gestisce le automazioni sulle `Instance` (soprattutto inattivita', e opzionalmente termination/submission se abilitate).

Percorso safe raccomandato:

```bash
cd /home/crownlabs/CrownLabs/operators
make run-instance-automation
```

Nota porta health probe:

- in questo repository locale, `make run-instance` usa `metrics=:8080` e `health=:8081`.
- in questo repository locale, `make run-instance-automation` usa `metrics=:8083` e `health=:8082`.
- i due processi possono quindi convivere sullo stesso host.
- per debug rapido delle porte: `ss -ltnp | grep 808`.

Se la porta resta occupata da un processo precedente:

```bash
ss -ltnp | grep 808
pkill -f 'cmd/instance-operator/main.go' || true
pkill -f 'cmd/instance-automation/main.go' || true
ss -ltnp | grep 808 || echo 'porte controller libere'
```

### 3.5 Check funzionali minimi

```bash
kubectl get tenants.crownlabs.polito.it,workspaces.crownlabs.polito.it,templates.crownlabs.polito.it -A
kubectl get ns --show-labels | grep operator-selector
kubectl -n tenant-john-doe get pods
```

Se i controller sono avviati bene, vedi reconcile nei log dei due terminali.

Check operativo pod (raccomandato):

```bash
kubectl -n tenant-john-doe get pods
kubectl -n tenant-john-doe describe pod <pod-name>
kubectl -n tenant-john-doe logs <pod-name> --previous
```

Nota: senza `-n tenant-john-doe`, `kubectl describe pod ...` cerca nel namespace `default` e puo' dare `NotFound`.

### 3.5.b Se vedi molti errori `Template ... not found` o `mydrive-info not found`

Questo succede spesso se nel cluster sono rimaste risorse legacy (`samples/`) non allineate ai manifest minimi locali.

Pulizia consigliata:

```bash
kubectl -n tenant-john-doe delete instance instance-jupyterlab instance-vscode-c-cpp-persistent instance-vscode-python-proxyenabled --ignore-not-found
kubectl -n tenant-john-doe delete instancesnapshot green-tea-6831-snapshot --ignore-not-found
```

Re-apply del template locale (ora con `mountMyDriveVolume: false`):

```bash
kubectl apply -f /home/crownlabs/zz-local-project7-notes/crownlabs-local-yaml/02-crownlabs-local-minimal-template.yaml
```

Se vuoi ricreare una sola istanza minimale pulita:

```bash
kubectl apply -f /home/crownlabs/zz-local-project7-notes/crownlabs-local-yaml/03-crownlabs-local-minimal-instance.yaml
```

### 3.5.c CrashLoop comune: immagine non compatibile con policy non-root

Se usi `nginx:alpine` in ambiente Standalone, puoi vedere:

- `mkdir() "/var/cache/nginx/client_temp" failed (13: Permission denied)`

perche' CrownLabs forza `runAsNonRoot`.

Nel template minimale locale e' gia' applicata una scelta compatibile:

- `traefik/whoami:v1.10.1` con `startupArgs: ["--port", "6080"]`
- `rewriteURL: true` (readiness su `/`)

Per aggiornare il cluster locale:

```bash
kubectl apply -f /home/crownlabs/zz-local-project7-notes/crownlabs-local-yaml/02-crownlabs-local-minimal-template.yaml
kubectl -n tenant-john-doe delete instance green-tea-6831 --ignore-not-found
kubectl apply -f /home/crownlabs/zz-local-project7-notes/crownlabs-local-yaml/03-crownlabs-local-minimal-instance.yaml
kubectl -n tenant-john-doe get pods -w
```

### 3.6 Nota su `make run-instance-local`

Se esegui `make run-instance-local`, quel target fa anche `kubectl apply -f ./samples/`.
Su alcune versioni CRD questo fallisce per incompatibilita' note, ad esempio:

- `unknown field spec.environmentList[0].mode`,
- nome ambiente non valido (`dark-coffee-1` troppo lungo per il vincolo),
- errore CEL su `spec.quota.cpu`.

Questi errori non bloccano il progetto: usa il percorso safe (3.2 -> 3.5).

---

## 4) Aggancio migrazione Gateway/Auth

Una volta su Modalita' B, applica il canary auth del progetto 7:

- runbook pratico: `/home/crownlabs/zz-local-project7-notes/progetto7-gateway-auth-migrazione-pratica.md`
- YAML examples: `/home/crownlabs/zz-local-project7-notes/gateway-auth-yaml-examples/`

Check Gateway:

```bash
kubectl get gatewayclass
kubectl get gateway -A
kubectl get httproute -A
kubectl get securitypolicy -A
```

Se `Programmed=False AddressNotAssigned`, applica il pool LB Cilium gia' preparato:

```bash
kubectl apply -f /home/crownlabs/kubernetes-config/cilium-lb-pool-p7-gw.yaml
```

---

## 5) Strategia Tailscale (VM -> laptop): utile o no?

Risposta breve: utile, ma solo in una fase precisa.

### Quando NON serve

- stai facendo solo debug controller/CRD/reconcile dalla VM,
- testi routing/auth direttamente da VM con curl/browser locale,
- non devi validare UX login dal laptop.

In questo caso: evita Tailscale e resta semplice.

### Quando serve davvero

- vuoi testare login OIDC dal browser del portatile,
- vuoi validare redirect/cookie/sessione da un client esterno alla VM,
- vuoi provare un percorso "quasi reale" prima del rollout.

In questo caso Tailscale e' sensato.

### Percorso minimo Tailscale (consigliato)

1. Installa Tailscale su VM e laptop.
2. Collega entrambi alla stessa tailnet.
3. Nella VM esponi temporaneamente il servizio Gateway con port-forward:

```bash
kubectl -n envoy-gateway-system get svc -l gateway.envoyproxy.io/owning-gateway-name=p7-gw
kubectl -n envoy-gateway-system port-forward svc/<gateway-service-name> 8080:80 --address 0.0.0.0
```

4. Dal laptop apri:

```text
http://<tailscale-ip-vm>:8080
```

5. Se vuoi test OIDC serio, allinea `redirect_uri` e host consentiti in Keycloak sul nome host che usi dal laptop.

### Limite importante

Senza allineamento `redirect_uri` in Keycloak, il test auth da laptop puo' fallire anche se Gateway e policy sono corretti.

---

## 6) Piano pratico consigliato (ordine giusto)

1. Porta su Modalita' B (operator locali attivi).
2. Esegui canary Gateway/Auth su namespace limitato.
3. Valida scope:
   - Standard con auth,
   - Exam no-auth,
   - Exercise no-auth.
4. Solo dopo, prova test da laptop con Tailscale.
5. Se non aggiunge segnale utile, disattivalo e torna al path semplice.

---

## 7) Go / No-Go

Go solo se tutti questi check passano:

- reconcile stabile dei controller locali,
- route Gateway attaccate correttamente,
- SecurityPolicy Standard effettiva,
- nessuna regressione su Exam/Exercise,
- rollback verificato.

No-Go se:

- test dipendono da workaround non ripetibili,
- auth da laptop fallisce per redirect non allineati,
- inattivita' non osservabile con metriche coerenti.

---

## 8) Comandi di cleanup locale

```bash
cd /home/crownlabs/CrownLabs/operators
make clean-local
```

Se ci sono finalizer bloccanti:

```bash
cd /home/crownlabs/CrownLabs/operators
make force-clean-local
```

Usa `force-clean-local` solo quando il cleanup standard non basta.
