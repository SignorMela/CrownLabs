# Progetto 7 - Fondamenti teorici (auth, inattivita', OIDC, Keycloak)

## Glossario CrownLabs: Workspace, Tenant, Template, Instance

- **Workspace**: rappresenta un gruppo di lavoro o corso (es: workspace di un insegnamento o laboratorio). Definisce le risorse disponibili (CPU, RAM, numero di istanze) e raggruppa Template e Tenant. È la “scatola” che contiene ambienti e utenti.

- **Tenant**: rappresenta un utente CrownLabs (studente, docente, ecc). Contiene le info personali (nome, email) e la lista dei workspace a cui è iscritto, con il relativo ruolo (user/manager). È la “scheda utente” che abilita l’accesso agli ambienti.

- **Template**: è la “ricetta” per creare un ambiente (VM o container) preconfigurato (es: Ubuntu con Python, JupyterLab, VSCode, ecc). Ogni Template è associato a un workspace e descrive le risorse, l’immagine da usare, le opzioni di avvio, ecc.

- **Instance**: è una “istanza” concreta di ambiente creata da un utente a partire da un Template, all’interno di un Workspace. È la VM o il container personale su cui lo studente lavora. Ogni Instance è collegata a un Tenant (utente) e a un Template (modello di ambiente).

In sintesi:

- Workspace = gruppo/corso + risorse
- Tenant = utente CrownLabs
- Template = modello di ambiente pronto all’uso
- Instance = ambiente reale, creato per un utente, pronto all’uso

## Perche' questo documento

Questo file e' pensato per dare una base chiara prima dell'implementazione.
L'obiettivo e' capire il modello CrownLabs attuale e il perche' della migrazione verso Gateway API + Envoy.

---

## 1) Mappa mentale di CrownLabs (versione semplice)

Quando un utente usa CrownLabs, entrano in gioco questi blocchi:

1. Frontend web: mostra dashboard, istanze, pulsanti Connect/SSH/Drive.
2. Operator (Go): crea risorse Kubernetes delle istanze (Service, Ingress oggi; HTTPRoute nel target).
3. Data plane L7: oggi NGINX Ingress, target Envoy Gateway.
4. Auth gateway: oggi oauth2-proxy, con Keycloak come Identity Provider.
5. Monitoring/automation: controller di inattivita' che decide stop/delete automatico delle istanze.

In pratica: l'operator crea l'esposizione, il data plane riceve traffico, il layer auth decide chi passa, l'automation controlla uso/idle.

---

## 2) OAuth2, OIDC, Keycloak, oauth2-proxy: differenze pratiche

### OAuth2

OAuth2 e' un protocollo di delega autorizzativa.
Serve a ottenere access token per chiamare API.
Da solo non e' pensato come protocollo identita' utente completo.

### OIDC

OIDC e' un layer di identita' costruito sopra OAuth2.
Aggiunge ID token e claim utente (es. preferred_username, email, groups).
Se devi sapere "chi e' l'utente", usi OIDC.

### Keycloak

Keycloak e' l'Identity Provider (IdP):

- gestisce login,
- emette token OIDC,
- pubblica endpoint di discovery e JWKS,
- contiene utenti, gruppi, ruoli.

### oauth2-proxy

oauth2-proxy e' un gateway di autenticazione davanti ai servizi:

- se utente non autenticato: redirect a Keycloak,
- se autenticato: valida sessione/cookie,
- inoltra la richiesta a backend.

Nel modello CrownLabs attuale, NGINX usa annotation auth-url/auth-signin per delegare il check a oauth2-proxy.

---

## 3) Come funziona oggi l'auth in CrownLabs

Modello attuale (semplificato):

1. L'operator crea Service + Ingress per ambiente.
2. Se scope Standard, aggiunge annotation auth-url/auth-signin sull'Ingress.
3. Se scope Exam/Exercise, non aggiunge annotation auth.
4. NGINX, per Standard, chiama oauth2-proxy prima di inoltrare al backend.
5. oauth2-proxy usa Keycloak OIDC per login/sessione.

Questa logica scope-aware e' il vincolo principale da preservare durante la migrazione.

---

## 4) Cosa cambia con Gateway API + Envoy

