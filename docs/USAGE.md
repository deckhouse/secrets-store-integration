---
title: "The secrets-store-integration module: examples"
description: Usage examples for the secrets-store-integration module.
---

This section contains usage examples for the `secrets-store-integration` module.

## CLI tool `d8` for Stronghold commands

Deckhouse CLI (`d8`) is a universal tool required to run commands such as `d8 stronghold` in the terminal.

To install `d8`, use one of the methods described in the [CLI tool documentation](/products/kubernetes-platform/documentation/v1/cli/d8/#installing-the-executable).

## Configuring the module to work with Deckhouse Stronghold

1. Enable the `stronghold` module by following the [instructions](/modules/stronghold/usage.html#how-to-enable-the-module).
1. To enable the `secrets-store-integration` module, apply the following resource:

   ```yaml
   apiVersion: deckhouse.io/v1alpha1
   kind: ModuleConfig
   metadata:
     name: secrets-store-integration
   spec:
     enabled: true
     version: 1
   ```

   You do not have to set the [`connectionConfiguration`](configuration.html#parameters-connectionconfiguration) parameter, because `DiscoverLocalStronghold` is used by default.

## Configuring the module to work with an external store

The module requires a preconfigured secret store compatible with HashiCorp Vault. An authentication path must already be configured in the store. An example of store configuration is shown [below](#preparing-a-test-environment).

To ensure that each API request is encrypted, sent, and processed by the correct recipient, you need a valid public Certificate Authority certificate used by the secret store. This CA public certificate in PEM format must be used as the `caCert` variable in the module configuration.

Example module configuration for using a Vault-compatible secret store running at `secretstoreexample.com` on the default TLS port (`443`):

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
    connectionConfiguration: Manual
```

{{< alert level="info">}}
Setting `caCert` is recommended. If it is not set, the module uses the system `ca-certificates` bundle.
{{< /alert >}}

## Preparing a test environment

{{< alert level="info">}}
To run the commands below, you need the Stronghold address and a token with `root` privileges.

You can get such a token when initializing a new secret store.

The examples below assume these settings are defined in environment variables:

```bash
export VAULT_TOKEN=xxxxxxxxxxx
export VAULT_ADDR=https://secretstoreexample.com
```

{{< /alert >}}

{{< alert level="info">}}
This section contains two variants of example commands:

- Commands using the [`d8` CLI tool](#cli-tool-d8-for-stronghold-commands)
- Commands using `curl` to make direct requests to the secret store API
{{< /alert >}}

Before injecting secrets, prepare a test environment.

1. Create a `kv2` secret in Stronghold at `demo-kv/myapp-secret` and put the `DB_USER` and `DB_PASS` values there.

   * Enable and create the Key-Value store:

     ```bash
     d8 stronghold secrets enable -path=demo-kv -version=2 kv
     ```

     Alternative using `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request POST \
       --data '{"type":"kv","options":{"version":"2"}}' \
       ${VAULT_ADDR}/v1/sys/mounts/demo-kv
     ```

   * Set the database username and password as the secret value:

     ```bash
     d8 stronghold kv put demo-kv/myapp-secret DB_USER="username" DB_PASS="secret-password"
     ```

     Alternative using `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request PUT \
       --data '{"data":{"DB_USER":"username","DB_PASS":"secret-password"}}' \
       ${VAULT_ADDR}/v1/demo-kv/data/myapp-secret
     ```

   * Verify the stored secret:

     ```bash
     d8 stronghold kv get demo-kv/myapp-secret
     ```

     Alternative verification using `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       ${VAULT_ADDR}/v1/demo-kv/data/myapp-secret
     ```

1. If necessary, add an authentication path ([`authPath`](configuration.html#parameters-connection-authpath)) for authentication and authorization in Stronghold using the Kubernetes API of a remote cluster.

   * By default, Stronghold enables and configures the Kubernetes authentication method under the name `kubernetes_local` for the cluster where Stronghold itself is running. If you need to configure access through remote clusters, set the authentication path (`authPath`) and enable authentication and authorization in Stronghold through the Kubernetes API for each cluster:

     ```bash
     d8 stronghold auth enable -path=remote-kube-1 kubernetes
     ```

     Alternative using `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request POST \
       --data '{"type":"kubernetes"}' \
       ${VAULT_ADDR}/v1/sys/auth/remote-kube-1
     ```

   * Set the Kubernetes API address for each cluster:

     ```bash
     d8 stronghold write auth/remote-kube-1/config \
       kubernetes_host="https://api.kube.my-deckhouse.com"
     ```

     Alternative using `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request PUT \
       --data '{"kubernetes_host":"https://api.kube.my-deckhouse.com"}' \
       ${VAULT_ADDR}/v1/auth/remote-kube-1/config
     ```

1. Create a `myapp-ro-policy` policy in Stronghold that allows reading secrets from `demo-kv/data/myapp-secret`:

   ```bash
   d8 stronghold policy write myapp-ro-policy - <<EOF
   path "demo-kv/data/myapp-secret" {
     capabilities = ["read"]
   }
   EOF
   ```

   Alternative using `curl`:

   ```bash
   curl \
     --header "X-Vault-Token: ${VAULT_TOKEN}" \
     --request PUT \
     --data '{"policy":"path \"demo-kv/data/myapp-secret\" {\n capabilities = [\"read\"]\n}\n"}' \
     ${VAULT_ADDR}/v1/sys/policies/acl/myapp-ro-policy
   ```

1. Create a role in Stronghold for the `myapp-sa` service account in the `myapp-namespace` namespace and bind the policy created earlier to it.

   {{< alert level="danger">}}
   In addition to the Stronghold-side configuration, you must configure authorization permissions for the ServiceAccount objects used in the Kubernetes cluster.

   See the required settings in the [next section](#how-to-allow-a-serviceaccount-to-authenticate-in-stronghold).
   {{< /alert >}}

   * Create a role consisting of the namespace and policy name. Bind it to the `myapp-sa` ServiceAccount in the `myapp-namespace` namespace and to the `myapp-ro-policy` policy:

     {{< alert level="info">}}
     The recommended TTL value for the Kubernetes token is `10m`.
     {{< /alert >}}

     ```bash
     d8 stronghold write auth/kubernetes_local/role/myapp-role \
         bound_service_account_names=myapp-sa \
         bound_service_account_namespaces=myapp-namespace \
         policies=myapp-ro-policy \
         ttl=10m
     ```

     Alternative using `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request PUT \
       --data '{"bound_service_account_names":"myapp-sa","bound_service_account_namespaces":"myapp-namespace","policies":"myapp-ro-policy","ttl":"10m"}' \
       ${VAULT_ADDR}/v1/auth/kubernetes_local/role/myapp-role
     ```

   * Repeat the same for remote clusters, specifying a different authentication path:

     ```bash
     d8 stronghold write auth/remote-kube-1/role/myapp-role \
         bound_service_account_names=myapp-sa \
         bound_service_account_namespaces=myapp-namespace \
         policies=myapp-ro-policy \
         ttl=10m
     ```

     Alternative using `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request PUT \
       --data '{"bound_service_account_names":"myapp-sa","bound_service_account_namespaces":"myapp-namespace","policies":"myapp-ro-policy","ttl":"10m"}' \
       ${VAULT_ADDR}/v1/auth/remote-kube-1/role/myapp-role
     ```

   These settings allow any pod in the `myapp-namespace` namespace in both Kubernetes clusters that uses the `myapp-sa` ServiceAccount to authenticate and authorize in Stronghold to read secrets according to the `myapp-ro-policy` policy.

1. Create the `myapp-namespace` namespace in the cluster:

   ```bash
   d8 k create namespace myapp-namespace
   ```

1. Create the `myapp-sa` service account in that namespace:

   ```bash
   d8 k -n myapp-namespace create serviceaccount myapp-sa
   ```

## How to allow a ServiceAccount to authenticate in Stronghold

To authenticate in Stronghold, a pod uses the token generated for its ServiceAccount. For Stronghold to validate the provided ServiceAccount data, the Stronghold service must have `get`, `list`, and `watch` permissions for the `tokenreviews.authentication.k8s.io` and `subjectaccessreviews.authorization.k8s.io` endpoints. You can also use the `system:auth-delegator` ClusterRole for this.

Stronghold can use different credentials to send requests to the Kubernetes API:

- A token of the application that is trying to authenticate in Stronghold. In this case, every service authenticating in Stronghold requires the `system:auth-delegator` ClusterRole or the API permissions listed above on the ServiceAccount it uses. See the example in the [Stronghold documentation](https://deckhouse.io/products/stronghold/documentation/user/auth/kubernetes.html#use-the-stronghold-clients-jwt-as-the-reviewer-jwt).
- A static token of a ServiceAccount created specifically for Stronghold and granted the necessary permissions. Configuring Stronghold for this case is described in detail in the [Stronghold documentation](https://deckhouse.io/products/stronghold/documentation/user/auth/kubernetes.html#continue-using-long-lived-tokens).

## Injecting environment variables

### How injection works

When the module is enabled, a `mutating-webhook` appears in the cluster. If a pod has the `secrets-store.deckhouse.io/role` annotation, the webhook modifies the pod manifest by adding the injector.

In the modified pod:

1. An init container is added.
1. The init container copies a statically linked injector binary from the service image into a temporary directory shared by all containers in the pod.
1. In the remaining containers, the original startup commands are replaced with a command that launches the injector binary.
1. The injector retrieves the required data from a Vault-compatible store using the application's service account.
1. It puts these variables into the process `ENV`.
1. It performs the `execve` system call and starts the original command.

If a container does not define a startup command in the pod manifest, the image manifest is fetched from the registry and the command is taken from it.

Credentials from `imagePullSecrets` specified in the pod manifest are used to retrieve the manifest from a private image registry.

### Injector annotations

The following annotations are available to modify injector behavior:

<style>.annotations-table-style + .table-wrapper td:first-child{min-width: 317px}</style>
<div class="annotations-table-style"></div>

| Annotation | Default value | Description |
| --- | --- | --- |
| `secrets-store.deckhouse.io/addr` | From module | Secret store address in the format `https://stronghold.mycompany.tld:8200` |
| `secrets-store.deckhouse.io/tls-secret` | From module | Name of the Secret object in Kubernetes containing the `ca.crt` key with the CA certificate value in PEM format |
| `secrets-store.deckhouse.io/tls-skip-verify` | `false` | Disables verification of the server TLS certificate |
| `secrets-store.deckhouse.io/auth-path` | From module | Path to use for authentication |
| `secrets-store.deckhouse.io/namespace` | From module | Namespace that will be used to connect to the store |
| `secrets-store.deckhouse.io/role` | | Role used to connect to the secret store |
| `secrets-store.deckhouse.io/env-from-path` | | Comma-separated list of secret paths in the store from which all keys will be extracted and placed into the environment. Keys from paths closer to the end of the list take precedence |
| `secrets-store.deckhouse.io/ignore-missing-secrets` | `false` | Starts the original application if retrieving a secret from the store fails |
| `secrets-store.deckhouse.io/client-timeout` | `10s` | Timeout for secret retrieval |
| `secrets-store.deckhouse.io/mutate-probes` | `false` | Injects environment variables into probes |
| `secrets-store.deckhouse.io/log-level` | `info` | Logging level |
| `secrets-store.deckhouse.io/enable-json-log` | `false` | Enables JSON log output |
| `secrets-store.deckhouse.io/skip-mutate-containers` | | Space-separated list of container names that will not be mutated |

Using the injector, you can specify templates in pod manifests instead of actual `env` values. They are replaced with values from the store at container startup time.

{{< alert level="info">}}
Importing variables from a store path has higher priority than explicitly defined variables from the store. This means that if you use the `secrets-store.deckhouse.io/env-from-path` annotation with a path to a secret containing, for example, the `MY_SECRET` key, and also define an environment variable with the same name in the manifest:

```yaml
env:
  - name: MY_SECRET
    value: secrets-store:demo-kv/data/myapp-secret#password
```

the `MY_SECRET` environment variable inside the container will be set to the secret value from the **annotation**.
{{< /alert >}}

Example of retrieving the `DB_PASS` key from a `kv2` secret at `demo-kv/myapp-secret` from a Vault-compatible store:

```yaml
env:
  - name: PASSWORD
    value: secrets-store:demo-kv/data/myapp-secret#DB_PASS
```

Example of retrieving version `4` of the `DB_PASS` key from a `kv2` secret at `demo-kv/myapp-secret`:

```yaml
env:
  - name: PASSWORD
    value: secrets-store:demo-kv/data/myapp-secret#DB_PASS#4
```

The template can also be stored in a ConfigMap or a Secret and connected using `envFrom`:

```yaml
envFrom:
  - secretRef:
      name: app-secret-env
  - configMapRef:
      name: app-env
```

Actual secrets from the Vault-compatible store are injected only at application startup. The Secret and ConfigMap objects contain templates.

### Importing variables from a store path

In this scenario, all keys from a single secret are imported.

1. Create a pod named `myapp1` that imports all variables from the store at `demo-kv/data/myapp-secret`:

   ```yaml
   kind: Pod
   apiVersion: v1
   metadata:
     name: myapp1
     namespace: myapp-namespace
     annotations:
       secrets-store.deckhouse.io/role: "myapp-role"
       secrets-store.deckhouse.io/env-from-path: demo-kv/data/common-secret,demo-kv/data/myapp-secret
   spec:
     serviceAccountName: myapp-sa
     containers:
     - image: alpine:3.20
       name: myapp
       command:
       - sh
       - -c
       - while printenv; do sleep 5; done
   ```

1. Apply the manifest:

   ```bash
   d8 k create --filename myapp1.yaml
   ```

1. Check the pod logs after startup. The output should include all variables from `demo-kv/data/myapp-secret`:

   ```bash
   d8 k -n myapp-namespace logs myapp1
   ```

1. Delete the pod:

   ```bash
   d8 k -n myapp-namespace delete pod myapp1 --force
   ```

### Importing explicitly defined variables from the store

1. Create a test pod named `myapp2` that imports the required variables from the store using templates:

   ```yaml
   kind: Pod
   apiVersion: v1
   metadata:
     name: myapp2
     namespace: myapp-namespace
     annotations:
       secrets-store.deckhouse.io/role: "myapp-role"
   spec:
     serviceAccountName: myapp-sa
     containers:
     - image: alpine:3.20
       env:
       - name: DB_USER
         value: secrets-store:demo-kv/data/myapp-secret#DB_USER
       - name: DB_PASS
         value: secrets-store:demo-kv/data/myapp-secret#DB_PASS
       name: myapp
       command:
       - sh
       - -c
       - while printenv; do sleep 5; done
   ```

1. Apply the configuration:

   ```bash
   d8 k create --filename myapp2.yaml
   ```

1. Check the pod logs after startup. The output should include variables from `demo-kv/data/myapp-secret`:

   ```bash
   d8 k -n myapp-namespace logs myapp2
   ```

1. Delete the pod:

   ```bash
   d8 k -n myapp-namespace delete pod myapp2 --force
   ```

## Mounting a secret from the store as a file into a container

Use the [SecretsStoreImport](cr.html#secretsstoreimport) custom resource to deliver secrets to the application.

This example uses the `myapp-sa` service account and the `myapp-namespace` namespace created during [test environment preparation](#preparing-a-test-environment).

1. Create a SecretsStoreImport custom resource named `myapp-ssi` in the cluster:

   ```yaml
   apiVersion: deckhouse.io/v1alpha1
   kind: SecretsStoreImport
   metadata:
     name: myapp-ssi
     namespace: myapp-namespace
   spec:
     type: CSI
     role: myapp-role
     files:
       - name: "db-password"
         source:
           path: "demo-kv/data/myapp-secret"
           key: "DB_PASS"
   ```

1. Create a test pod named `myapp3` in the cluster that mounts the secret from the store as a file:

   ```yaml
   kind: Pod
   apiVersion: v1
   metadata:
     name: myapp3
     namespace: myapp-namespace
   spec:
     serviceAccountName: myapp-sa
     containers:
     - image: alpine:3.20
       name: backend
       command:
       - sh
       - -c
       - while cat /mnt/secrets/db-password; do echo; sleep 5; done
       volumeMounts:
       - name: secrets
         mountPath: "/mnt/secrets"
     volumes:
     - name: secrets
       csi:
         driver: secrets-store.csi.deckhouse.io
         volumeAttributes:
           secretsStoreImport: "myapp-ssi"
   ```

   After these resources are applied, a pod is created with a `backend` container. Inside the container filesystem, the `/mnt/secrets` directory contains the mounted `secrets` volume. This directory contains the `db-password` file with the database password (`DB_PASS`) from the Stronghold Key-Value store.

1. Check the pod logs after startup. The output should contain the contents of `/mnt/secrets/db-password`:

   ```bash
   d8 k -n myapp-namespace logs myapp3
   ```

1. Delete the pod:

   ```bash
   d8 k -n myapp-namespace delete pod myapp3 --force
   ```

### Delivering binary files into a container

In some cases, you may need to deliver a binary file into a container, for example:

- A JKS keystore
- A `keytab` for Kerberos authentication

In this case, you can encode the binary file as Base64 and place it into the secret store. When retrieved, the CSI driver decodes the data and places the binary file into the container. To do this, set `decodeBase64` to `true` for the corresponding file.

If decoding fails, for example because the store contains invalid Base64 data, the container will not be created.

Example:

1. Encode the file as Base64 and place it into the store:

   ```bash
   d8 stronghold kv put demo-kv/myapp-secret keytab=$(cat /path/to/keytab_file | base64 -w0)
   ```

1. Create a [SecretsStoreImport](cr.html#secretsstoreimport) manifest with the decoding parameter set:

   ```yaml
   apiVersion: deckhouse.io/v1alpha1
   kind: SecretsStoreImport
   metadata:
     name: myapp-ssi
     namespace: myapp-namespace
   spec:
     type: CSI
     role: myapp-role
     files:
       - name: "keytab"
         decodeBase64: true
         source:
           path: "demo-kv/data/myapp-secret"
           key: "keytab"
   ```

1. A binary file named `keytab` will be created in the container.

## Autorotation feature

The autorotation feature in the `secrets-store-integration` module is enabled by default. Every two minutes, the module polls Stronghold and synchronizes secrets in the mounted file if they have changed.

There are two ways to track changes to the secret file in a pod:

- watch the modification time of the mounted file and react when it changes;
- use the `inotify` API, which provides a file system event subscription mechanism.

`Inotify` is part of the Linux kernel. Once a change is detected, there are many possible responses depending on the application architecture and programming language. The simplest option is to make Kubernetes restart the pod by failing the `livenessProbe`.

Example of using `inotify` in a Python application:

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

Example of using `inotify` in a Go application:

```go
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
            log.Println("file modified")
        }
    case err := <-watcher.Error:
        log.Println("error:", err)
    }
}
```

### Limitations when updating secrets

Files with secrets are not updated if `subPath` is used.

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
