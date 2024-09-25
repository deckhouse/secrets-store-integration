---
title: "The secrets-store-integration module: usage"
description: Usage of the secrets-store-integration Deckhouse module.
---

## Configuring the module to work with Deckhouse Stronghold

[Enable](../../stronghold/stable/usage.html#how-to-enable-the-module) the Stronghold module beforehand to automatically configure the secrets-store-integration module to work with [Deckhouse Stronghold](../../stronghold/).

Next, apply the `ModuleConfig`:

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: ModuleConfig
metadata:
  name: secrets-store-integration
spec:
  enabled: true
  version: 1
```

The [connectionConfiguration](../../secrets-store-integration/stable/configuration.html#parameters-connectionconfiguration) paramater is optional and set to `DiscoverLocalStronghold` value by default.

## Configuring the module to work with the external secret store

The module requires a pre-configured secret vault compatible with HashiCorp Vault. An authentication path must be preconfigured in the vault. An example of how to configure the secret vault is provided in [Setting up the test environment](#setting-up-the-test-environment).

To ensure that each API request is encrypted, sent to, and replied by the correct recipient, a valid public Certificate Authority certificate used by the secret store is required. A `caCert` variable in the module configuration must refer to such a CA certificate in PEM format.

The following is an example module configuration for using a Vault-compliant secret store running at "secretstoreexample.com" on a regular port (443 TLS). Note that you will need to replace the parameters values in the configuration with the values that match your environment.

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: ModuleConfig
metadata:
  name: secrets-store-integration
spec:
  version: 1
  enabled: true
  settings:
    connection:
      url: "https://secretstoreexample.com"
      authPath: "main-kube"
      caCert: |
        -----BEGIN CERTIFICATE-----
        MIIFoTCCA4mgAwIBAgIUX9kFz7OxlBlALMEj8WsegZloXTowDQYJKoZIhvcNAQEL
        ................................................................
        WoR9b11eYfyrnKCYoSqBoi2dwkCkV1a0GN9vStwiBnKnAmV3B8B5yMnSjmp+42gt
        o2SYzqM=
        -----END CERTIFICATE-----
```

**It is strongly recommended to set the `caCert` variable. Otherwise, the module will use system ca-certificates.**

## Setting up the test environment

{{< alert level="info">}}
First of all, you'll need a root or similiar token and the Stronghold address.
You can get such a root token while initializing a new secrets store.

All subsequent commands will assume that these settings are specified in environment variables.
```bash
export VAULT_TOKEN=xxxxxxxxxxx
export VAULT_ADDR=https://secretstoreexample.com
```
{{< /alert >}}

> This guide will cover two ways to do this:
>   * using the console version of Stronghold (see the [Vault installation guide](https://developer.hashicorp.com/vault/docs/install));
>   * using curl to make direct requests to the secrets store API.

Before proceeding with the secret injection instructions in the examples below, do the following:

1. Create a kv2 type secret in Stronghold in `test-kv/myapp` and copy `DB_USER` and `DB_PASS` there.
2. If necessary, add an authentication path (authPath) for authentication and authorization to Stronghold using the Kubernetes API of the remote cluster
3. Create a policy in Stronghold that allows reading secrets from `secret/myapp`.
4. Create a `myapp` role in Stronghold for the `myapp` service account in the `my-namespace` namespace and bind the policy you created earlier to it.
5. Create a `my-namespace` namespace in the cluster.
6. Create a `myapp` service account in the created namespace.

Example commands to set up the environment:

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

* Set the database username and password as the value of the secret:

  ```bash
  stronghold kv put secret/myapp DB_USER="username" DB_PASS="secret-password"
  ```
  The curl equivalent of the above command:

  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"data":{"DB_USER":"username","DB_PASS":"secret-password"}}' \
    ${VAULT_ADDR}/v1/secret/data/myapp
  ```

* Double-check that the password has been saved successfully:

  ```bash
  stronghold kv get secret/myapp
  ```  
  
  The curl equivalent of the above command:
  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    ${VAULT_ADDR}/v1/secret/data/myapp
  ```

* By default, the method of authentication in Stronghold via Kubernetes API of the cluster on which Stronghold itself is running is enabled and configured under the name `kubernetes_local`. If you want to configure access via remote clusters, set the authentication path (`authPath`) and enable authentication and authorization in Stronghold via Kubernetes API for each cluster:

  ```bash
  stronghold auth enable -path=remote-kube-1 kubernetes
  ```
  The curl equivalent of the above command:
  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request POST \
    --data '{"type":"kubernetes"}' \
    ${VAULT_ADDR}/v1/sys/auth/remote-kube-1
  ```

* Set the Kubernetes API address for each cluster (in this case, it is the K8s's API server service):

  ```bash
  stronghold write auth/remote-kube-1/config \
    kubernetes_host="https://api.kube.my-deckhouse.com"
  ```
  The curl equivalent of the above command:
  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"kubernetes_host":"https://api.kube.my-deckhouse.com"}' \
    ${VAULT_ADDR}/v1/auth/remote-kube-1/config
  ```

* Create a policy in Vault called `backend` that allows reading of the `myapp` secret:

  ```bash
  stronghold policy write backend - <<EOF
  path "secret/data/myapp" {
    capabilities = ["read"]
  }
  EOF
  ```
  The curl equivalent of the above command:
  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"policy":"path \"secret/data/myapp\" {\n capabilities = [\"read\"]\n}\n"}' \
    ${VAULT_ADDR}/v1/sys/policies/acl/backend
  ```


* Create a database role and bind it to the `myapp` ServiceAccount in the `my-namespace` namespace and the `backend` policy:

  {{< alert level="danger">}}
  **Important!**  
  In addition to the Vault side settings, you must configure the authorization permissions of the `serviceAccount` used in the kubernetes cluster.
  See the [paragraph below](#how-to-allow-a-serviceaccount-to-log-in-to-vault) section for details.
  {{< /alert >}}

  ```bash
  stronghold write auth/kubernetes_local/role/my-namespace_backend \
      bound_service_account_names=myapp \
      bound_service_account_namespaces=my-namespace \
      policies=backend \
      ttl=10m
  ```
  The curl equivalent of the above command:
  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"bound_service_account_names":"myapp","bound_service_account_namespaces":"my-namespace","policies":"backend","ttl":"10m"}' \
    ${VAULT_ADDR}/v1/auth/kubernetes_local/role/my-namespace_backend
  ```


* Repeat the same for the rest of the clusters, specifying a different authentication path:

  ```bash
  stronghold write auth/remote-kube-1/role/my-namespace_backend \
      bound_service_account_names=myapp \
      bound_service_account_namespaces=my-namespace \
      policies=backend \
      ttl=10m
  ```
  The curl equivalent of the above command:
  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"bound_service_account_names":"myapp","bound_service_account_namespaces":"my-namespace","policies":"backend","ttl":"10m"}' \
    ${VAULT_ADDR}/v1/auth/remote-kube-1/role/my-namespace_backend
  ```


  {{< alert level="info">}}
  **Important!**  
  The recommended TTL value of the Kubernetes token is 10m.
  {{< /alert >}}

These settings allow any pod within the `my-namespace` namespace in both K8s clusters that uses the `myapp` ServiceAccount to authenticate, authorize, and read secrets in the Vault according to the `backend` policy.

* Create namespace and then ServiceAccount in the specified namespace:
  ```bash
  kubectl create namespace my-namespace
  kubectl -n my-namespace create serviceaccount myapp
  ```

## How to allow a ServiceAccount to log in to Vault?

To log in to Vault, a k8s pod uses a token generated for its ServiceAccount. In order for Vault to be able to check the validity of the ServiceAccount data provided by the service, Vault must have permission to `get`, `list`, and `watch` for the `tokenreviews.authentication.k8s.io` and `subjectaccessreviews.authorization.k8s.io` endpoints. You can also use the `system:auth-delegator` clusterRole for this.

Vault can use different credentials to make requests to the Kubernetes API:
1. Use the token of the application that is trying to log in to Vault. In this case, each service that logs in to Vault must have the `system:auth-delegator` clusterRole (or the API rights listed above) in the ServiceAccount it uses.
2. Use a static token created specifically for Vault `ServiceAccount` that has the necessary rights. Setting up Vault for this case is described in detail in [Vault documentation](https://developer.hashicorp.com/vault/docs/auth/kubernetes#continue-using-long-lived-tokens).

## Injecting environment variables

### How it works

When the module is enabled, a mutating-webhook becomes available in the cluster. It modifies the pod manifest, adding an injector, if the pod has the `secrets-store.deckhouse.io/role` annotation An init container is added to the modified pod. Its mission is to copy a statically compiled binary injector file from a service image into a temporary directory shared by all containers in the pod. In the other containers, the original startup commands are replaced with a command that starts the injector. It then fetches the required data from a Vault-compatible storage using the application's service account, sets these variables in the process ENV, and then issues an execve system call, invoking the original command.

If the container does not have a startup command in the pod manifest, the image manifest is retrieved from the image registry,
and the command is retrieved from it.
The credentials from `imagePullSecrets` specified in the pod manifest are used to retrieve the manifest from the private image registry.


The following are the available annotations to modify the injector behavior:
| Annotation                                       | Default value |  Function |
|--------------------------------------------------|-----------|-------------|
|secrets-store.deckhouse.io/role                   |           | Sets the role to be used to connect to the secret store |
|secrets-store.deckhouse.io/env-from-path          |           | Specifies the path to the secret in the vault to retrieve all keys from and add them to the environment |
|secrets-store.deckhouse.io/ignore-missing-secrets | false     | Runs the original application if an attempt to retrieve a secret from the store fails |
|secrets-store.deckhouse.io/client-timeout         | 10s       | Timeout to use for secrets retrieval |
|secrets-store.deckhouse.io/mutate-probes          | false     | Injects environment variables into the probes |
|secrets-store.deckhouse.io/log-level              | info      | Logging level |
|secrets-store.deckhouse.io/enable-json-log        | false     | Log format (string or JSON) |

The injector allows you to specify env templates instead of values in the pod manifests. They will be replaced at the container startup stage with the values from the store.

For example, here's how you can retrieve the `mypassword` key from the kv2-secret at `secret/myapp` from the Vault-compatible store:

```yaml
env:
  - name: PASSWORD
    value: secrets-store:secret/data/myapp#mypassword
