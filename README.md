# go-musthave-diploma-tpl

# Запуск проекта
В корневой папке проекта создайте файл .env и заполните его в соответствии с .env-example.

## Команды для взаимодействия с проектом
- Список доступных команд
```
make help
```

- Запуск проекта - всех сервисов через docker compose (accrual-system, gophermart-db, gophermart-app)
```
make run
```
- Запуск accrual-system:
1. на MacOS
```
make as-darwin-arm64
```
или
```
./cmd/accrual/accrual_darwin-arm64
```
2. на Linux
```
make as-linux
```
или
```
./cmd/accrual/accrual_linux_amd64
```

При запуске через ```./cmd/accrual``` доступен флаг ```-a``` - адрес сервиса. Адрес по дефолту -  ```localhost:8080```

- Запуск gophermart-app
- в режиме разработчика - stdout DUBUG логов
```
make dev
```

## Запуск тестов
- Создайте тестовую базу данных
```
make up-test-db
```

- Запустите тесты
```
make test
```

## Дополнительный команды
- Завершение работы проекта (всех сервисов)
```
make down
```

- Завершение работы тестовой базы данных
```
make down-test-db
```
