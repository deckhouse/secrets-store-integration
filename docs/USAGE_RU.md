---
title: "The secrets-store-integration module: примеры"
description: Использование модуля secrets-store-integration.
---

## Как настроить модуль для работы c Deckhouse Stronghold

Для автоматической настройки работы модуля secrets-store-integration в связке с модулем [Deckhouse Stronghold](../../stronghold/) потребуется ранее [включенный](../../stronghold/stable/usage.html#%D0%BA%D0%B0%D0%BA-%D0%B2%D0%BA%D0%BB%D1%8E%D1%87%D0%B8%D1%82%D1%8C) и настроенный Stronghold.

Далее достаточно применить следующий ресурс:

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: ModuleConfig
metadata:
  name: secrets-store-integration
spec:
  enabled: true
  version: 1
```

Параметр [connectionConfiguration](../../secrets-store-integration/stable/configuration.html#parameters-connectionconfiguration) можно опустить, поскольку он стоит в значении `DiscoverLocalStronghold` по умолчанию.

## Как настроить модуль для работы c внешним хранилищем

Для работы модуля требуется предварительно настроенное хранилище секретов, совместимое с Hashicorp Vault. В хранилище предварительно должен быть настроен путь аутентификациию. Пример настройки хранилища секретом в FAQ.

Для того, чтоб убедиться в том, что каждый API запрос зашифрован, послан и отвечен правильным адресатом, потребуется валидный публичный сертификат Certificate Authority, который используется хранилищем секретов. Такой публичный сертификат CA в PEM-формате необходимо использовать в качестве переменной caCert в конфигурации модуля.

Пример конфигурации модуля для использования Vault-совместимого хранилища секретов, запущенного по адресу “secretstoreexample.com” на TLS-порту по умолчанию - 443 TLS:

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

**Крайне рекомендуется задавать переменную caCert. Если она не задана, будет использовано содержимое системного ca-certificates.**

## Подготовка тестового окружения

Для использования иструкций по инжектированию секретов из примеров ниже вам понадобится:

1. Создать в Stronhold секрет типа kv2 по пути `secret/myapp` и поместить туда значения `DB_USER` и `DB_PASS`
2. Создать в Stronhold политику, разрешающую чтение секретов по пути `secret/myapp`
3. Создать в Stronhold роль `myapp` для сервис-аккаунта `myapp` в неймспейсе `my-namespace` и привязать к ней созданную ранее политику
4. Создать в кластере неймспейс `my-namespace`
5. Создать в созданном неймспейсе сервис-аккаунт `myapp`

Пример команд, с помощью которых можно подготовить окружение

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

## Инжектирование переменных окружения

### Как работает

При включении модуля в кластере появляется mutating-webhook, который, при наличии у пода аннотации `secrets-store.deckhouse.io/role` изменяет манифест пода,
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
|secrets-store.deckhouse.io/ignore-missing-secrets | false     | Запускать оригинальное приложение в случае ошибки получения секрета из хранилища |
|secrets-store.deckhouse.io/client-timeout         | 10s       | Таймаут операции получения секретов |
|secrets-store.deckhouse.io/mutate-probes          | false     | Инжектировать переменные окружения в пробы |
|secrets-store.deckhouse.io/log-level              | info      | Уровень логирования |
|secrets-store.deckhouse.io/enable-json-log        | false     | Формат логов, строка или json |

Используя инжектор вы сможете задавать в манифестах пода вместо значений env шаблоны, которые будут заменяться на этапе запуска контейнера на значения из хранилища.

Пример: извлечь из Vault-совместимого хранилица ключ mypassword из kv2-секрета по адресу secret/myapp

```yaml
env:
  - name: PASSWORD
    value: secrets-store:secret/data/myapp#mypassword
```

Пример: извлечь из Vault-совместимого хранилица ключ mypassword версии 4 из kv2-секрета по адресу secret/myapp

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

## Монтирование секрета из хранилища в качестве файла в контейнер:

Для доставки секретов в приложение нужно использовать CustomResource “SecretStoreImport”.

Создадим неймспейс

```bash
kubectl create namespace my-namespace
```

Создадим CustomResource SecretsStoreImport с названием “myapp”:

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

Применим его:

Создадим тестовый под с названием `myapp3`, который подключит требуемые переменные из хранилища в виде файла:

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
    - while cat /mnt/secrets/db-password; do sleep 5; done
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

Применим его:

```bash
kubectl create --filename myapp3.yaml
```

Проверим логи пода после его запуска, мы должны содержимое файла `/mnt/secrets/db-password`:

```bash
kubectl -n my-namespace logs myapp3
```

Удалим под

```bash
kubectl -n my-namespace delete pod myapp3 --force
```

