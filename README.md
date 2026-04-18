# godsl

**godsl** — транспилятор, расширяющий язык Go конструкциями обработки ошибок, привычными из Java, Python и C#. Файлы `.godsl` компилируются в стандартный `.go` код, полностью совместимый с обычным тулчейном Go.

Поддерживает Linux, macOS и Windows.

## Зачем это нужно?

Идиоматичный Go-код для обработки ошибок многословен:

```go
result, err := doSomething()
if err != nil {
    return err
}
more, err := doMore(result)
if err != nil {
    return err
}
```

godsl позволяет написать то же самое компактнее:

```godsl
result := doSomething()?
more := doMore(result)?
```

---

## Установка

Для macOS/Linux:

```bash
bash <(curl -s https://raw.githubusercontent.com/sviridovkonstantin42/godsl/main/install.sh)
```

Windows: https://github.com/sviridovkonstantin42/godsl/releases

Или через `go install`:

```bash
go install github.com/sviridovkonstantin42/godsl@latest
```

---

## Быстрый старт

```bash
# Создать новый проект
godsl init myapp
cd myapp

# Запустить
godsl run

# Собрать бинарник
godsl build

# Форматировать исходники
godsl fmt ./...
```

---

## Команды CLI

| Команда              | Описание                                          |
| -------------------- | ------------------------------------------------- |
| `godsl init [имя]`   | Инициализировать новый godsl-проект               |
| `godsl generate`     | Транспилировать `.godsl` → `.go` в папку `build/` |
| `godsl run [путь]`   | Транспилировать и запустить через `go run`        |
| `godsl build [путь]` | Транспилировать и собрать через `go build`        |
| `godsl test [флаги]` | Транспилировать и запустить тесты через `go test` |
| `godsl fmt [путь]`   | Форматировать `.godsl` файлы                      |
| `godsl version`      | Показать версию                                   |
| `godsl update`       | Обновить CLI до последней версии                  |

### Флаги

```bash
godsl generate --clean         # Полная пересборка (без инкремента)
godsl generate --watch         # Перегенерировать при изменении файлов
godsl run --watch              # Перезапускать при изменении файлов
godsl fmt --check ./...        # Проверить форматирование без записи
godsl fmt --list  ./...        # Вывести список неотформатированных файлов
godsl test -- -v -run TestFoo  # Передать флаги напрямую в go test
```

### Инкрементальная сборка

`godsl generate` использует SHA-256-кэш (`.godslcache.json` в папке `build/`): повторная транспиляция выполняется только для изменившихся файлов.

---

## Возможности языка

### 1. `throw` — явное бросание ошибки

`throw <expr>` транспилируется в `return <expr>`.

**Источник:**

```godsl
func validate(s string) error {
    if s == "" {
        throw errors.New("validation failed: empty string")
    }
    if len(s) < 3 {
        throw fmt.Errorf("validation failed: too short (%d chars)", len(s))
    }
    return nil
}
```

**Результат транспиляции:**

```go
func validate(s string) error {
    if s == "" {
        return errors.New("validation failed: empty string")
    }
    if len(s) < 3 {
        return fmt.Errorf("validation failed: too short (%d chars)", len(s))
    }
    return nil
}
```

---

### 2. `?` — оператор автоматического возврата ошибки

Суффикс `?` после выражения автоматически проверяет ошибку и возвращает её из текущей функции.

**Форма с присваиванием** — `a := f()?`:

```godsl
func process() error {
    a := readFile("input.txt")?
    b := parseData(a)?
    return nil
}
```

**Результат транспиляции:**

```go
func process() error {
    a, err := readFile("input.txt")
    if err != nil {
        return err
    }
    b, err := parseData(a)
    if err != nil {
        return err
    }
    return nil
}
```

**Форма с выражением** — `f()?`:

```godsl
func run() error {
    validate("input")?
    return nil
}
```

**Результат транспиляции:**

```go
func run() error {
    if err := validate("input"); err != nil {
        return err
    }
    return nil
}
```

---

### 3. `must` — паника при ошибке

`must` аналогичен `?`, но вместо `return err` генерирует `panic(err)`. Используется для критических инициализаций, которые не могут завершиться ошибкой в рабочей программе.

**Форма с присваиванием** — `x := must f()`:

```godsl
func main() {
    db := must openDB("localhost:5432/myapp")
    fmt.Println("Connected:", db)
}
```

**Результат транспиляции:**

```go
func main() {
    db, err := openDB("localhost:5432/myapp")
    if err != nil {
        panic(err)
    }
    fmt.Println("Connected:", db)
}
```

**Форма с выражением** — `must f()`:

```godsl
must setup()
```

**Результат транспиляции:**

```go
if err := setup(); err != nil {
    panic(err)
}
```

---

### 4. `try / catch` — блок обработки ошибок