```

The example below retrieves the `mypassword` key version `4` from the kv2 secret at `secret/myapp` from the Vault-compatible store:

```yaml
env:
  - name: PASSWORD
    value: secrets-store:secret/data/myapp#mypassword#4
```

The template can also be stored in the ConfigMap or in the Secret and can be hooked up using `envFrom`:

```yaml
envFrom:
  - secretRef:
      name: app-secret-env
  - configMapRef:
      name: app-env

```
The actual secrets from the Vault-compatible store will be injected at the application startup; the Secret and ConfigMap will only contain the templates.

### Setting environment variables by specifying the path to the secret in the vault to retrieve all keys from

The following is the specification of a pod named `myapp1`. In it, all the values are retrieved from the store at the `secret/data/myapp` path and stored as environment variables:

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: myapp1
  namespace: my-namespace
  annotations:
    secrets-store.deckhouse.io/role: "myapp"
    secrets-store.deckhouse.io/env-from-path: secret/data/myapp
spec:
  serviceAccountName: myapp
  containers:
  - image: alpine:3.20
    name: myapp
    command:
    - sh
    - -c
    - while printenv; do sleep 5; done
```

Let's apply it:

```bash
kubectl create --filename myapp1.yaml
```

Check the pod logs after it has been successfully started. You should see all the values from `secret/data/myapp`:

