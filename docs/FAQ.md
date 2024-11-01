---
title: "The secrets-store-integration module: FAQ"
description: Hashicorp Vault configuration examples. Example of secret autorotation implementation.
---

## How to set up the Hashicorp Vault as a secret store to use with the secrets-store-integration module:

{{< alert level="info">}}
First of all, you'll need a root or similiar token and the vault address.
You can get such a root token while initializing a new secrets store.


```bash
export VAULT_TOKEN=xxxxxxxxxxx
export VAULT_ADDR=https://secretstoreexample.com
```
{{< /alert >}}

> This guide will cover two ways to do this:
>   * using the console version of HashiCorp Vault (see the [Vault installation guide](https://developer.hashicorp.com/vault/docs/install));
>   * using curl to make direct requests to the secrets store API.

This section provides an example of the settings that need to be made so that the service pod can access the secret located in the Key-Value storage. The secret will be the password for the database that the Python application uses.

* Enable and create the Key-Value store:

  ```bash
  stronghold secrets enable -path=secret -version=2 kv
  ```

  The same command as a curl HTTP request:

  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request POST \
    --data '{"type":"kv","options":{"version":"2"}}' \
    ${VAULT_ADDR}/v1/sys/mounts/secret
  ```

* Set the database password as the secret value:

  ```bash
  stronghold kv put secret/database-for-python-app password="db-secret-password"
  ```

  The curl equivalent of the above command:

  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"data":{"password":"db-secret-password"}}' \
    ${VAULT_ADDR}/v1/secret/data/database-for-python-app
  ```

* Double-check that the password has been saved successfully:

  ```bash
  stronghold kv get secret/database-for-python-app
  ```

  The curl equivalent of the above command:

  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    ${VAULT_ADDR}/v1/secret/data/database-for-python-app
  ```

* Set the authentication path (`authPath`) and enable authentication and authorization in Vault using the Kubernetes API:

  ```bash
  stronghold auth enable -path=main-kube kubernetes
  ```

  The curl equivalent of the above command:

  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request POST \
    --data '{"type":"kubernetes"}' \
    ${VAULT_ADDR}/v1/sys/auth/main-kube
  ```

* If you have more than one cluster, set the authentication path (`authPath`) and enable authentication and authorization in Vault using the Kubernetes API of the second cluster:

  ```bash
  stronghold auth enable -path=secondary-kube kubernetes
  ```

  The curl equivalent of the above command:

  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request POST \
    --data '{"type":"kubernetes"}' \
    ${VAULT_ADDR}/v1/sys/auth/secondary-kube
  ```

* Set the Kubernetes API address for each cluster (in this case, it is the K8s's API server service):

  ```bash
  stronghold write auth/main-kube/config \
    kubernetes_host="https://api.kube.my-deckhouse.com"
  ```

  The curl equivalent of the above command:

  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"kubernetes_host":"https://api.kube.my-deckhouse.com"}' \
    ${VAULT_ADDR}/v1/auth/main-kube/config
  ```
  For another cluster:

  ```bash
  stronghold write auth/secondary-kube/config \
    kubernetes_host="https://10.11.12.10:443"
  ```

  The curl equivalent of the above command:

  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"kubernetes_host":"https://10.11.12.10:443"}' \
    ${VAULT_ADDR}/v1/auth/secondary-kube/config
  ```

* Create a policy in Vault called "backend" that allows reading of the `database-for-python-app` secret:

  ```bash
  stronghold policy write backend - <<EOF
  path "secret/data/database-for-python-app" {
  capabilities = ["read"]
  }
  EOF
  ```

  The curl equivalent of the above command:

  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"policy":"path \"secret/data/database-for-python-app\" {\n capabilities = [\"read\"]\n}\n"}' \
    ${VAULT_ADDR}/v1/sys/policies/acl/backend
  ```

* Create a database role and bind it to the `backend-sa` ServiceAccount in the `my-namespace1` namespace and the `backend` policy:

  {{< alert level="danger">}}
  **Important!**  
  In addition to the Vault side settings, you must configure the authorization permissions of the `serviceAccount` used in the kubernetes cluster.
  See the [FAQ](faq.html#how-to-allow-serviceaccount-to-authorize-in-vault) section for details.
  {{< /alert >}}

  ```bash
  stronghold write auth/main-kube/role/my-namespace1_backend \
    bound_service_account_names=backend-sa \
    bound_service_account_namespaces=my-namespace1 \
    policies=backend \
    ttl=10m
  ```

  The curl equivalent of the above command:

  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"bound_service_account_names":"backend-sa","bound_service_account_namespaces":"my-namespace1","policies":"backend","ttl":"10m"}' \
    ${VAULT_ADDR}/v1/auth/main-kube/role/my-namespace1_backend
  ```

  Do the same for the second K8s cluster:

  ```bash
  stronghold write auth/secondary-kube/role/my-namespace1_backend \
    bound_service_account_names=backend-sa \
    bound_service_account_namespaces=my-namespace1 \
    policies=backend \
    ttl=10m
    ```

  The curl equivalent of the above command:

  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"bound_service_account_names":"backend-sa","bound_service_account_namespaces":"my-namespace1","policies":"backend","ttl":"10m"}' \
    ${VAULT_ADDR}/v1/auth/secondary-kube/role/my-namespace1_backend
  ```
  {{< alert level="info">}}
  **Important!**  
  The recommended TTL value of the Kubernetes token is 10m.
  {{< /alert >}}
  
These settings allow any pod within the `my-namespace1` namespace in both K8s clusters that uses the `backend-sa` ServiceAccount to authenticate, authorize, and read secrets in the Vault according to the `backend` policy. 

## How to allow a ServiceAccount to log in to Vault?

To log in to Vault, a k8s pod uses a token generated for its ServiceAccount. In order for Vault to be able to check the validity of the ServiceAccount data provided by the service, Vault must have permission to `get`, `list`, and `watch` for the `tokenreviews.authentication.k8s.io` and `subjectaccessreviews.authorization.k8s.io` endpoints. You can also use the `system:auth-delegator` clusterRole for this.

Vault can use different credentials to make requests to the Kubernetes API:
1. Use the token of the application that is trying to log in to Vault. In this case, each service that logs in to Vault must have the `system:auth-delegator` clusterRole (or the API rights listed above) in the ServiceAccount it uses.
2. Use a static token created specifically for Vault `ServiceAccount` that has the necessary rights. Setting up Vault for this case is described in detail in [Vault documentation](https://developer.hashicorp.com/vault/docs/auth/kubernetes#continue-using-long-lived-tokens).

## How to autorotate secrets mounted as files in containers without restarting them?

The autorotation feature of the secret-store-integration module is enabled by default. Every two minutes, the module polls Vault and synchronizes the secrets in the mounted file if it has been changed.

Create the ```backend-sa``` ServiceAccount 

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: backend-sa
  namespace: my-namespace1
```

Below is an example of the SecretStoreImport definition:

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

In the `backend` example below, the SecretStoreImport (defined above) is mounted as a volume to push the database password to the application:

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

Once these resources have been applied, a `backend` pod will be started. In it, there will be a `/mnt/secrets` directory with the `secrets` volume mounted. The directory will contain a `db-password` file with the password for the Vault database.

There are two ways to keep track of changes to the secret file in the pod. The first is to keep track of when the mounted file changes (mtime), reacting to changes in the file. The second is to use the inotify API, which provides a mechanism for subscribing to file system events. Inotify is part of the Linux kernel. Once a change is detected, there are a large number of options for responding to the change event, depending on the application architecture and programming language used. The most simple one is to force K8s to restart the pod by failing the liveness probe.

Here is how you can use inotify in a Python application leveraging the `inotify` Python package:

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

Sample code to detect whether a password has been changed within a Go application using inotify and the `inotify` Go package:

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

A container that uses the `subPath` volume mount will not get secret updates when the latter is rotated.

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