#### 4.1 Аннотация `@errcheck`

Внутри блока `try` аннотация `@errcheck` помечает следующий оператор для автоматической проверки ошибки. После него вставляется `if err != nil { <тело catch> }`.

Аннотацию можно записать тремя способами:

```godsl
@errcheck
a, err := f()       // отдельной строкой перед оператором

//@errcheck
b, err := g()       // как комментарий перед оператором

c, err := h()  //@errcheck  // inline-комментарий в строке оператора
```

#### 4.2 `catch` без типа (catch-all)

```godsl
func foo() error {
    try {
        @errcheck
        result, err := fetchData()
        fmt.Println("got:", result)
    } catch {
        return err
    }
    return nil
}
```

**Результат транспиляции:**

```go
func foo() error {
    result, err := fetchData()
    if err != nil {
        return err
    }
    fmt.Println("got:", result)
    return nil
}
```

#### 4.3 `catch` с типом — перехват конкретного типа ошибки

```godsl
try {
    @errcheck
    res, err := fetchResource(0)
    fmt.Println("Got:", res)
} catch(NotFoundError) {
    fmt.Println("Caught NotFoundError:", err)
} catch {
    fmt.Println("Unknown error:", err)
}
```

**Результат транспиляции:**

```go
res, err := fetchResource(0)
if err != nil {
    if _, ok := err.(NotFoundError); ok {
        fmt.Println("Caught NotFoundError:", err)
    } else {
        fmt.Println("Unknown error:", err)
    }
}
fmt.Println("Got:", res)
```

#### 4.4 `catch` с переменной — привязка к переменной

```godsl
try {
    @errcheck
    res, err := fetchResource(-1)
    fmt.Println("Got:", res)
} catch(e PermissionError) {
    fmt.Printf("Access denied for '%s'\n", e.User)
}
```

**Результат транспиляции:**

```go
res, err := fetchResource(-1)
if err != nil {
    if e, ok := err.(PermissionError); ok {
        fmt.Printf("Access denied for '%s'\n", e.User)
    }
}
fmt.Println("Got:", res)
```

#### 4.5 `catch` с несколькими типами через `|`

```godsl
try {
    @errcheck
    res, err := riskyOp("timeout")
    fmt.Println("result:", res)
} catch(ErrTimeout | ErrNetwork) {
    fmt.Println("transient error:", err)
} catch(e ErrDisk) {
    fmt.Printf("fatal disk error at '%s'\n", e.Path)
}
```

**Результат транспиляции:**

```go
res, err := riskyOp("timeout")
if err != nil {
    if func() bool {
        if _, ok := err.(ErrTimeout); ok {
            return true
        }
        if _, ok := err.(ErrNetwork); ok {
            return true
        }
        return false
    }() {
        fmt.Println("transient error:", err)
    } else if e, ok := err.(ErrDisk); ok {
        fmt.Printf("fatal disk error at '%s'\n", e.Path)
    }
}
fmt.Println("result:", res)
```

---

### 5. `try / catch / finally` — блок с гарантированной очисткой

`finally` выполняется всегда — и при ошибке, и при успехе.

```godsl
func runQuery(conn *Connection) {
    try {
        @errcheck
        result, err := query(conn)
        fmt.Println("Query result:", result)
    } catch {
        fmt.Println("Query failed:", err)
    } finally {
        conn.Close()
    }
}
```

**Результат транспиляции (catch без `return`):**

```go
func runQuery(conn *Connection) {
    func() bool {
        result, err := query(conn)
        if err != nil {
            fmt.Println("Query failed:", err)
            return false
        }
        fmt.Println("Query result:", result)
        return false
    }()
    conn.Close()
}
```

**Результат транспиляции (catch с `return`):**

```go
func runQuery(conn *Connection) {
    _godslRet := func() bool {
        result, err := query(conn)
        if err != nil {
            fmt.Println("Query failed:", err)
            return true
        }
        fmt.Println("Query result:", result)
        return false
    }()
    conn.Close()
    if _godslRet {
        return
    }
}
```

---

### 6. Тернарный оператор `? :`

`cond ? then : else` транспилируется в анонимную функцию (IIFE). Транспилятор **автоматически выводит тип** возвращаемого значения — точный тип вместо `any`.

#### Правила вывода типа (приоритет по убыванию)

| Условие                                               | Выведенный тип                                            |
| ----------------------------------------------------- | --------------------------------------------------------- |
| Обе ветки — литералы одного вида                      | тип литерала (`string`, `int`, `float64`, `rune`, `bool`) |
| Одна ветка — литерал, другая — произвольное выражение | тип литерала                                              |
| Ветки не содержат литералов                           | возвращаемый тип текущей функции                          |
| Тип определить невозможно                             | `any`                                                     |

