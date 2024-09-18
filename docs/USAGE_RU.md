---
title: "The secrets-store-integration module: примеры"
description: Использование модуля secrets-store-integration.
---

## Настройка модуля для работы c Deckhouse Stronghold

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

## Настройка модуля для работы с внешним хранилищем

Для работы модуля требуется предварительно настроенное хранилище секретов, совместимое с HashiCorp Vault. В хранилище предварительно должен быть настроен путь аутентификации. Пример настройки хранилища секретом в FAQ.

Чтобы убедиться, что каждый API запрос зашифрован, послан и отвечен правильным адресатом, потребуется валидный публичный сертификат Certificate Authority, который используется хранилищем секретов. Такой публичный сертификат CA в PEM-формате необходимо использовать в качестве переменной `caCert` в конфигурации модуля.

Пример конфигурации модуля для использования Vault-совместимого хранилища секретов, запущенного по адресу «secretstoreexample.com» на TLS-порту по умолчанию - 443 TLS:

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

**Крайне рекомендуется задавать переменную `caCert`. Если она не задана, будет использовано содержимое системного ca-certificates.**

## Подготовка тестового окружения

{{< alert level="info">}}
Для выполнения дальнейших команд необходим адрес и токен с правами root от Vault.
Такой токен можно получить во время инициализации нового secrets store.

Далее в командах будет подразумеваться что данные настойки указаны в переменных окружения.
```bash
export VAULT_TOKEN=xxxxxxxxxxx
export VAULT_ADDR=https://secretstoreexample.com
```
{{< /alert >}}

