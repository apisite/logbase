<p align="center">
  <a href="README.md#apisitelogbase">English</a> |
  <span>Pусский</span>
</p>

---

# apisite/logbase
> Сервис для размещения в БД Postgresql журнальных файлов

[![GoDoc][gd1]][gd2]
 [![GoCard][gc1]][gc2]
 [![GitHub Release][gr1]][gr2]
 [![GitHub code size in bytes][sz]]()
 [![GitHub license][gl1]][gl2]

[gd1]: https://godoc.org/github.com/apisite/logbase?status.svg
[gd2]: https://godoc.org/github.com/apisite/logbase
[gc1]: https://goreportcard.com/badge/github.com/apisite/logbase
[gc2]: https://goreportcard.com/report/github.com/apisite/logbase
[gr1]: https://img.shields.io/github/release-pre/apisite/logbase.svg
[gr2]: https://github.com/apisite/logbase/releases
[sz]: https://img.shields.io/github/languages/code-size/apisite/logbase.svg
[gl1]: https://img.shields.io/github/license/apisite/logbase.svg
[gl2]: https://github.com/apisite/logbase/blob/master/LICENSE

<p align="center">
<a target="_blank" rel="noopener noreferrer" href="nginx.png"><img src="nginx.png" title="Схема БД для Nginx" style="max-width:100%;"></a>
</p>

## Назначение

Экспериментальный проект по созданию сервиса анализа журнальных файлов со следующими свойствами

* загрузка журналов штатными средствами ОС (в текущей версии используется curl)
* минимальный размер дистрибутива (один бинарный файл)
* загрузка файла в несколько потоков
* дозагрузка данных из обновленного файла
* поддержка архивированных логов (deflate/gzip/bz2)
* поддержка кириллицы в адресах и аргументах (в т.ч. utf8 / cp1251 в зависимости от префиса)
* отсутствие дублей при хранении строк (адрес, аргументы, агент, реферер)
* обновление статистики по файлу в процессе загрузки

## Статус проекта

Готова пилотная версия загрузки журналов nginx

## Использование

### Конфигурация

Размещается в БД (таблица logs.config). Пример:
```
select id,key,type_id,jsonb_pretty(data) as data from logs.config;
-[ RECORD 1 ]+--------------------------------------------
id           | 1
key          | f091d3c43b3189c8c4cacde8cf47c00f
type_id      | 1
data         |  { "host": "^https?://.*tender\\.pro/",
                  "channels": 4,
                  "utf8_prefix": "/api/",
                  "skip": "\\.(js|gif|png|css|ico|jpg|eot)$",
                  "format": "$remote_addr $user1 $user2 [$time_local] \"$request\" \"$status\" $size \"$referer\"
                            \"$user_agent\" \"$t_size\" $fresp $fload $pipe $request_length $request_id"
                }
```

где

* **key** - ключ авторизации для загрузки данных
* **type_id** - тип настроек (1 - nginx)

настройки nginx:

* **host** - regexp для $referer, при совпадении с которым адреса будут считаться внутренними и добавляться в список своих
* **channels** - сколько потоков использовать при загрузке в БД
* **utf8_prefix** - аргументы адресов не с таким префиксом декодируются из cp1251 
* **skip** - regexp для $request, совпадающие строки не грузятся в БД
* **format** - формат журнала, строка из nginx.conf

### Загрузка данных Nginx

```
curl -X POST -o send.log -H "Content-Type: application/octet-stream" -H "Content-Encoding: bz2" \
 -H "Auth: $KEY" -H "File: $FILE" --data-binary '@'$DIR/$FILE $HOST/upload/nginx && cat send.log 
```
запрос возвращает json:
```
{"ID":7,"Bytes":4992,"File":"log10ki.bz2","Type":"nginx"}
```

где

* **ID** - присвоенный файлу file_id (см logs.file, logs.request_data)
* **Type** - тип обработчика
* **Bytes** - размер полученного файла
* **File** - имя файла

### Алгоритм загрузки файла

* по URL (/nginx) пределяем тип файла
* по заголовку Auth - ключ, по которому извлекаем config
* по заголовку `Content-Encoding` определяем архиватор
* регистрируем начало загрузки файла в БД и получаем его `_file_id`
* по первой корректной строке файла определяем его timestamp, по которому из БД получаем `_stamp_id`
* далее для каждой строки файла
  * получаем map с ключами из format (если строка не по формату - пишем в лог и пропускаем)
  * request меняем на `method`, `proto` и unescaped `url`, `args`
  * если префикс url не совпадает с utf8_prefix - декодируем аргументы из 1251 (TODO: no_utf_enc)
  * разбираем аргументы в map и конвертируем в json
  * добавляем  `_stamp_id`, `_file_id`, `_line_num`
  * формируем список аргументов (если arg_prefix = '-', как массив, иначе - map с добавлением префикса в ключ)
  * вызываем ф-ю request_add

## TODO

* [ ] geoip
* [ ] Grafana intergation
* [ ] metrics
* [ ] postgresql logs
* [ ] journald logs
* [ ] pgmig
* [ ] single-binary distribution (inc windows)

## License

The MIT License (MIT), see [LICENSE](LICENSE).

Copyright (c) 2020 Aleksei Kovrizhkin <lekovr+logbase@gmail.com>