```bash
kubectl -n my-namespace logs myapp1
```

Delete the pod:

```bash
kubectl -n my-namespace delete pod myapp1 --force
```

### Explicitly specifying the values to be retrieved from the vault and used as environment variables

Below is the spec of a test pod named `myapp2`. The pod will retrieve the required values from the vault according to the template and turn them into environment variables:

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: myapp2
  namespace: my-namespace
  annotations:
    secrets-store.deckhouse.io/role: "myapp"
spec:
  serviceAccountName: myapp
  containers:
  - image: alpine:3.20
    env:
    - name: DB_USER
      value: secrets-store:secret/data/myapp#DB_USER
    - name: DB_PASS
      value: secrets-store:secret/data/myapp#DB_PASS
    name: myapp
    command:
    - sh
    - -c
    - while printenv; do sleep 5; done
```

Apply it:

```bash
kubectl create --filename myapp2.yaml
```

Check the pod logs after it has been successfully started. You should see the values from `secret/data/myapp` matching those in the pod specification:

```bash
kubectl -n my-namespace logs myapp2
```

Delete the pod:

```bash
kubectl -n my-namespace delete pod myapp2 --force
```

## Retrieving a secret from the vault and mounting it as a file in a container

Use the `SecretStoreImport` CustomResource to deliver secrets to the application.

In this example, we use the already created ServiceAccount `myapp` and namespace `my-namespace` from step [Setting up the test environment](#setting-up-the-test-environment)

Create a _SecretsStoreImport_ CustomResource named `myapp-ssi` in the cluster:

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: SecretsStoreImport
metadata:
  name: myapp-ssi
  namespace: my-namespace
spec:
  type: CSI
  role: myapp
  files:
    - name: "db-password"
      source:
        path: "secret/data/myapp"
        key: "DB_PASS"
```

Create a test pod in the cluster named `myapp3`. It will retrieve the required values from the vault and mount them as a file:

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: myapp3
  namespace: my-namespace
spec:
  serviceAccountName: myapp
  containers:
  - image: alpine:3.20
    name: myapp
    command:
    - sh
    - -c
    - while cat /mnt/secrets/db-password; do echo; sleep 5; done
    name: backend
    volumeMounts:
    - name: secrets
      mountPath: "/mnt/secrets"
  volumes:
  - name: secrets
    csi:
      driver: secrets-store.csi.deckhouse.io
      volumeAttributes:
        secretsStoreImport: "myapp"
```

Once these resources have been applied, a `backend` pod will be started. In it, there will be a `/mnt/secrets` directory with the `secrets` volume mounted. The directory will contain a `db-password` file with the password for the Vault database.

Check the pod logs after it has been successfully started (you should see the contents of the `/mnt/secrets/db-password` file):

```bash
kubectl -n my-namespace logs myapp3
```

Delete the pod:

```bash
kubectl -n my-namespace delete pod myapp3 --force
```

### The autorotation feature

The autorotation feature of the secret-store-integration module is enabled by default. Every two minutes, the module polls Vault and synchronizes the secrets in the mounted file if it has been changed.

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

#### Secret rotation limitations

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