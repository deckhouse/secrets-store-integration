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

Для доставки секретов в приложение нужно использовать CustomResource “SecretStoreImport”.

## Монтирование секрета из хранилища в качестве файла в контейнер:

Создадим неймспейс

```bash
kubectl create namespace my-namespace1
```

Создадим CustomResource SecretsStoreImport с названием “python-backend”:

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

Применим его:

```bash
kubectl apply --filename python-backend-secrets-store-import.yaml
```

Создадим ServiceAccount с названием “backend-sa”:

```bash
kubectl -n my-namespace1 create serviceaccount backend-sa
```

Создадим тестовый деплоймент с названием “backend”, который запускает под с доступом к нужному секрету в хранилище:

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

Применим его:

```bash
kubectl apply --filename backend-deployment.yaml
```

Проверим наличие секрета в поде после его запуска:

```bash
kubectl exec backend -- cat /mnt/secrets/db-password
```

