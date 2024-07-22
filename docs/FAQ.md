---
title: "The secrets-store-integration module: FAQ"
description: Hashicorp vault example configuration. Example of secret autorotation implementation.
---

## How to set up the Hashicorp Vault as a secret store for use with the secrets-store-integration module:

First of all, we need a root or similiar token and the vault address.
Root token can be obtained during new secrets store initialization.


```bash
export VAULT_TOKEN=xxxxxxxxxxx
export VAULT_ADDR=https://secretstoreexample.com
```

In this guide we provide two ways to obtain needed result: 
- usage of the console version of the Hashicorp Vault (installation guide: https://developer.hashicorp.com/vault/docs/install);
- usage of the curl equivalent command to make a direct requests to the secrets store API.

Enable and create Key-Value storage:

```bash
vault secrets enable -path=secret -version=2 kv
```

or curl equivalent:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request POST \
  --data '{"type":"kv","options":{"version":"2"}}' \
  ${VAULT_ADDR}/v1/sys/mounts/secret
```

Set a secret with a database password:

```bash
vault kv put secret/database-for-python-app password="db-secret-password"
```

or curl equivalent:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request PUT \
  --data '{"data":{"password":"db-secret-password"}}' \
  ${VAULT_ADDR}/v1/secret/data/database-for-python-app
```

Double-check that it is written:

```bash
vault kv get secret/database-for-python-app
```

or curl equivalent:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  ${VAULT_ADDR}/v1/secret/data/database-for-python-app
```

Allow authentication and authorization in the vault with Kubernetes API by defining the authentication path:

```bash
vault auth enable -path=main-kube kubernetes
```

or curl equivalent:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request POST \
  --data '{"type":"kubernetes"}' \
  ${VAULT_ADDR}/v1/sys/auth/main-kube
```

If we have more than one cluster, we need to allow authentication and authorization in the vault with Kubernetes API for the second cluster, defining the second authentication path:

```bash
vault auth enable -path=secondary-kube kubernetes
```

or curl equivalent:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request POST \
  --data '{"type":"kubernetes"}' \
  ${VAULT_ADDR}/v1/sys/auth/secondary-kube
```

Set up Kubernetes API address for each auth point (in that case, it is k8s API server service):

```bash
vault write auth/main-kube/config \
  kubernetes_host="https://api.kube.my-deckhouse.com"
```

or curl equivalent:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request PUT \
  --data '{"kubernetes_host":"https://api.kube.my-deckhouse.com"}' \
  ${VAULT_ADDR}/v1/auth/main-kube/config
```

```bash
vault write auth/secondary-kube/config \
  kubernetes_host="https://10.11.12.10:443"
```

or curl equivalent:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request PUT \
  --data '{"kubernetes_host":"https://10.11.12.10:443"}' \
  ${VAULT_ADDR}/v1/auth/secondary-kube/config
```

Create an internal-app policy in the vault:

```bash
vault policy write backend - <<EOF
path "secret/data/database-for-python-app" {
 capabilities = ["read"]
}
EOF
```

or curl equivalent:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request PUT \
  --data '{"policy":"path \"secret/data/database-for-python-app\" {\n capabilities = [\"read\"]\n}\n"}' \
  ${VAULT_ADDR}/v1/sys/policies/acl/backend
```

Create database role and link it with backend-sa ServiceAccount in "my-namespace1" namespace and "backend" policy:

```bash
vault write auth/main-kube/role/my-namespace1_backend \
   bound_service_account_names=backend-sa \
   bound_service_account_namespaces=my-namespace1 \
   policies=backend \
   ttl=10m
```

or curl equivalent:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request PUT \
  --data '{"bound_service_account_names":"backend-sa","bound_service_account_namespaces":"my-namespace1","policies":"backend","ttl":"10m"}' \
  ${VAULT_ADDR}/v1/auth/main-kube/role/my-namespace1_backend
```

Almost the same for the second k8s cluster:

```bash
vault write auth/secondary-kube/role/my-namespace1_backend \
   bound_service_account_names=backend-sa \
   bound_service_account_namespaces=my-namespace1 \
   policies=backend \
   ttl=10m
```

or curl equivalent:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request PUT \
  --data '{"bound_service_account_names":"backend-sa","bound_service_account_namespaces":"my-namespace1","policies":"backend","ttl":"10m"}' \
  ${VAULT_ADDR}/v1/auth/secondary-kube/role/my-namespace1_backend
```

**The recommended value for TTL of the Kubernetes token is 10m.**

Those settings allow any pod within the "my-namespace1" namespace from both k8s clusters and with the "backend-sa" ServiceAccount to authenticate, authorize, and read secrets inside Vault covered by the backend policy.

## How to use autorotation with the file-mounted secret inside a container without restarting:

The autorotation feature of the secret-store-integration module is enabled by default. Every two minutes, module polls and resyncs mounted secret values if someone changed the secret's value inside the secret store.

Create ServiceAccount ```backend-sa```

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: backend-sa
  namespace: my-namespace1
```

Here we have the example SecretStoreImport definition:

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: SecretsStoreImport
metadata:
 name: python-backend
 namespace: my-namespace1
spec:
 type: CSI
 role: my-namespace1_backend
 files:
   - name: "db-password"
     source:
       path: "secret/data/database-for-python-app"
       key: "password"
```

And the example “backend” Deployment definition, which has the SecretStoreImport as a volume to deliver the database password to the application:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
 name: backend
 namespace: my-namespace1
 labels:
   app: backend
spec:
 selector:
   matchLabels:
     app: backend
 template:
   metadata:
     labels:
       app: backend
   spec:
     serviceAccountName: backend-sa
     containers:
     - image: some/app:0.0.1
       name: backend
       volumeMounts:
       - name: secrets
         mountPath: "/mnt/secrets"
     volumes:
     - name: secrets
       csi:
         driver: secrets-store.csi.deckhouse.io
         volumeAttributes:
           secretsStoreImport: "python-backend"
```

Upon applying this deployment the pod “backend” will be started, inside which we have “secrets” Volume, mounted to /mnt/secrets/ with file “db-password”, containing a password from the Vault.
We have two options to detect changes in a secret file mounted to the pod. The first is to monitor the mtime of the mounted file to detect when it changes. The second option is to monitor filesystem changes with inotify API, which provides a mechanism for monitoring filesystem events. Inotify is a part of the Linux kernel. Many options exist for reacting to detected changes, depending on the architecture and language used. The simplest example is to force the k8s to restart the pod by failing the liveness probe.

To determine if the password has changed inside the Python application using inotify and Python inotify package:

```python
#!/usr/bin/python3

import inotify.adapters

def _main():
    i = inotify.adapters.Inotify()
    i.add_watch('/mnt/secrets-store/db-password')

    for event in i.event_gen(yield_nones=False):
        (_, type_names, path, filename) = event

        if 'IN_MODIFY' in type_names:
            print("file modified")

if __name__ == '__main__':
    _main()
```

The example code to determine if the password has changed inside the Go application using inotify and Go inotify package:

```python
watcher, err := inotify.NewWatcher()
if err != nil {
    log.Fatal(err)
}
err = watcher.Watch("/mnt/secrets-store/db-password")
if err != nil {
    log.Fatal(err)
}
for {
    select {
    case ev := <-watcher.Event:
        if ev == 'InModify' {
        	log.Println("file modified")}
    case err := <-watcher.Error:
        log.Println("error:", err)
    }
}
```

## Secret rotation limitations

A container using `subPath` volume mount will not receive secret updates when it is rotated.

```yaml
   volumeMounts:
   - mountPath: /app/settings.ini
     name: app-config
     subPath: settings.ini
...
 volumes:
 - name: app-config
   csi:
     driver: secrets-store.csi.deckhouse.io
     volumeAttributes:
       secretsStoreImport: "python-backend"
```
