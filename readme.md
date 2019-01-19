# Сервис-интерфейс для работы с базой данных.

## Эндпоинты

- GET `/` - список таблиц.
- GET `/{tableName}` - все записи из таблицы tableName.
- GET `/{tableName}/{id}` - конкретная запись (id) из таблицы tableName.
- PUT `/{tableName}/` - Создание записи в таблице tableName.
- POST `/{tableName}/{id}` - Обновление записи (id) в таблице tableName.
- DELETE `/{tableName}/{id}` - удаление записи (id) из таблицы tableName.

Для запуска сервиса выполните:

`docker-compose up db-explorer`

По ходу запуска выполнятся тесты, и сервис будет доступен по адресу [localhost:8084](http://localhost:8084/)

___

Спасибо [xenmayer](https://github.com/xenmayer) за помощь в развертывании приложения :)