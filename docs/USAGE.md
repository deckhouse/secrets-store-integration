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

При включении модуля в кластере появляется mutating-webhook, который при наличии у пода аннотации `secrets-store.deckhouse.io/role` изменяет манифест пода,
добавляя туда инжектор. В измененном поде добавляется инит-контейнер, который помещает из служебного образа собранный статически бинарный файл-инжектор
в общую для всех контейнеров пода временную директорию. В остальных контейнерах оригинальные команды запуска заменяются на запуск файла-инжектора,
который получает из Vault-совместимого хранилища необходимые данные, используя для подключения сервисный аккаунт приложения, помещает эти переменные в ENV процесса, после чего выполняет системный вызов execve, запуская оригинальную команду.

Если в манифесте пода у контейнера отсутствует команда запуска, то выполняется извлечение манифеста образа из хранилица образов (реджистри),
и команда извлекается из него.
Для получения манифеста из приватного хранилища образов используются заданные в манифесте пода учетные данные из `imagePullSecrets`.

Доступные аннотации, позволяющие изменять поведение инжектора
| Аннотация                                        | Умолчание |  Назначение |
|--------------------------------------------------|-----------|-------------|
|secrets-store.deckhouse.io/role                   |           | Задает роль, с которой будет выполнено подключение к хранилищу секретов |
|secrets-store.deckhouse.io/env-from-path          |           | Задает путь к секрету в хранилище, из которого будут извлечены все ключи и помещены в environment |
|secrets-store.deckhouse.io/ignore-missing-secrets | false     | Запускает оригинальное приложение в случае ошибки получения секрета из хранилища |
|secrets-store.deckhouse.io/client-timeout         | 10s       | Таймаут операции получения секретов |
|secrets-store.deckhouse.io/mutate-probes          | false     | Инжектирует переменные окружения в пробы |
|secrets-store.deckhouse.io/log-level              | info      | Уровень логирования |
|secrets-store.deckhouse.io/enable-json-log        | false     | Формат логов, строка или json |

Используя инжектор вы сможете задавать в манифестах пода вместо значений env-шаблоны, которые будут заменяться на этапе запуска контейнера на значения из хранилища.

Пример: извлечь из Vault-совместимого хранилища ключ `mypassword` из kv2-секрета по адресу `secret/myapp`:

```yaml
env:
  - name: PASSWORD
    value: secrets-store:secret/data/myapp#mypassword
```

Пример: извлечь из Vault-совместимого хранилища ключ `mypassword` версии `4` из kv2-секрета по адресу `secret/myapp`:

```yaml
env:
  - name: PASSWORD
    value: secrets-store:secret/data/myapp#mypassword#4
```

Шаблон может также находиться в ConfigMap или в Secret и быть подключен с помощью `envFrom`
```yaml
envFrom:
  - secretRef:
      name: app-secret-env
  - configMapRef:
      name: app-env

```
Инжектирование реальных секретов из Vault-совместимого хранилища выполнится только на этапе запуска приложения, в Secret и ConfigMap будут находиться шаблоны.


### Подключение переменных из ветки хранилища (всех ключей одного секрета)

Создадим под с названием `myapp1`, который подключит все переменные из хранилища по пути `secret/data/myapp`:

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

Применим его:

```bash
kubectl create --filename myapp1.yaml
```

Проверим логи пода после его запуска, мы должны увидеть все переменные из `secret/data/myapp`:

```bash
kubectl -n my-namespace logs myapp1
```

Удалим под

```bash
kubectl -n my-namespace delete pod myapp1 --force
```

### Подключение явно заданных переменных из хранилища

Создадим тестовый под с названием `myapp2`, который подключит требуемые переменные из хранилища по шаблону:

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

Применим его:

```bash
kubectl create --filename myapp2.yaml
```

Проверим логи пода после его запуска, мы должны увидеть переменные из `secret/data/myapp`:

```bash
kubectl -n my-namespace logs myapp2
```

Удалим под

```bash
kubectl -n my-namespace delete pod myapp2 --force
```

## Монтирование секрета из хранилища в качестве файла в контейнер

Для доставки секретов в приложение нужно использовать CustomResource “SecretStoreImport”.

Создайте namespace:

```bash
kubectl create namespace my-namespace
```

Создайте в кластере CustomResource _SecretsStoreImport_ с названием “myapp”:

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

Создайте в кластере тестовый под с названием `myapp3`, который подключит требуемые переменные из хранилища в виде файла:

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

Проверьте логи пода после его запуска (должно выводиться содержимое файла `/mnt/secrets/db-password`):

```bash
kubectl -n my-namespace logs myapp3
```

Удалите под:

```bash
kubectl -n my-namespace delete pod myapp3 --force
```