Con Gateway API + Envoy sposti la logica L7 da annotation NGINX a risorse policy dedicate.

Pattern target:

1. Gateway centralizzato (entrypoint).
2. HTTPRoute per istanza/path.
3. SecurityPolicy attaccata alla route Standard (auth attiva).
4. Nessuna policy auth su Exam/Exercise (no-auth).

Vantaggi:

- modello piu' dichiarativo e portabile,
- separazione migliore tra routing e security,
- meno dipendenza da annotation specifiche NGINX.

---

## 5) Per-user auth: cosa significa davvero

Per-user significa applicare regole non solo "utente autenticato", ma "quale utente" o "quale gruppo".

Esempi:

- consenti solo utenti nel gruppo crownlabs-admin,
- consenti utente specifico preferred_username=federico.farci,
- nega tutti gli altri.

In pratica, con SecurityPolicy puoi usare claim JWT (groups, preferred_username, ecc.).
Questo richiede attenzione su:

- claim reali emessi da Keycloak,
- audience/issuer corretti,
- allineamento tra token e regole policy.

Nota importante:

- Per-user e scope risolvono problemi diversi.
- Scope dice se l'environment deve avere auth o no.
- Per-user dice chi puo' passare quando auth e' attiva.

---

## 6) Inattivita' in CrownLabs: perche' e' cruciale nella migrazione

L'automation inattivita' fa questo:

1. Verifica che Prometheus sia raggiungibile.
2. Prova a ricavare "ultima attivita'" da metriche:
   - NGINX (frontend access),
   - SSH/WebSSH (fallback).
3. Aggiorna annotation di last activity sull'Instance.
4. Se supera timeout, invia warning e poi stop/delete in base alla policy.

Perche' la migrazione impatta:

- Se togli dipendenza NGINX senza query equivalenti Envoy, rischi falsi negativi o mancati stop.
- Quindi serve una fase transitoria con doppia sorgente metriche.

Regola pratica:

- prima parity metriche,
- poi cutover completo.

---

## 7) Dobbiamo far partire CrownLabs in locale?

Risposta breve: si', almeno in modalita' ridotta, se vuoi migrazione affidabile.

Guida operativa dedicata (scenario VM dentro CrownLabs):

- `/home/crownlabs/zz-local-project7-notes/progetto7-crownlabs-locale-guida-operativa.md`

### Opzione A - Solo lab minimale (non sufficiente da sola)

- Gateway + whoami + policy.
- Utile per capire CRD e comportamento Envoy.
- Non valida davvero integrazione operator/frontend/automation.

### Opzione B - Locale integrato (consigliata)

- Cluster locale,
- CRD CrownLabs,
- operator avviati localmente,
- route/policy canary,
- test scope Standard/Exam/Exercise.

Questa e' la soglia minima realistica per migrazione senza sorprese.

### Opzione C - Full stack locale (massima fedelta')

- anche auth stack completo (Keycloak/oauth2-proxy),
- monitoring completo.

Piu' costosa, ma migliore per test e2e completi.

Conclusione pratica:

- Non accontentarti del solo test lab HTTPRoute.
- Per il piano auth serve almeno Opzione B.

Nota su Tailscale:

- Non e' obbligatorio per iniziare.
- Diventa utile quando vuoi testare dal laptop redirect/cookie/sessione OIDC in condizioni piu' vicine al reale.
- Se richiede troppo setup (redirect URI non allineate, DNS non pronto), rimandalo dopo la stabilizzazione canary.

---

## 8) Strategia consigliata per imparare e implementare insieme

1. Capire flusso attuale (Ingress + oauth2-proxy + Keycloak).
2. Capire flusso target (Gateway + HTTPRoute + SecurityPolicy).
3. Implementare canary scope-aware.
4. Aggiungere per-user su Standard come step successivo.
5. Chiudere il gap metriche inattivita'.
6. Solo dopo: rollout progressivo e deprecazione legacy.

---

## 9) Cose da ricordare durante il lavoro

- Non cambiare semantica scope.
- Non rompere path URL usati dal frontend.
- Non togliere rollback rapido.
- Non migrare inattivita' "a meta'": o hai fallback robusto o aspetti.
- Validare prima canary, poi produzione.
