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

### ВАЖНО

Для использования иструкций по инжектированию секретов из примеров ниже вам понадобится:

1. Создать в Stronhold секрет типа kv2 по пути `secret/myapp` и поместить туда значения `DB_USER` и `DB_PASS`
2. Создать в Stronhold политику, разрешающую чтение секретов по пути `secret/myapp`
3. Создать в Stronhold роль `myapp` для сервис-аккаунта `myapp` в неймспейсе `my-namespace` и привязать к ней созданную ранее политику

## Инжектирование переменных окружения из Stronghold:

Создадим неймспейс

```bash
kubectl create namespace my-namespace
```

Создадим ServiceAccount с названием `myapp`:

```bash
kubectl -n my-namespace create serviceaccount myapp
```

Создадим под с названием `myapp1`, который подключит все переменные из хранилища по пути `secret/data/myapp`:

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: myapp1
  namespace: my-namespace
  annotations:
    secret-store.deckhouse.io/role: "myapp"
    secret-store.deckhouse.io/env-from-path: secret/data/myapp
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

Создадим тестовый под с названием `myapp2`, который подключит требуемые переменные из хранилища по шаблону:

```yaml
kind: Pod
apiVersion: v1
metadata:
  name: myapp2
  namespace: my-namespace
  annotations:
    secret-store.deckhouse.io/role: "myapp"
spec:
  serviceAccountName: myapp
  containers:
  - image: alpine:3.20
    env:
    - name: DB_USER
      value: stronghold:secret/data/myapp#DB_USER
    - name: DB_PASS
      value: stronghold:secret/data/myapp#DB_PASS
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

```bash
kubectl create --filename myapp-secrets-store-import.yaml
```

Создадим ServiceAccount с названием `myapp`:

```bash
kubectl -n my-namespace create serviceaccount myapp
```

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

