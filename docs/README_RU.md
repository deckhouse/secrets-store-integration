---
title: "Модуль secrets-store-integration"
description: "Модуль secrets-store-integration реализует интеграцию хранилища секретов и приложений в Kubernetes-кластерах"
---

Модуль secrets-store-integration реализует доставку секретов для приложения в Kubernetes-кластерах
путем подключения секретов, ключей и сертификатов, хранящихся во внешних хранилищах секретов.

Секреты монтируются в поды в виде тома с использованием реализации драйвера CSI.
Хранилища секретов должны быть совместимы с API-интерфейсом HashiCorp Vault.

## Доставка секретов в приложения

Доставить секреты в приложение из vault-совместимого хранилища можно несколькими способами:

1. Пользовательское приложение само обращается в хранилище.

   > Это наиболее безопасный вариант, но требует модификации приложений.

1. В хранилище обращается приложение-прослойка, а ваше приложение получает доступ к секретам из файлов, созданных в контейнере.

   > Если нет возможности модифицировать приложение, используйте этот вариант. Он проще в реализации, но менее безопасный, так как секретные данные хранятся в файлах в контейнере.

1. В хранилище обращается приложение-прослойка, и пользовательское приложение получает доступ к секретам из переменных среды.

   > Если нет возможности читать из файлов, можно использовать этот вариант, но он небезопасен. При таком подходе секретные данные хранятся в Kubernetes (а так же в etcd) и потенциально могут быть прочитаны на любом узле кластера.

<table>
<thead>
<tr>
<th>Вариант доставки</th>
<th>Потребление ресурсов</th>
<th>Как приложение получает данные?</th>
<th>Где хранится в Kubernetes?</th>
<th>Статус</th>
</tr>
</thead>
<tbody>
<tr>
<td><a style="color: ##0066FF;" href="#вариант-1-получение-секретов-самим-приложением">Приложение</a></td>
<td>Не меняется</td>
<td>Напрямую из хранилища секретов</td>
<td>Не хранится</td>
<td>Реализовано</td>
</tr>
<tr>
<td><a style="color: ##0066FF;" href="#механизм-csi">Механизм CSI</a></td>
<td>Два пода на каждую ноду (daemonset)</td>
<td><ul><li>Из дискового тома (как файл)</li><li>Из переменной окружения</li></ul></td>
<td>Не хранится</td>
<td>Реализовано</td>
</tr>
<tr>
<td><a style="color: ##0066FF;" href="#вариант-3-инъекция-entrypoint">Инъекция entrypoint</a></td>
<td>Один под на каждую ноду (daemonset)</td>
<td>Секреты доставляются из хранилища в момент запуска приложения в виде переменных окружения</td>
<td>Не хранится</td>
<td>В процессе реализации</td>
</tr>
<tr>
<td><a style="color: ##0066FF;" href="#вариант-4-доставка-секретов-через-механизмы-kubernetes">Секреты Kubernetes</a></td>
<td>Одно приложение на кластер (deployment)</td>
<td><ul><li>Из дискового тома (как файл)</li><li>Из переменной окружения</li></ul></td>
<td>Хранится в Secrets</td>
<td>Планируется</td>
</tr>
<tr>
<td><a style="color: #A9A9A9; font-style: italic;" href="#справочно-инжектор-vault-agent">Инжектор vault-agent</a></td>
<td style="color: #A9A9A9; font-style: italic;">По одному агенту на каждый под (sidecar)</td>
<td style="color: #A9A9A9; font-style: italic;">Из дискового тома (как файл)</td>
<td style="color: #A9A9A9; font-style: italic;">Не хранится</td>
<td style="color: #A9A9A9; font-style: italic;"><sup><b>*</b></sup>Не будет реализовано</td>
</tr>
</tbody>
</table>

<i><sup>*</sup>Поддержка отсутствует и не планируется, поскольку этот вариант не имеет преимуществ перед использованием механизма CSI.</i>

### Вариант №1: Получение секретов самим приложением

> *Статус:* наиболее безопасный вариант. Рекомендован к использованию, если есть возможность модификации приложений.

Приложение обращается к API Stronghold и запрашивает необходимый секрет по HTTPS-протоколу с использованием токена авторизации (токен из SA).

Плюсы:
- Секрет, полученный приложением, нигде не хранится, кроме как в самом приложении, нет опасности что он будет скомпрометирован в процессе передачи.

