# godsl

**godsl** — это CLI-инструмент для транспиляции файлов `.godsl` в Go-код с поддержкой многопоточной обработки. Поддерживает Linux, macOS и Windows. Главная задача инструмента улучшить взаимодействие с языком программирования Golang. Первая версия инструмента улучшает работу с ошибками, заменяя привычный `err!=nil` на `try/catch` блок, который активно применяется в других популярных языках программирования.

## ⚙️ Возможности

- 🚀 Транспиляция `.godsl` файлов в `.go` с сохранением структуры проекта
- 🧵 Параллельная обработка файлов с использованием всех ядер процессора
- 📦 Команда `generate` обрабатывает как один файл, так и рекурсивно всю директорию
- 📌 Поддержка кросс-платформенности: **Linux**, **MacOS**, **Windows**

## 📦 Команды

- `version` — показывает текущую версию
- `update` — обновление CLI до последней версии ( пока реализовано только для linux/macos )
- `generate` — запуск транспиляции
- `build` — транспиляция + `go build` (сборка транспилированного проекта из папки `build`)
- `run` — транспиляция + `go run` (запуск транспилированного проекта из папки `build`)

## 🚀 Установка

Для Macos/Linux:

```bash
bash <(curl -s https://raw.githubusercontent.com/sviridovkonstantin42/godsl/main/install.sh)
```

Windows: </br>
https://github.com/sviridovkonstantin42/godsl/releases

## 🧪 Пример использования

1. Напиши проект на godsl

```go
package main

import "log"
import "errors"

func main(){
    try{
        //@errcheck
        a, err:=functionWithError()
        log.Println("computed a...")

        //@errcheck
        b, err:=functionWithError()
        log.Println("computed b...")

        //@errcheck
        c, err:=functionWithError()
        log.Println("computed c...")

        log.Println(a,b,c)
    } catch {
        log.Println(err)
    }
}

func functionWithError() (string, error){
    return "", errors.New("У вас ошибка!")
}
```

2. Выполни команду:

```bash
godsl generate <путь проекта>
```

Или сразу собрать/запустить:

```bash
godsl build <путь проекта>
godsl run <путь проекта>
```

3. Транспилированный проект лежит в текущей папке с названием `build`. Результат:

```go
package main

import "log"
import "errors"

func main() {
	a, err := functionWithError()
	if err != nil {
		log.Println(err)
	}
	log.Println("computed a...")

	b, err := functionWithError()
	if err != nil {
		log.Println(err)
	}
	log.Println("computed b...")

	c, err := functionWithError()
	if err != nil {
		log.Println(err)
	}
	log.Println("computed c...")

	log.Println(a, b, c)
}

func functionWithError() (string, error) { return "", errors.New("У вас ошибка!") }
```
