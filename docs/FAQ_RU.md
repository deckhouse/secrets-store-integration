---
title: "Модуль secrets-store-integration: FAQ"
description: Как настроить HashiCorp Vault в качестве secret store. Пример реализации авторотации секретов.
---

## Как настроить HashiCorp Vault в качестве secret store для использования с модулем secrets-store-integration?

Прежде всего нам необходим адрес и токен с правами root от Vault.
Такой токен с правами root можно получить во время инициализации нового secrets store.

```bash
export VAULT_TOKEN=xxxxxxxxxxx
export VAULT_ADDR=https://secretstoreexample.com
```

В этом руководстве мы приводим два вида примерных команд: 
- команда с использованием консольной версии HashiCorp Vault (руководство по установке: https://developer.hashicorp.com/vault/docs/install);
- команда с использованием curl для выполнения прямых запросов в API secrets store.

Включим и создадим Key-Value хранилище:

```bash
vault secrets enable -path=secret -version=2 kv
```

Та же команда в виде HTTP-запроса, сделанного с помощью curl:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request POST \
  --data '{"type":"kv","options":{"version":"2"}}' \
  ${VAULT_ADDR}/v1/sys/mounts/secret
```

Зададим пароль базы в качестве значения секрета:

```bash
vault kv put secret/database-for-python-app password="db-secret-password"
```

Та же команда в виде HTTP-запроса, сделанного с помощью curl:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request PUT \
  --data '{"data":{"password":"db-secret-password"}}' \
  ${VAULT_ADDR}/v1/secret/data/database-for-python-app
```

Проверим, правильно ли записался пароль:

```bash
vault kv get secret/database-for-python-app
```

Та же команда в виде HTTP-запроса, сделанного с помощью curl:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  ${VAULT_ADDR}/v1/secret/data/database-for-python-app
```

Задаём путь аутентификации (authPath) и включаем аутентификацию и авторизацию в Vault с помощью Kubernetes API:

```bash
vault auth enable -path=main-kube kubernetes
```

Та же команда в виде HTTP-запроса, сделанного с помощью curl:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request POST \
  --data '{"type":"kubernetes"}' \
  ${VAULT_ADDR}/v1/sys/auth/main-kube
```

Если у нас больше чем один кластер, то задаём путь аутентификации (authPath) и включаем аутентификацию и авторизацию в Vault с помощью Kubernetes API второго кластера:

```bash
vault auth enable -path=secondary-kube kubernetes
```

Та же команда в виде HTTP-запроса, сделанного с помощью curl:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request POST \
  --data '{"type":"kubernetes"}' \
  ${VAULT_ADDR}/v1/sys/auth/secondary-kube
```

Задаём адрес Kubernetes API для каждого кластера (в данном случае, это сервис k8s API server):

```bash
vault write auth/main-kube/config \
  kubernetes_host="https://api.kube.my-deckhouse.com"
```

Та же команда в виде HTTP-запроса, сделанного с помощью curl:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request PUT \
  --data '{"kubernetes_host":"https://api.kube.my-deckhouse.com"}' \
  ${VAULT_ADDR}/v1/auth/main-kube/config
```

Для второго кластера:

```bash
vault write auth/secondary-kube/config \
  kubernetes_host="https://10.11.12.10:443"
```

Та же команда в виде HTTP-запроса, сделанного с помощью curl:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request PUT \
  --data '{"kubernetes_host":"https://10.11.12.10:443"}' \
  ${VAULT_ADDR}/v1/auth/secondary-kube/config
```

Создаём в Vault политику с названием «backend», разрешающую чтение секретов:

```bash
vault policy write backend - <<EOF
path "secret/data/database-for-python-app" {
  capabilities = ["read"]
}
EOF
```

Та же команда в виде HTTP-запроса, сделанного с помощью curl:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request PUT \
  --data '{"policy":"path \"secret/data/database-for-python-app\" {\n capabilities = [\"read\"]\n}\n"}' \
  ${VAULT_ADDR}/v1/sys/policies/acl/backend
```

Создаём роль, состоящую из названия пространства имён и приложения. Связываем её с ServiceAccount «backend-sa» из пространства имён «my-namespace1» и политикой «backend»:

```bash
vault write auth/main-kube/role/my-namespace1_backend \
    bound_service_account_names=backend-sa \
    bound_service_account_namespaces=my-namespace1 \
    policies=backend \
    ttl=10m
```

Та же команда в виде HTTP-запроса, сделанного с помощью curl:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request PUT \
  --data '{"bound_service_account_names":"backend-sa","bound_service_account_namespaces":"my-namespace1","policies":"backend","ttl":"10m"}' \
  ${VAULT_ADDR}/v1/auth/main-kube/role/my-namespace1_backend
```

Повторяем то же самое для второго кластера, указав другой путь аутентификации:

```bash
vault write auth/secondary-kube/role/my-namespace1_backend \
    bound_service_account_names=backend-sa \
    bound_service_account_namespaces=my-namespace1 \
    policies=backend \
    ttl=10m
```

Та же команда в виде HTTP-запроса, сделанного с помощью curl:

```bash
curl \
  --header "X-Vault-Token: ${VAULT_TOKEN}" \
  --request PUT \
  --data '{"bound_service_account_names":"backend-sa","bound_service_account_namespaces":"my-namespace1","policies":"backend","ttl":"10m"}' \
  ${VAULT_ADDR}/v1/auth/secondary-kube/role/my-namespace1_backend
```

**Рекомендованное значение TTL для токена Kubernetes составляет 10m.**

Эти настройки позволяют любому поду из пространства имён «my-namespace1» из обоих K8s-кластеров, который использует ServiceAccount «backend-sa», аутентифицироваться и авторизоваться в Vault для чтения секретов согласно политике «backend».

## Как использовать авторотацию секретов, примонтированных как файл в контейнер без его перезапуска?

Функция авторотации секретов в модуле secret-store-integration включена по умолчанию. Каждые две минуты модуль опрашивает Vault и синхронизирует секреты в примонтированном файле в случае его изменения.

Создадим ServiceAccount ```backend-sa```

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: backend-sa
  namespace: my-namespace1
```

Пример CustomResource SecretStoreImport:

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

Пример Deployment `backend`, который использует указанный выше SecretStoreImport как том, чтоб доставить пароль от базы данных в файловую систему приложения:

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

После применения этих ресурсов будет запущен под с названием `backend`, внутри которого будет каталог `/mnt/secrets` с примонтированным внутрь томом `secrets`. Внутри каталога будет лежать файл `db-password` с паролем от базы данных из Vault.

Есть два варианта следить за изменениями файла с секретом в поде. Первый - следить за временем изменения примонтированного файла, реагируя на его изменение. Второй - использовать inotify API, который предоставляет механизм для подписки на события файловой системы. Inotify является частью ядра Linux. После обнаружения изменений есть большое количество вариантов реагирования на событие изменения в зависимости от используемой архитектуры приложения и используемого языка программирования. Самый простой — заставить K8s перезапустить под, перестав отвечать на liveness-пробу.

Пример использования inotify в приложении на Python с использованием пакета inotify:

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

Пример использования inotify в приложении на Go, используя пакет inotify:

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

## Ограничения при обновлении секретов

Файлы с секретами не будут обновляться, если будет использован `subPath`.

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