> В этом руководстве мы приводим два вида примерных команд: 
>   * команда с использованием консольной версии HashiCorp Vault ([руководство по установке](https://developer.hashicorp.com/vault/docs/install));
>   * команда с использованием curl для выполнения прямых запросов в API secrets store.

Для использования инструкций по инжектированию секретов из примеров ниже вам понадобится:

1. Создать в Stronhold секрет типа kv2 по пути `secret/myapp` и поместить туда значения `DB_USER` и `DB_PASS`.
2. Создать в Stronhold политику, разрешающую чтение секретов по пути `secret/myapp`.
3. Создать в Stronhold роль `myapp` для сервис-аккаунта `myapp` в неймспейсе `my-namespace` и привязать к ней созданную ранее политику.
4. Создать в кластере неймспейс `my-namespace`.
5. Создать в созданном неймспейсе сервис-аккаунт `myapp`.

Пример команд, с помощью которых можно подготовить окружение

* Включим и создадим Key-Value хранилище:

  ```bash
  stronghold secrets enable -path=secret -version=2 kv
  ```
  Команда с использованием curl:

  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request POST \
    --data '{"type":"kv","options":{"version":"2"}}' \
    ${VAULT_ADDR}/v1/sys/mounts/secret
  ```

* Зададим имя пользователя и пароль базы данных в качестве значения секрета:

  ```bash
  stronghold kv put secret/myapp DB_USER="username" DB_PASS="secret-password"
  ```
  Команда с использованием curl:

  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"data":{"DB_USER":"username","DB_PASS":"secret-password"}}' \
    ${VAULT_ADDR}/v1/secret/data/myapp
  ```

* Проверим, правильно ли записались секреты:

  ```bash
  stronghold kv get secret/myapp
  ```  
  
  Команда с использованием curl:
  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    ${VAULT_ADDR}/v1/secret/data/myapp
  ```

* Задаём путь аутентификации (`authPath`) и включаем аутентификацию и авторизацию в Stronghold с помощью Kubernetes API:

  ```bash
  stronghold auth enable -path=main-kube kubernetes
  ```
  Команда с использованием curl:
  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request POST \
    --data '{"type":"kubernetes"}' \
    ${VAULT_ADDR}/v1/sys/auth/main-kube
  ```

* Если требуется настроить доступ для более чем одного кластера, то задаём путь аутентификации (`authPath`) и включаем аутентификацию и авторизацию в Stronghold с помощью Kubernetes API для каждого кластера:

  ```bash
  stronghold auth enable -path=secondary-kube kubernetes
  ```
  Команда с использованием curl:
  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request POST \
    --data '{"type":"kubernetes"}' \
    ${VAULT_ADDR}/v1/sys/auth/secondary-kube
  ```

* Задаём адрес Kubernetes API для каждого кластера:

  ```bash
  stronghold write auth/main-kube/config \
    kubernetes_host="https://kubernetes.default.svc:443"
  ```
  Команда с использованием curl:
  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"kubernetes_host":"https://kubernetes.default.svc:443"}' \
    ${VAULT_ADDR}/v1/auth/main-kube/config
  ```

  Для другого кластера:

  ```bash
  stronghold write auth/secondary-kube/config \
    kubernetes_host="https://api.kube.my-deckhouse.com"
  ```
  Команда с использованием curl:
  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"kubernetes_host":"https://api.kube.my-deckhouse.com"}' \
    ${VAULT_ADDR}/v1/auth/secondary-kube/config
  ```

* Создаём в Vault политику с названием `backend`, разрешающую чтение секрета `myapp`:

  ```bash
  stronghold policy write backend - <<EOF
  path "secret/data/myapp" {
    capabilities = ["read"]
  }
  EOF
  ```
  Команда с использованием curl:
  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"policy":"path \"secret/data/myapp\" {\n capabilities = [\"read\"]\n}\n"}' \
    ${VAULT_ADDR}/v1/sys/policies/acl/backend
  ```


* Создаём роль, состоящую из названия пространства имён и политики. Связываем её с ServiceAccount `myapp` из пространства имён `my-namespace` и политикой `backend`:

  {{< alert level="danger">}}
  **Важно!**  
  Помимо настроек со стороны Vault, вы должны настроить разрешения авторизации используемых `serviceAccount` в кластере kubernetes.  
  Подробности в разделе [FAQ](faq.html#как-разрешить-serviceaccount-авторизоваться-в-vault)
  {{< /alert >}}

  ```bash
  stronghold write auth/main-kube/role/my-namespace_backend \
      bound_service_account_names=myapp \
      bound_service_account_namespaces=my-namespace \
      policies=backend \
      ttl=10m
  ```
  Команда с использованием curl:
  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"bound_service_account_names":"myapp","bound_service_account_namespaces":"my-namespace","policies":"backend","ttl":"10m"}' \
    ${VAULT_ADDR}/v1/auth/main-kube/role/my-namespace_backend
  ```


* Повторяем то же самое для остальных кластеров, указав другой путь аутентификации:

  ```bash
  vault write auth/secondary-kube/role/my-namespace_backend \
      bound_service_account_names=myapp \
      bound_service_account_namespaces=my-namespace \
      policies=backend \
      ttl=10m
  ```
  Команда с использованием curl:
  ```bash
  curl \
    --header "X-Vault-Token: ${VAULT_TOKEN}" \
    --request PUT \
    --data '{"bound_service_account_names":"myapp","bound_service_account_namespaces":"my-namespace","policies":"backend","ttl":"10m"}' \
    ${VAULT_ADDR}/v1/auth/secondary-kube/role/my-namespace_backend
  ```


  {{< alert level="info">}}
  **Важно!**  
  Рекомендованное значение TTL для токена Kubernetes составляет 10m.
  {{< /alert >}}

Эти настройки позволяют любому поду из пространства имён `my-namespace1` из обоих K8s-кластеров, который использует ServiceAccount `backend-sa`, аутентифицироваться и авторизоваться в Vault для чтения секретов согласно политике `backend`.

```bash
kubectl create namespace my-namespace

kubectl -n my-namespace create serviceaccount myapp
```

## Инжектирование переменных окружения

### Как работает

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