Минусы:

- Требует доработки приложения для возможности работы со Stronghold.
- Требует повторения реализации доступа к секретам в каждом приложении. В случае обновления библиотеки требует пересборки всех приложений.
- Приложение должно поддерживать TLS и проверку сертификатов.
- Нет кэширования. При перезапуске приложения нужно повторно запросить секрет напрямую из хранилища.

### Вариант №2: Доставка секретов через файлы

#### Механизм CSI

> *Статус:* безопасный вариант. Рекомендован к использованию, если отсутствует возможность модификация приложений.

При создании подов, запрашивающих тома CSI, драйвер хранилища секретов CSI отправляет запрос к Vault CSI. Затем Vault CSI использует указанный SecretProviderClass и ServiceAccount пода для получения секретов из хранилища и монтирования их в том пода.

#### Инъекция переменных окружений:

Если нет возможности изменить код приложения, то можно реализовать безопасную инъекцию секрета в качестве переменной окружения для приложения.

Для этого нужно:
- прочитать все файлы, примонтированные CSI в контейнер;
- определить переменные окружения с именами, соответствующими именам файлов, и значениями, соответствующим содержимому файлов.
- запустить оригинальное приложение.

Пример на Bash:

```bash
bash -c "for file in $(ls /mnt/secrets); do export  $file=$(cat /mnt/secrets/$file); done ; exec my_original_file_to_startup"
```

Плюсы:

- Всего два контейнера с прогнозируемыми ресурсами на каждом узле для обслуживания системы доставки секретов в приложения;
- Создание ресурсов _SecretsStore/SecretProviderClass_ уменьшает количество повторяемого кода по сравнению с другими вариантами реализации vault agent;
- При необходимости есть возможность создавать копию секрета из хранилища в виде секрета Kubernetes.
- Секрет извлекается из хранилища драйвером CSI на этапе создания контейнера. Это означает, что запуск подов заблокируется до тех пор, пока секреты не будут прочитаны из хранилища и записаны в том.

### Вариант №3: Инъекция entrypoint

#### Доставка переменных окружения через инъекцию entrypoint в контейнер

> *Статус:* безопасный вариант. В процессе реализации.

Переменные доставляются из хранилища в момент запуска приложения и находятся только в памяти. В момент первого этапа реализации метода переменные будут доставляться через entrypoint, проброшенный в контейнер. В дальнейшем планируется интеграция функционала доставки секретов в containerd.

### Вариант №4: Доставка секретов через механизмы Kubernetes

> *Статус:* небезопасный вариант, не рекомендован к использованию. Поддержка отсутствует, но планируется в будущем.

Этот метод интеграции, который реализует оператор секретов Kubernetes с набором CRD, отвечающих за синхронизацию секретов из Vault в секреты Kubernetes.

Минусы:

- Секрет находится и в хранилище секретов, и в секрете Kubernetes (доступном через API Kubernetes). Секрет также хранится в etcd и потенциально может быть считан на любом узле кластера или извлечён из резервной копии etcd. Нет возможности не хранить данные в секретах Kubernetes.

Плюсы:

- Классический способ передачи секрета в приложение через переменные окружения — достаточно подключить секрет Kubernetes.

### Справочно: Инжектор vault-agent

> *Статус:* не имеет плюсов в сравнении с механизмом CSI. Поддержка отсутствует и не планируется, поскольку этот вариант не имеет преимуществ перед использованием механизма CSI.

При создании пода происходит мутация, которая добавляет контейнер с vault-agent. Агент обращается к хранилищу секретов, извлекает их, и помещает в общий том на диске, к которому может обратиться приложение.

Минусы:

- Для каждого пода нужен sidecar-контейнер, который так или иначе потребляет ресурсы.

  Например, возьмем кластер в котором 50 приложений, и каждое приложение имеет от 3 до 15 реплик. Так как для каждого sidecar-контейнера с агентом нужно выделить ресурсы CPU и памяти, то даже при незначительных ресурсах для sidecar-контейнера в размере 0.05 CPU и 100 MiB памяти, на все приложения в сумме получаются десятки ядер CPU и десятки ГБ памяти.
- Так как сбор метрик осуществляется с каждого контейнера, то с таким подходом мы получим в два раза больше метрик только по контейнерам.