#### Примеры с выводом типов

```godsl
// Обе ветки — строковые литералы → string
result := x > 0 ? "positive" : "non-positive"

// Функция возвращает int → int (ветки не литералы)
func max(a, b int) int {
    return a > b ? a : b
}

// Одна ветка — литерал → string
func label(x int, custom string) string {
    return x > 0 ? "default" : custom
}
```

**Результат транспиляции:**

```go
// string — оба строковых литерала
result := func() string {
    if x > 0 {
        return "positive"
    }
    return "non-positive"
}()

// int — из возвращаемого типа функции
func max(a, b int) int {
    return func() int {
        if a > b {
            return a
        }
        return b
    }()
}

// string — из ветки-литерала
func label(x int, custom string) string {
    return func() string {
        if x > 0 {
            return "default"
        }
        return custom
    }()
}
```

Вложенный тернарный оператор:

```godsl
func classify(x int) string {
    return x > 0 ? "positive" : x < 0 ? "negative" : "zero"
}
```

**Результат** (тип `string` — из возвращаемого типа функции и строковых литералов):

```go
func classify(x int) string {
    return func() string {
        if x > 0 {
            return "positive"
        }
        return func() string {
            if x < 0 {
                return "negative"
            }
            return "zero"
        }()
    }()
}
```

---

## Примеры

В папке [`examples/`](examples/) находятся подпроекты, каждый из которых демонстрирует отдельную возможность языка.

### Запуск примеров

```bash
godsl run ./examples/01_throw
godsl run ./examples/02_question_operator
godsl run ./examples/03_must
godsl run ./examples/04_try_catch
godsl run ./examples/05_try_catch_finally
godsl run ./examples/06_multi_type_catch
godsl run ./examples/07_ternary
```

---

### [01_throw](examples/01_throw/) — оператор `throw`

**Источник** [`examples/01_throw/main.godsl`](examples/01_throw/main.godsl):

```godsl
func validate(s string) error {
    if s == "" {
        throw errors.New("validation failed: empty string")
    }
    if len(s) < 3 {
        throw fmt.Errorf("validation failed: too short (%d chars)", len(s))
    }
    return nil
}
```

**Результат транспиляции** `build/examples/01_throw/main.go`:

```go
func validate(s string) error {
    if s == "" {
        return errors.New("validation failed: empty string")
    }
    if len(s) < 3 {
        return fmt.Errorf("validation failed: too short (%d chars)", len(s))
    }
    return nil
}
```

**Вывод программы:**

```
Error: validation failed: empty string
Error: validation failed: too short (2 chars)
OK: hello is valid
```

---

### [02_question_operator](examples/02_question_operator/) — оператор `?`

**Источник** [`examples/02_question_operator/main.godsl`](examples/02_question_operator/main.godsl):

```godsl
func processWithQuestion(aStr, bStr string) (int, error) {
    a := readNumber(aStr)?
    b := readNumber(bStr)?
    result := divide(a, b)?
    return result, nil
}
```

**Результат транспиляции** `build/examples/02_question_operator/main.go`:

```go
func processWithQuestion(aStr, bStr string) (int, error) {
    a, err := readNumber(aStr)
    if err != nil {
        return 0, err
    }
    b, err := readNumber(bStr)
    if err != nil {
        return 0, err
    }
    result, err := divide(a, b)
    if err != nil {
        return 0, err
    }
    return result, nil
}
```

**Вывод программы:**

```
10 / 2 = 5
Error: division by zero
Error: readNumber: strconv.Atoi: parsing "abc": invalid syntax
```

---

### [03_must](examples/03_must/) — оператор `must`

**Источник** [`examples/03_must/main.godsl`](examples/03_must/main.godsl):

```godsl
func main() {
    db := must openDB("localhost:5432/myapp")
    fmt.Println("Connected:", db)

    fmt.Println("About to panic if setup fails...")
    must setup()
}
```

**Результат транспиляции** `build/examples/03_must/main.go`:

```go
func main() {
    db, err := openDB("localhost:5432/myapp")
    if err != nil {
        panic(err)
    }
    fmt.Println("Connected:", db)
    fmt.Println("About to panic if setup fails...")
    if err := setup(); err != nil {
        panic(err)
    }
}
```

**Вывод программы:**

```
Connected: db:localhost:5432/myapp
About to panic if setup fails...
panic: setup failed

goroutine 1 [running]:
main.main()
        .../main.go:20 +0x...
```

---

### [04_try_catch](examples/04_try_catch/) — `try / catch` с типами

**Источник** [`examples/04_try_catch/main.godsl`](examples/04_try_catch/main.godsl):

