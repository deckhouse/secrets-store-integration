---
title: "The secrets-store-integration module: usage"
description: Usage of the secrets-store-integration Deckhouse module.
---

## Configuring the module to work with Deckhouse Stronghold

[Enable](../../stronghold/stable/usage.html#how-to-enable-the-module) the Stronghold module beforehand to automatically configure the secrets-store-integration module to work with [Deckhouse Stronghold](../../stronghold/).

Next, apply the ModuleConfig:

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

The module requires a pre-configured secret vault compatible with HashiCorp Vault. An authentication path must be preconfigured in the vault. An example of how to configure the secret vault is provided in the FAQ.

To ensure that each API request is encrypted, sent to, and replied by the correct recipient, a valid public Certificate Authority certificate used by the secret store is required. A `caCert` variable in the module configuration must refer to such a CA certificate in PEM format.

The following is an example module configuration for using a Vault-compliant secret store running at "secretstoreexample.com" on a regular port (443 TLS). Note that you will need to replace the variable values in the configuration with the values that match your environment.

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

To ensure that each API request is encrypted, sent and responded to by the exact host that sent it, insert the PEM-formatted caCert for the Vault-compatible secret store into ModuleConfig.

**It is strongly recommended to set the `caCert` variable. Otherwise, the module will use system ca-certificates.**

## Setting up the test environment

Before moving on to the instructions for secret injection given in the examples below,

1. Create a kv2 type secret in Stronghold in `secret/myapp` and copy `DB_USER` and `DB_PASS` there.
2. Create a policy in Stronghold that allows reading secrets from `secret/myapp`.
3. Create a `myapp` role in Stronghold for the `myapp` service account in the `my-namespace` namespace and bind the policy you created earlier to it.
4. Create a `my-namespace` namespace in the cluster.
5. Create a `myapp` service account in the created namespace.

Example commands to set up the environment:

```bash
stronghold secrets enable -path=secret -version=2 kv

stronghold kv put secret/myapp DB_USER="username" DB_PASS="secret-password"

stronghold policy write myapp - <<EOF
path "secret/data/myapp" {
  capabilities = ["read"]
}
EOF

stronghold write auth/kubernetes_local/role/myapp \
    bound_service_account_names=myapp \
    bound_service_account_namespaces=my-namespace \
    policies=myapp \
    ttl=60s

kubectl create namespace my-namespace

kubectl -n my-namespace create serviceaccount myapp
```

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

Create a namespace:

```bash
kubectl create namespace my-namespace
```

Create a _SecretsStoreImport_ CustomResource named `myapp` in the cluster:

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: SecretsStoreImport
metadata:
  name: myapp
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

Check the pod logs after it has been successfully started (you should see the contents of the `/mnt/secrets/db-password` file):

```bash
kubectl -n my-namespace logs myapp3
```

Delete the pod:

```bash
kubectl -n my-namespace delete pod myapp3 --force
```
