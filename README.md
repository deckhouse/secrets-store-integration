## How to set up the module for work with Deckhouse Stronghold

To automatically set up the secrets-store-integration module for work with [Deckhouse Stronghold](../../stronghold/ you need to [turn on](../../stronghold/stable/usage.html) the Stronghold module previously.

After that just apply the ModuleConfig:

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: ModuleConfig
metadata:
  name: secrets-store-integration
spec:
  enabled: true
  version: 1
```

The [connectionConfiguration](../../secrets-store-integration/stable/configuration.html#parameters-connectionconfiguration) paramater is optional, it is set to `DiscoverLocalStronghold` value by default.

## How to set up the module for work with external secrets store

To operate the module, a preconfigured secrets store compatible with HashiCorp Vault is required. The store must have an authentication path configured in advance. An example of configuring the secrets store is provided in the FAQ.

To ensure that each API request is encrypted, sent, and responded to by the correct recipient, a valid public Certificate Authority (CA) certificate used by the secrets store is required. You need to use such a public CA certificate in PEM format as the caCert variable in the module configuration.

The following is an example module configuration for using a Vault-compatible secrets store deployed at "secretstoreexample.com" on the default TLS port, 443 TLS. Please note that you'll need to replace the variable values in the configuration with the actual values corresponding to your environment.

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

To verify that every API request is encrypted, sent, and answered by the exact host, we must embed the caCert of the used Vault-compatible secrets storage in PEM format in the ModuleConfig.

**It is strongly recommended to set the caCert variable, if not, the module will use the system ca-certificates.**

To deliver secrets to the application, use the “SecretStoreImport” CustomResource.

## Mounting the vault’s secret as a file to a container:

Let’s create namespace

```bash
kubectl create namespace my-namespace1
```

Let’s create SecretsStoreImport CustomResource with the “python-backend” name:

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

Apply it:

```bash
kubectl apply --filename python-backend-secrets-store-import.yaml
```

Create ServiceAccount with “backend-sa” name:

```bash
kubectl -n my-namespace1 create serviceaccount backend-sa
```

Create a test deployment with the name “backend”, which starts a pod that will be able to access the needed vault’s secret:

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

Apply it:

```bash
kubectl apply --filename backend-deployment.yaml
```

After the pod successfully starts, let’s check if we have a secret inside:

```bash
kubectl exec backend -- cat /mnt/secrets/db-password
```