```godsl
try {
    @errcheck
    res, err := fetchResource(0)
    fmt.Println("Got:", res)
} catch(NotFoundError) {
    fmt.Println("Caught NotFoundError:", err)
} catch(e PermissionError) {
    fmt.Printf("Access denied for '%s'\n", e.User)
} catch {
    fmt.Println("Caught unknown error:", err)
}
```

**Результат транспиляции** `build/examples/04_try_catch/main.go`:

```go
res, err := fetchResource(0)
if err != nil {
    if _, ok := err.(NotFoundError); ok {
        fmt.Println("Caught NotFoundError:", err)
    } else if e, ok := err.(PermissionError); ok {
        fmt.Printf("Access denied for '%s'\n", e.User)
    } else {
        fmt.Println("Caught unknown error:", err)
    }
}
fmt.Println("Got:", res)
```

**Вывод программы:**

```
Caught NotFoundError: not found: item#0
Access denied for 'guest'
Got: resource-42
```

---

### [05_try_catch_finally](examples/05_try_catch_finally/) — `try / catch / finally`

**Источник** [`examples/05_try_catch_finally/main.godsl`](examples/05_try_catch_finally/main.godsl):

```godsl
func runQuery(addr string) {
    conn, err := openConnection(addr)
    if err != nil {
        fmt.Println("Failed to open connection:", err)
        return
    }
    try {
        @errcheck
        result, err := query(conn)
        fmt.Println("Query result:", result)
    } catch {
        fmt.Println("Query failed:", err)
    } finally {
        conn.Close()
    }
}
```

**Результат транспиляции** `build/examples/05_try_catch_finally/main.go`:

```go
func runQuery(addr string) {
    conn, err := openConnection(addr)
    if err != nil {
        fmt.Println("Failed to open connection:", err)
        return
    }
    func() bool {
        result, err := query(conn)
        if err != nil {
            fmt.Println("Query failed:", err)
            return false
        }
        fmt.Println("Query result:", result)
        return false
    }()
    conn.Close()
}
```

**Вывод программы:**

```
--- Successful query ---
Connection opened: db.example.com:5432
Query result: SELECT result
Connection closed: db.example.com:5432

--- Failed connection ---
Failed to open connection: empty address
```

---

### [06_multi_type_catch](examples/06_multi_type_catch/) — `catch(A | B)` несколько типов

**Источник** [`examples/06_multi_type_catch/main.godsl`](examples/06_multi_type_catch/main.godsl):

```godsl
try {
    @errcheck
    res, err := riskyOp(kind)
    fmt.Println("result:", res)
} catch(ErrTimeout | ErrNetwork) {
    fmt.Println("transient error (retry later):", err)
} catch(e ErrDisk) {
    fmt.Printf("fatal disk error at '%s', aborting\n", e.Path)
} catch {
    fmt.Println("unexpected error:", err)
}
```

**Результат транспиляции** `build/examples/06_multi_type_catch/main.go`:

```go
res, err := riskyOp(kind)
if err != nil {
    if func() bool {
        if _, ok := err.(ErrTimeout); ok {
            return true
        }
        if _, ok := err.(ErrNetwork); ok {
            return true
        }
        return false
    }() {
        fmt.Println("transient error (retry later):", err)
    } else if e, ok := err.(ErrDisk); ok {
        fmt.Printf("fatal disk error at '%s', aborting\n", e.Path)
    } else {
        fmt.Println("unexpected error:", err)
    }
}
fmt.Println("result:", res)
```

**Вывод программы:**

```
[ok] result: ok
[timeout] transient error (retry later): timeout after 30s
[network] transient error (retry later): network error: connection refused
[disk] fatal disk error at '/var/data', aborting
```

---

### [07_ternary](examples/07_ternary/) — тернарный оператор `? :`

**Источник** [`examples/07_ternary/main.godsl`](examples/07_ternary/main.godsl):

```godsl
func abs(x int) any {
    return x >= 0 ? x : -x
}

func classify(x int) any {
    return x > 0 ? "positive" : x < 0 ? "negative" : "zero"
}
```

**Результат транспиляции** `build/examples/07_ternary/main.go` (типы выводятся автоматически):

```go
// abs: return type = any, ветки — идентификатор и унарный минус → any (нет литералов)
func abs(x int) any {
    return func() any {
        if x >= 0 {
            return x
        }
        return -x
    }()
}

// classify: return type = any, обе ветки — строковые литералы → string
func classify(x int) any {
    return func() string {
        if x > 0 {
            return "positive"
        }
        return func() string {
            if x < 0 {
                return "negative"
            }
            return "zero"
        }()
    }()
}
```

**Вывод программы:**

```
abs(-5) = 5
abs(7)  = 7
classify(-3) = negative
classify(0)  = zero
classify(5)  = positive
max(3, 7) = 7
max(9, 2) = 9
flag is enabled
```

## Тестирование

```bash
go test ./...
```

Тесты покрывают все конструкции языка.
