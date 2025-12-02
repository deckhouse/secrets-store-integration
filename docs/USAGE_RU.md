---
title: "The secrets-store-integration module: примеры"
description: Использование модуля secrets-store-integration.
---

## Настройка модуля для работы c Deckhouse Stronghold

1. Включите модуль `stronghold`, следуя [инструкции](/modules/stronghold/usage.html#включение-модуля).

1. Чтобы включить модуль `secrets-store-integration`, примените следующий ресурс:

   ```yaml
   apiVersion: deckhouse.io/v1alpha1
   kind: ModuleConfig
   metadata:
     name: secrets-store-integration
   spec:
     enabled: true
     version: 1
   ```

   Параметр [connectionConfiguration](configuration.html#parameters-connectionconfiguration) можно опустить, поскольку он стоит в значении `DiscoverLocalStronghold` по умолчанию.

## Настройка модуля для работы с внешним хранилищем

Для работы модуля требуется предварительно настроенное хранилище секретов, совместимое с HashiCorp Vault. В хранилище предварительно должен быть настроен путь аутентификации. Пример настройки хранилища секретов [ниже](#подготовка-тестового-окружения).

Чтобы убедиться, что каждый API-запрос зашифрован, отправлен и обработан правильным адресатом, потребуется валидный публичный сертификат Certificate Authority, который используется хранилищем секретов. Такой публичный сертификат CA в PEM-формате необходимо использовать в качестве переменной `caCert` в конфигурации модуля.

Пример конфигурации модуля для использования Vault-совместимого хранилища секретов, запущенного по адресу `secretstoreexample.com` на TLS-порту по умолчанию (`443`):

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
Рекомендуется задавать переменную `caCert`. Если она не задана, будет использовано содержимое системного `ca-certificates`.
{{< /alert >}}

## Подготовка тестового окружения

{{< alert level="info">}}
Для выполнения дальнейших команд необходим адрес и токен с правами root от Stronghold.
Такой токен можно получить во время инициализации нового хранилища секретов.

Далее в командах будет подразумеваться что данные настройки указаны в переменных окружения.

```bash
export VAULT_TOKEN=xxxxxxxxxxx
export VAULT_ADDR=https://secretstoreexample.com
```

{{< /alert >}}

{{< alert level="info">}}

В этом разделе приведены два варианта команд с примерами:

* команды с использованием [CLI-утилиты `d8`](#cli-утилита-d8-для-команд-stronghold);
* команды с использованием `curl` для выполнения прямых запросов в API хранилища секретов.

{{< /alert >}}

Перед инжектированием секретов подготовьте тестовое окружение:

1. Создайте в Stronghold секрет типа kv2 по пути `demo-kv/myapp-secret` и поместите туда значения `DB_USER` и `DB_PASS`.

   * Включите и создайте Key-Value-хранилище:

     ```bash
     d8 stronghold secrets enable -path=demo-kv -version=2 kv
     ```

     Альтернативный вариант с использованием `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request POST \
       --data '{"type":"kv","options":{"version":"2"}}' \
       ${VAULT_ADDR}/v1/sys/mounts/demo-kv
     ```

   * Задайте имя пользователя и пароль базы данных в качестве значения секрета:

     ```bash
     d8 stronghold kv put demo-kv/myapp-secret DB_USER="username" DB_PASS="secret-password"
     ```

     Альтернативный вариант с использованием `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request PUT \
       --data '{"data":{"DB_USER":"username","DB_PASS":"secret-password"}}' \
       ${VAULT_ADDR}/v1/demo-kv/data/myapp-secret
     ```

   * Проверьте записанный секрет:

     ```bash
     d8 stronghold kv get demo-kv/myapp-secret
     ```

     Альтернативный вариант проверки с использованием `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       ${VAULT_ADDR}/v1/demo-kv/data/myapp-secret
     ```

1. При необходимости добавьте путь аутентификации ([`authPath`](/modules/secrets-store-integration/configuration.html#parameters-connection-authpath)) для аутентификации и авторизации в Stronghold с помощью Kubernetes API удалённого кластера.

   * По умолчанию в Stronghold включен и настроен под именем `kubernetes_local` метод аутентификации через Kubernetes API кластера, на котором запущен сам Stronghold. Если требуется настроить доступ через удалённые кластера, задайте путь (`authPath`) и включите аутентификацию и авторизацию в Stronghold с помощью Kubernetes API для каждого кластера:

     ```bash
     d8 stronghold auth enable -path=remote-kube-1 kubernetes
     ```

     Альтернативный вариант с использованием `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request POST \
       --data '{"type":"kubernetes"}' \
       ${VAULT_ADDR}/v1/sys/auth/remote-kube-1
     ```

   * Задайте адрес Kubernetes API для каждого кластера:

     ```bash
     d8 stronghold write auth/remote-kube-1/config \
       kubernetes_host="https://api.kube.my-deckhouse.com"
     ```

     Альтернативный вариант с использованием `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request PUT \
       --data '{"kubernetes_host":"https://api.kube.my-deckhouse.com"}' \
       ${VAULT_ADDR}/v1/auth/remote-kube-1/config
     ```

1. Создайте в Stronghold политику `myapp-ro-policy`, разрешающую чтение секретов по пути `demo-kv/data/myapp-secret`:

   ```bash
   d8 stronghold policy write myapp-ro-policy - <<EOF
   path "demo-kv/data/myapp-secret" {
     capabilities = ["read"]
   }
   EOF
   ```

   Альтернативный вариант с использованием `curl`:

   ```bash
   curl \
     --header "X-Vault-Token: ${VAULT_TOKEN}" \
     --request PUT \
     --data '{"policy":"path \"demo-kv/data/myapp-secret\" {\n capabilities = [\"read\"]\n}\n"}' \
     ${VAULT_ADDR}/v1/sys/policies/acl/myapp-ro-policy
   ```

1. Создайте в Stronghold роль для сервис-аккаунта `myapp-sa` в пространстве имён `myapp-namespace` и привяжите к ней созданную ранее политику.

   {{< alert level="danger">}}
   Помимо настроек со стороны Stronghold, необходимо настроить разрешения авторизации используемых ServiceAccount в кластере Kubernetes.
   Ознакомьтесь с необходимыми настройками [в следующем разделе](#как-разрешить-serviceaccount-авторизоваться-в-stronghold).
   {{< /alert >}}

   * Создайте роль, состоящую из названия пространства имён и политики. Свяжите её с ServiceAccount `myapp-sa` из пространства имён `myapp-namespace` и политикой `myapp-ro-policy`:

     {{< alert level="info">}}
     Рекомендованное значение TTL для токена Kubernetes составляет `10m`.
     {{< /alert >}}

     ```bash
     d8 stronghold write auth/kubernetes_local/role/myapp-role \
         bound_service_account_names=myapp-sa \
         bound_service_account_namespaces=myapp-namespace \
         policies=myapp-ro-policy \
         ttl=10m
     ```

     Альтернативный вариант с использованием `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request PUT \
       --data '{"bound_service_account_names":"myapp-sa","bound_service_account_namespaces":"myapp-namespace","policies":"myapp-ro-policy","ttl":"10m"}' \
       ${VAULT_ADDR}/v1/auth/kubernetes_local/role/myapp-role
     ```

   * Повторите то же самое для удалённых кластеров, указав другой путь аутентификации:

     ```bash
     d8 stronghold write auth/remote-kube-1/role/myapp-role \
         bound_service_account_names=myapp-sa \
         bound_service_account_namespaces=myapp-namespace \
         policies=myapp-ro-policy \
         ttl=10m
     ```

     Альтернативный вариант с использованием `curl`:

     ```bash
     curl \
       --header "X-Vault-Token: ${VAULT_TOKEN}" \
       --request PUT \
       --data '{"bound_service_account_names":"myapp-sa","bound_service_account_namespaces":"myapp-namespace","policies":"myapp-ro-policy","ttl":"10m"}' \
       ${VAULT_ADDR}/v1/auth/remote-kube-1/role/myapp-role
     ```

   Эти настройки позволяют любому поду из пространства имён `myapp-namespace` из обоих кластеров Kubernetes, который использует ServiceAccount `myapp-sa`, аутентифицироваться и авторизоваться в Stronghold для чтения секретов согласно политике `myapp-ro-policy`.

1. Создайте в кластере пространство имён `myapp-namespace`:

   ```bash
   d8 k create namespace myapp-namespace
   ```

1. Создайте в созданном пространстве имён сервис-аккаунт `myapp-sa`:

   ```bash
   d8 k -n myapp-namespace create serviceaccount myapp-sa
   ```

## Как разрешить ServiceAccount авторизоваться в Stronghold

Для авторизации в Stronghold под использует токен, сгенерированный для своего ServiceAccount. Для того чтобы Stronghold мог проверить валидность предоставляемых данных ServiceAccount, используемый сервисом Stronghold должен иметь разрешение на действия `get`, `list` и `watch`  для эндпоинтов `tokenreviews.authentication.k8s.io` и `subjectaccessreviews.authorization.k8s.io`. Для этого также можно использовать ClusterRole `system:auth-delegator`.

Stronghold может использовать различные авторизационные данные для осуществления запросов в API Kubernetes:

* Токен приложения, которое пытается авторизоваться в Stronghold — в этом случае для каждого сервиса, авторизующегося в Stronghold, требуется в используемом ServiceAccount иметь ClusterRole `system:auth-delegator` (либо права на API, представленные выше). За примером обратитесь к [документации Stronghold](https://deckhouse.ru/products/stronghold/documentation/user/auth/kubernetes.html#используйте-jwt-клиента-в-качестве-рецензента-jwt).

* Статичный токен созданного специально для Stronghold ServiceAccount, у которого имеются необходимые права — настройка Stronghold для такого случая подробно описана в [документации Stronghold](https://deckhouse.ru/products/stronghold/documentation/user/auth/kubernetes.html#использование-долгоживущих-токенов).

## Инжектирование переменных окружения

### Как работает инжектирование

При включении модуля в кластере появляется mutating-webhook, который при наличии у пода аннотации `secrets-store.deckhouse.io/role` изменяет манифест пода, добавляя туда инжектор. В измененном поде добавляется init-контейнер, который помещает из служебного образа собранный статически бинарный файл-инжектор в общую для всех контейнеров пода временную директорию. В остальных контейнерах оригинальные команды запуска заменяются на запуск файла-инжектора, который получает из Vault-совместимого хранилища необходимые данные, используя для подключения сервисный аккаунт приложения, помещает эти переменные в ENV процесса, после чего выполняет системный вызов `execve`, запуская оригинальную команду.

Если в манифесте пода у контейнера отсутствует команда запуска, то выполняется извлечение манифеста образа из registry, и команда извлекается из него.
Для получения манифеста из приватного хранилища образов используются заданные в манифесте пода учетные данные из `imagePullSecrets`.

Доступные аннотации, позволяющие изменять поведение инжектора:
<style>.annotations-table-style + .table-wrapper td:first-child{min-width: 317px}</style>
<div class="annotations-table-style"></div>

| Аннотация                                        | Значение по умолчанию |  Описание |
|--------------------------------------------------|-----------|-------------|
|`secrets-store.deckhouse.io/addr`                   | Из модуля | Адрес хранилища секретов в формате `https://stronghold.mycompany.tld:8200` |
|`secrets-store.deckhouse.io/tls-secret`             | Из модуля | Имя объекта Secret в Kubernetes, в котором есть ключ `ca.crt` со значением сертификата CA (Центра сертификации) в формате PEM |
|`secrets-store.deckhouse.io/tls-skip-verify`        | `false`     | Отключение проверки TLS-сертификата сервера |
|`secrets-store.deckhouse.io/auth-path`              | Из модуля | Путь, который следует использовать при аутентификации |
|`secrets-store.deckhouse.io/namespace`              | Из модуля | Пространство имен, которое будет использоваться для подключения к хранилищу |
|`secrets-store.deckhouse.io/role`                   |           | Роль, с которой будет выполнено подключение к хранилищу секретов |
|`secrets-store.deckhouse.io/env-from-path`          |           | Строка, содержащая список путей к секретам в хранилище через запятую, из которых будут извлечены все ключи и помещены в environment. Приоритет имеют ключи, которые находятся в списке ближе к концу |
|`secrets-store.deckhouse.io/ignore-missing-secrets` | `false`     | Запускает оригинальное приложение в случае ошибки получения секрета из хранилища |
|`secrets-store.deckhouse.io/client-timeout`         | `10s`       | Таймаут операции получения секретов |
|`secrets-store.deckhouse.io/mutate-probes`          | `false`     | Инжектирует переменные окружения в пробы |
|`secrets-store.deckhouse.io/log-level`              | `info`      | Уровень логирования |
|`secrets-store.deckhouse.io/enable-json-log`        | `false`     | Включает ведение логов в формате JSON |
|`secrets-store.deckhouse.io/skip-mutate-containers` |           | Список имен контейнеров через пробел, к которым не будет применяться инжектирование |

Используя инжектор, вы сможете задавать в манифестах пода вместо значений env-шаблоны, которые будут заменяться на этапе запуска контейнера на значения из хранилища.

{{< alert level="info">}}
Подключение переменных из ветки хранилища имеет более высокий приоритет, чем подключение явно заданных переменных из хранилища. Это значит, что при использовании одновременно аннотации `secrets-store.deckhouse.io/env-from-path` с путем до секрета, который содержит, к примеру, ключ `MY_SECRET`, и переменную окружения в манифесте с тем же именем:

```yaml
env:
  - name: MY_SECRET
    value: secrets-store:demo-kv/data/myapp-secret#password
```

в переменную окружения `MY_SECRET` внутри контейнера запишется значение секрета из **аннотации**.
{{< /alert >}}

Пример извлечения из Vault-совместимого хранилища ключа `DB_PASS` из kv2-секрета по адресу `demo-kv/myapp-secret`:

```yaml
env:
  - name: PASSWORD
    value: secrets-store:demo-kv/data/myapp-secret#DB_PASS
```

Пример извлечения из Vault-совместимого хранилища ключа `DB_PASS` версии `4` из kv2-секрета по адресу `demo-kv/myapp-secret`:

```yaml
env:
  - name: PASSWORD
    value: secrets-store:demo-kv/data/myapp-secret#DB_PASS#4
```

Шаблон может также находиться в ConfigMap или в Secret и быть подключен с помощью `envFrom`:

```yaml
envFrom:
  - secretRef:
      name: app-secret-env
  - configMapRef:
      name: app-env
```

Инжектирование реальных секретов из Vault-совместимого хранилища выполнится только на этапе запуска приложения.
В Secret и ConfigMap будут находиться шаблоны.

### Подключение переменных из ветки хранилища (всех ключей одного секрета)

1. Создайте под с названием `myapp1`, который подключит все переменные из хранилища по пути `demo-kv/data/myapp-secret`:

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

1. Примените созданный манифест:

   ```bash
   d8 k create --filename myapp1.yaml
   ```

1. Проверьте логи пода после его запуска.
   В результате должны быть выведены все переменные из `demo-kv/data/myapp-secret`:

   ```bash
   d8 k -n myapp-namespace logs myapp1
   ```

1. Удалите под:

   ```bash
   d8 k -n myapp-namespace delete pod myapp1 --force
   ```

### Подключение явно заданных переменных из хранилища

1. Создайте тестовый под с названием `myapp2`, который подключит требуемые переменные из хранилища по шаблону:

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

1. Примените созданную конфигурацию:

   ```bash
   d8 k create --filename myapp2.yaml
   ```

1. Проверьте логи пода после его запуска.
   В результате должны быть выведены переменные из `demo-kv/data/myapp-secret`:

   ```bash
   d8 k -n myapp-namespace logs myapp2
   ```

1. Удалите под:

   ```bash
   d8 k -n myapp-namespace delete pod myapp2 --force
   ```

## Монтирование секрета из хранилища в качестве файла в контейнер

Для доставки секретов в приложение нужно использовать кастомный [ресурс SecretsStoreImport](/modules/secrets-store-integration/cr.html#secretsstoreimport).

В этом примере используется сервисный аккаунт `myapp-sa` и пространство имен `myapp-namespace`, созданные на этапе [подготовки тестового окружения](#подготовка-тестового-окружения).

1. Создайте в кластере кастомный ресурс SecretsStoreImport с названием `myapp-ssi`:

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

1. Создайте в кластере тестовый под с названием `myapp3`, который подключит требуемые переменные из хранилища в виде файла:

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

   После применения этих ресурсов будет создан под, внутри которого запустится контейнер с названием `backend`. В файловой системе этого контейнера будет каталог `/mnt/secrets` с примонтированным к нему томом `secrets`. Внутри этого каталога будет лежать файл `db-password` с паролем от базы данных (`DB_PASS`) из Key-Value-хранилища Stronghold.

1. Проверьте логи пода после его запуска (должно выводиться содержимое файла `/mnt/secrets/db-password`):

   ```bash
   d8 k -n myapp-namespace logs myapp3
   ```

1. Удалите под:

   ```bash
   d8 k -n myapp-namespace delete pod myapp3 --force
   ```

### Доставка бинарных файлов в контейнер

В некоторых ситуациях вам может понадобиться доставить бинарный файл в контейнер.
Это может быть JKS-контейнер с ключами или keytab для Kerberos-аутентификации.
В этом случае вы можете закодировать бинарный файл через Base64 и поместить его в хранилище секретов, а при извлечении
CSI-драйвер раскодирует ваши данные и поместит в контейнер бинарный файл. Для этого нужно установить параметр
`decodeBase64` в `true` для соответствующего файла.
Если декодирование произвести не получится (например, если в хранилище находится невалидный Base64), контейнер не будет создан.

Пример:

1. Закодируйте файл через Base64 и поместите его в хранилище:

   ```bash
   d8 stronghold kv put demo-kv/myapp-secret keytab=$(cat /path/to/keytab_file | base64 -w0)
   ```

1. Создайте манифест [SecretsStoreImport](/modules/secrets-store-integration/cr.html#secretsstoreimport), указав параметр для раскодирования файла:

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

1. В контейнере будет создан бинарный файл с именем `keytab`.

### Функция авторотации

Функция авторотации секретов в модуле `secret-store-integration` включена по умолчанию. Каждые две минуты модуль опрашивает Stronghold и синхронизирует секреты в примонтированном файле в случае его изменения.

Существует два способа отслеживания изменений файла с секретом в поде:

* Следить за временем изменения примонтированного файла, реагируя на его изменение.
* Использовать inotify API, который предоставляет механизм подписки на события файловой системы. Inotify является частью ядра Linux. При обнаружении изменений существует множество вариантов реагирования в зависимости от используемой архитектуры приложения и используемого языка программирования. Самый простой — заставить Kubernetes перезапустить под, перестав отвечать на liveness-пробу.

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

Пример использования inotify в приложении на Go с применением пакета inotify:

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

#### Ограничения при обновлении секретов

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

## CLI-утилита d8 для команд Stronghold

Deckhouse CLI (`d8`) — это универсальный инструмент, необходимый для выполнения команд вида `d8 stronghold` в терминале.

Чтобы установить `d8`, воспользуйтесь одним из способов, описанных в [документации CLI-утилиты](/products/kubernetes-platform/documentation/v1/cli/d8/#установка-исполняемого-файла).
