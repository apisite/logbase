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
* оптимальная для анализа организация журналов (см. схему БД для nginx)
* анализ журналов средствами БД (будет использован procapi)
* простота добавления новых структур и форматов журналов

## Статус проекта

Готова пилотная версия загрузки журналов nginx

## Использование

### Загрузка данных Nginx

```
curl -X POST -o send.log -H "Content-Type: application/octet-stream" \
 -H "Auth: $KEY" -H "File: $FILE" --data-binary '@'$DIR/$FILE $HOST/upload/nginx && cat send.log 
```

## TODO

* [ ] nginx: API
* [ ] nginx: Frontend
* [ ] postgresql logs
* [ ] journald logs
* [ ] pgmig
* [ ] single-binary distribution (inc windows)

## License

The MIT License (MIT), see [LICENSE](LICENSE).

Copyright (c) 2020 Aleksei Kovrizhkin <lekovr+logbase@gmail.com>
