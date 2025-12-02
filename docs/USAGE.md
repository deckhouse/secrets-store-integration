---
title: "The secrets-store-integration module: usage"
description: Usage of the secrets-store-integration Deckhouse module.
---

## Configuring the module to work with Deckhouse Stronghold

1. Enable the `stronghold` module following the [guide](/modules/stronghold/usage.html#how-to-enable-the-module).

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

   The [connectionConfiguration](configuration.html#parameters-connectionconfiguration) paramater is optional and set to `DiscoverLocalStronghold` value by default.

## Configuring the module to work with the external secret store

The module requires a pre-configured secret vault compatible with HashiCorp Vault. An authentication path must be preconfigured in the vault. An example of how to configure the secret vault is provided [further](#setting-up-the-test-environment).

To ensure that each API request is encrypted, sent, and processed by the correct recipient, a valid public Certificate Authority certificate used by the secret store is required. A `caCert` variable in the module configuration must refer to such a CA certificate in PEM format.

The following is an example module configuration for using a Vault-compliant secret store running at `secretstoreexample.com` on a regular port (`443`):

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
        kD8MMYv5NHHko/3jlBJCjVG6cI+5HaVekOqRN9l3D9ZXsdg2RdXLU8CecQAD7yYa
        ................................................................
        C2ZTJJonuI8dA4qUadvCXrsQqJEa2nw1rql4LfPP5ztJz1SwNCSYH7EmwqW+Q7WR
        bZ6GhOj=
        -----END CERTIFICATE-----
    connectionConfiguration: Manual
```

{{< alert level="info">}}
It is strongly recommended that you set the `caCert` variable. Otherwise, the module will use system ca-certificates.
{{< /alert >}}

## Setting up the test environment

{{< alert level="info">}}
To run the following commands, you will need a root-access token and the Stronghold address.
You can get such a root token while initializing a new secrets store.

It is assumed in all subsequent commands that these settings are specified in environment variables.

```bash
export VAULT_TOKEN=xxxxxxxxxxx
export VAULT_ADDR=https://secretstoreexample.com
```

{{< /alert >}}

{{< alert level="info">}}

This section includes two types of command examples:

* Commands using the [`d8` CLI tool](#cli-tool-d8-for-stronghold-commands).
* Commands using `curl` for direct requests to the secrets store API.

{{< /alert >}}

Before the secret injection, prepare the test environment:

1. Create a kv2-type secret in Stronghold at `demo-kv/myapp-secret` and copy `DB_USER` and `DB_PASS` there.

   * Enable and create the Key-Value store:

     ```bash
     d8 stronghold secrets enable -path=demo-kv -version=2 kv
     ```

     An alternative command using `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request POST \
       --data '{"type":"kv","options":{"version":"2"}}' \
       ${VAULT_ADDR}/v1/sys/mounts/demo-kv
     ```

   * Set the database username and password as the secret's value:

     ```bash
     d8 stronghold kv put demo-kv/myapp-secret DB_USER="username" DB_PASS="secret-password"
     ```

     An alternative command using `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request PUT \
       --data '{"data":{"DB_USER":"username","DB_PASS":"secret-password"}}' \
       ${VAULT_ADDR}/v1/demo-kv/data/myapp-secret
     ```

   * Check the recorded secret:

     ```bash
     d8 stronghold kv get demo-kv/myapp-secret
     ```

     An alternative command using `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       ${VAULT_ADDR}/v1/demo-kv/data/myapp-secret
     ```

1. If necessary, add an authentication path ([`authPath`](/modules/secrets-store-integration/configuration.html#parameters-connection-authpath)) for authentication and authorization to Stronghold using the Kubernetes API of the remote cluster.

   * By default, the authentication method named `kubernetes_local` is enabled and configured in Stronghold via Kubernetes API of the cluster on which Stronghold is running. If you need to configure access via remote clusters, set the authentication path (`authPath`) and enable authentication and authorization in Stronghold via Kubernetes API for each cluster:

     ```bash
     d8 stronghold auth enable -path=remote-kube-1 kubernetes
     ```

     An alternative command using `curl`:

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

     An alternative command using `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request PUT \
       --data '{"kubernetes_host":"https://api.kube.my-deckhouse.com"}' \
       ${VAULT_ADDR}/v1/auth/remote-kube-1/config
     ```

1. Create a policy named `myapp-ro-policy` in Stronghold that allows reading secrets at `demo-kv/myapp-secret`:

   ```bash
   d8 stronghold policy write myapp-ro-policy - <<EOF
   path "demo-kv/data/myapp-secret" {
     capabilities = ["read"]
   }
   EOF
   ```

   An alternative command using `curl`:

   ```bash
   curl \
     --header "X-Vault-Token: ${VAULT_TOKEN}" \
     --request PUT \
     --data '{"policy":"path \"demo-kv/data/myapp-secret\" {\n capabilities = [\"read\"]\n}\n"}' \
     ${VAULT_ADDR}/v1/sys/policies/acl/myapp-ro-policy
   ```

1. Create a role in Stronghold for the `myapp-sa` service account in the `myapp-namespace` namespace and bind the policy you created earlier to it.

   {{< alert level="danger">}}
   In addition to the Stronghold side settings, you must configure the authorization permissions of the ServiceAccounts used in the Kubernetes cluster.
   See the [following section](#allowing-a-serviceaccount-to-log-in-to-stronghold) section for required settings.
   {{< /alert >}}

   * Create a role made of the namespace and policy name. Bind it to the `myapp-sa` ServiceAccount in the `myapp-namespace` namespace and the `myapp-ro-policy` policy:

     {{< alert level="info">}}
     The recommended TTL value of the Kubernetes token is `10m`.
     {{< /alert >}}

     ```bash
     d8 stronghold write auth/kubernetes_local/role/myapp-role \
         bound_service_account_names=myapp-sa \
         bound_service_account_namespaces=myapp-namespace \
         policies=myapp-ro-policy \
         ttl=10m
     ```

     An alternative command using `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request PUT \
       --data '{"bound_service_account_names":"myapp-sa","bound_service_account_namespaces":"myapp-namespace","policies":"myapp-ro-policy","ttl":"10m"}' \
       ${VAULT_ADDR}/v1/auth/kubernetes_local/role/myapp-role
     ```

   * Repeat this for remote clusters, specifying a different authentication path:

     ```bash
     d8 stronghold write auth/remote-kube-1/role/myapp-role \
         bound_service_account_names=myapp-sa \
         bound_service_account_namespaces=myapp-namespace \
         policies=myapp-ro-policy \
         ttl=10m
     ```

     An alternative command using `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request PUT \
       --data '{"bound_service_account_names":"myapp-sa","bound_service_account_namespaces":"myapp-namespace","policies":"myapp-ro-policy","ttl":"10m"}' \
       ${VAULT_ADDR}/v1/auth/remote-kube-1/role/myapp-role
     ```

   These settings allow any pod within the `myapp-namespace` namespace in both Kubernetes clusters that uses the `myapp-sa` ServiceAccount to authenticate, authorize, and read secrets in Stronghold according to the `myapp-ro-policy` policy.

1. Create a `myapp-namespace` namespace in the cluster:

   ```bash
   d8 k create namespace myapp-namespace
   ```

1. Create a `myapp-sa` service account in the created namespace:

   ```bash
   d8 k -n myapp-namespace create serviceaccount myapp-sa
   ```

## Allowing a ServiceAccount to log in to Stronghold

To log in to Stronghold, a pod uses a token generated for its ServiceAccount. In order for Stronghold to be able to check the validity of the ServiceAccount data, the Stronghold used by the service must have a permission to `get`, `list`, and `watch` for the `tokenreviews.authentication.k8s.io` and `subjectaccessreviews.authorization.k8s.io` endpoints. You can also use the `system:auth-delegator` ClusterRole for this.

Stronghold can use different credentials to make requests to the Kubernetes API:

* A token of the application that is trying to log in to Stronghold. In this case, each service that logs in to Stronghold must have the `system:auth-delegator` ClusterRole (or the API permissions listed above) in the ServiceAccount it uses. Refer to examples in the [Stronghold documentation](https://deckhouse.io/products/stronghold/documentation/user/auth/kubernetes.html#use-the-stronghold-clients-jwt-as-the-reviewer-jwt).

* A static token created specifically for Stronghold ServiceAccount that has the necessary permissions. Setting up Stronghold for this case is described in detail in the [Stronghold documentation](https://deckhouse.io/products/stronghold/documentation/user/auth/kubernetes.html#continue-using-long-lived-tokens).

## Injecting environment variables

### How injecting works

When the module is enabled, a mutating-webhook becomes available in the cluster. It modifies the pod manifest, adding an injector, if the pod has the `secrets-store.deckhouse.io/role` annotation. An init container is added to the modified pod. The init container copies a statically compiled binary injector file from a service image into a temporary directory shared by all containers in the pod. In the other containers, the original startup commands are replaced with a command that starts the injector. It then fetches the required data from a Vault-compatible storage using the application's service account, sets these variables in the process ENV, and then issues an `execve` system call, invoking the original command.

If the container does not have a startup command in the pod manifest, the image manifest is retrieved from the image registry,
and the command is retrieved from it.
The credentials from `imagePullSecrets` specified in the pod manifest are used to retrieve the manifest from the private image registry.

The following are the available annotations to modify the injector behavior:
<style>.annotations-table-style + .table-wrapper td:first-child{min-width: 317px}</style>
<div class="annotations-table-style"></div>

| Annotation                                       | Default value |  Description |
|--------------------------------------------------|-------------|-------------|
|`secrets-store.deckhouse.io/addr`                   | From module | Address of the secrets store in the `https://stronghold.mycompany.tld:8200` format |
|`secrets-store.deckhouse.io/tls-secret`             | From module | Name of the Secret object in Kubernetes that contains the `ca.crt` key with a CA certificate value in PEM format |
|`secrets-store.deckhouse.io/tls-skip-verify`        | `false`       | Disable verification of TLS certificates |
|`secrets-store.deckhouse.io/auth-path`              | From module | Path to use for authentication |
|`secrets-store.deckhouse.io/namespace`              | From module | Namespace that will be used to connect to the store |
|`secrets-store.deckhouse.io/role`                   |             | Sets the role to be used to connect to the secret store |
|`secrets-store.deckhouse.io/env-from-path`          |             | String containing a comma-delimited list of paths to secrets in the repository, from which all keys will be extracted and placed in the environment. Priority is given to keys that are closer to the end of the list |
|`secrets-store.deckhouse.io/ignore-missing-secrets` | `false`       | Runs the original application if an attempt to retrieve a secret from the store fails |
|`secrets-store.deckhouse.io/client-timeout`         | `10s`         | Timeout to use for secrets retrieval |
|`secrets-store.deckhouse.io/mutate-probes`          | `false`       | Injects environment variables into the probes |
|`secrets-store.deckhouse.io/log-level`              | `info`        | Logging level |
|`secrets-store.deckhouse.io/enable-json-log`        | `false`       | Enables JSON format for logging |
|`secrets-store.deckhouse.io/skip-mutate-containers` |           | Space-separated list of container names excluded from the injection |

The injector allows you to specify env templates instead of values in the pod manifests. They will be replaced at the container startup stage with the values from the store.

{{< alert level="info">}}
Including variables from a store branch has a higher priority than including explicitly defined variables from the store. This means that when using both the `secrets-store.deckhouse.io/env-from-path` annotation with a path to a secret that contains, for example, the `MY_SECRET` key, and an environment variable in the manifest with the same name:

```yaml
env:
  - name: MY_SECRET
    value: secrets-store:demo-kv/data/myapp-secret#password
```

the `MY_SECRET` environment variable inside the container will be set to the value of the secret from the **annotation**.
{{< /alert >}}

An example of retrieving the `DB_PASS` key from the kv2-secret at `demo-kv/myapp-secret` from the Vault-compatible store:

```yaml
env:
  - name: PASSWORD
    value: secrets-store:demo-kv/data/myapp-secret#DB_PASS
```

An example of retrieving the `DB_PASS` key version `4` from the kv2 secret at `demo-kv/myapp-secret` from the Vault-compatible store:

```yaml
env:
  - name: PASSWORD
    value: secrets-store:demo-kv/data/myapp-secret#DB_PASS#4
```

The template can also be stored in the ConfigMap or in the Secret and can be hooked up using `envFrom`:

```yaml
envFrom:
  - secretRef:
      name: app-secret-env
  - configMapRef:
      name: app-env
```

The actual secrets from the Vault-compatible store will be injected at the application startup. The Secret and ConfigMap will only contain the templates.

### Retrieving environment variables from the store branch (all keys of a single secret)

1. Create a Pod named `myapp1` that will retrieve all variables from the store at the `demo-kv/data/myapp-secret` path:

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

1. Apply the created manifest:

   ```bash
   d8 k create --filename myapp1.yaml
   ```

1. Check the Pod logs after it has been successfully started. In the output, you should see all the variables from `demo-kv/data/myapp-secret`:

   ```bash
   d8 k -n myapp-namespace logs myapp1
   ```

1. Delete the Pod:

   ```bash
   d8 k -n myapp-namespace delete pod myapp1 --force
   ```

### Retrieving explicitly specified variables from the store

1. Create a test Pod named `myapp2` that will retrieve the required variables from the store according to the template:

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

1. Apply the created manifest:

   ```bash
   d8 k create --filename myapp2.yaml
   ```

1. Check the Pod logs after it has been successfully started. In the output, you should see the variables from `demo-kv/data/myapp-secret`:

   ```bash
   d8 k -n myapp-namespace logs myapp2
   ```

1. Delete the Pod:

   ```bash
   d8 k -n myapp-namespace delete pod myapp2 --force
   ```

## Mounting a secret from the store as a file in a container

Use the [SecretStoreImport](/modules/secrets-store-integration/cr.html#secretsstoreimport) custom resource to deliver secrets to the application.

In this example, you will be using the ServiceAccount `myapp-sa` and namespace `myapp-namespace` that were created earlier when you [set up the test environment](#setting-up-the-test-environment).

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

1. Create a test Pod in the cluster named `myapp3` that will retrieve the required variables from the store as a file:

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
           secretsStoreImport: "myapp-ssi"
   ```

   Once these resources have been applied, a Pod will be created, inside which a container named `backend` will then be started. This container's filesystem will have a directory `/mnt/secrets`, with the `secrets` volume mounted to it. The directory will contain a `db-password` file with the password for database (`DB_PASS`) from the Stronghold Key-Value store.

1. Check the Pod logs after it has been successfully started (you should see the contents of the `/mnt/secrets/db-password` file):

   ```bash
   d8 k -n myapp-namespace logs myapp3
   ```

1. Delete the Pod:

   ```bash
   d8 k -n myapp-namespace delete pod myapp3 --force
   ```

### Delivering binary files to a container

There are situations when you need to deliver a binary file to a container.
This could be a JKS container with keys or a keytab for Kerberos authentication.
In this case, you can encode the binary file using Base64 and place it in the secrets store. When you retrieve it,
the CSI driver will decode your data and place the binary file in the container. To do this, set the `decodeBase64`
parameter to `true` for the corresponding file.
If decoding fails (for example, if the storage contains an invalid Base64), the container will not be created.

Example:

1. Encode the file using Base64 and place it into the store:

   ```bash
   d8 stronghold kv put demo-kv/myapp-secret keytab=$(cat /path/to/keytab_file | base64 -w0)
   ```

1. Create a [SecretsStoreImport](/modules/secrets-store-integration/cr.html#secretsstoreimport) manifest including the parameter required to decode the file:

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

### The autorotation feature

The autorotation feature of the `secret-store-integration` module is enabled by default. Every two minutes, the module polls Stronghold and synchronizes the secrets in the mounted file if it has been changed.

There are two ways to keep track of changes to the secret file in the pod:

* Keeping track of when the mounted file changes, reacting to changes in the file.
* Using the inotify API, which provides a mechanism for subscribing to file system events. Inotify is part of the Linux kernel. Once a change is detected, there are a large number of options for responding to the change event, depending on the application architecture and programming language used. The most simple one is to force Kubernetes to restart the pod by failing the liveness probe.

The following is an example of using inotify in a Python application leveraging the inotify Python package:

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

The following is an example of using inotify in a Go application using the inotify package:

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

Files with secrets will not be rotated if `subPath` is used.

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

## CLI tool d8 for Stronghold commands

Deckhouse CLI (`d8`) is a multipurpose tool that is required to run commands like `d8 stronghold` from the terminal.

To install `d8`, use one of the options described in the [CLI tool documentation](/products/kubernetes-platform/documentation/latest/cli/d8/#installing-the-executable).
