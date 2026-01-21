# DuplicateLink

🔍 Утилита для поиска, анализа и удаления дубликатов файлов с поддержкой восстановления.

> ⚠️ **ВАЖНО:** Перед использованием программы обязательно ознакомьтесь с [ОТКАЗОМ ОТ ОТВЕТСТВЕННОСТИ](DISCLAIMER.md). Использование этого ПО связано с риском потери данных. Вы используете программу на свой риск.

## Возможности

✨ **Основные функции:**

- 🔎 **Поиск дубликатов** по SHA1 или MD5 хешам
- 📊 **Сканирование папок** с поддержкой рекурсии
- 🔐 **Безопасное удаление** дубликатов с созданием симлинков
- 📁 **Автоматическая сортировка** файлов по категориям (изображения, видео, аудио и т.д.)
- 📝 **Генерация отчётов** в формате Markdown с активными ссылками (сканирование и очистка)
- 🔄 **Восстановление** исходного состояния из бэкапа
- 🔒 **Поддержка скрытых папок** (опционально)
- 📈 **Прогресс-бар** для отслеживания хода сканирования
- ❗ **Лог ошибок** при очистке + отдельный JSON для проблемных файлов

## Структура проекта

```
duplicatelink/
├── cmd/
│   └── duplicatelink/    # Точка входа приложения
│       └── main.go
├── internal/             # Внутренние пакеты
│   ├── models/          # Модели данных
│   ├── scanner/         # Модуль сканирования
│   ├── cleaner/         # Модуль очистки
│   └── restore/         # Модуль восстановления
├── bin/                 # Скомпилированные бинарные файлы
├── docs/                # Документация
├── Makefile            # Автоматизация сборки
├── go.mod              # Зависимости Go
└── README.md           # Этот файл
```

## Установка

### Требования

- Go 1.16 или выше
- Linux/macOS/Windows
- Make (опционально, для упрощения сборки)

### Сборка с помощью Make

```bash
# Собрать проект
make build

# Показать доступные команды
make help

# Установить в систему
make install
```

### Сборка вручную

```bash
# Из корня проекта
go build -o bin/duplicatelink ./cmd/duplicatelink

# Или с помощью go install
go install ./cmd/duplicatelink
```

## Использование

### 📖 Справка

Показать доступные команды Make:

```bash
make help
```

При запуске без аргументов программа выводит справку:

```bash
./bin/duplicatelink
# или
make run
```

Или используйте флаг `-h`:

```bash
./bin/duplicatelink -h
```

### 1️⃣ Режим сканирования (SCAN)

Поиск и анализ дубликатов файлов.

**Базовое использование:**

```bash
# С помощью Make
make run ARGS="/path/to/folder"

# Напрямую
./bin/duplicatelink /path/to/folder
```

**С параметрами:**

```bash
# С помощью Make
make run ARGS="-hash sha1 -output results.json /path/to/folder"

# Напрямую
./bin/duplicatelink -hash sha1 -output results.json /path/to/folder
```

**Параметры сканирования:**

| Флаг | Описание | По умолчанию |
|------|---------|-------------|
| `-hash` | Алгоритм хеширования (sha1 или md5) | sha1 |
| `-output` | Файл для сохранения результатов в JSON | duplicates.json |
| `-scan-report` | Файл Markdown отчёта сканирования | scan_report.md |
| `-include-hidden` | Включить сканирование скрытых папок | false |

**Примеры:**

```bash
# Сканирование с SHA1 (с помощью Make)
make run ARGS="-hash sha1 /home/user/Downloads/"

# Сканирование с MD5 и сохранением в пользовательский файл
make run ARGS="-hash md5 -output my_duplicates.json -scan-report my_report.md /home/user/Documents/"

# Сканирование со скрытыми папками
make run ARGS="-include-hidden -output full_scan.json /home/user/"

# Напрямую через бинарный файл
./bin/duplicatelink -hash sha1 /home/user/Downloads/
./bin/duplicatelink -hash md5 -output my_duplicates.json -scan-report my_report.md /home/user/Documents/
./bin/duplicatelink -include-hidden -output full_scan.json /home/user/
```

**Результат:**

- JSON файл с информацией о дубликатах, готовый для очистки
- Markdown отчёт сканирования (`-scan-report`), c группами дубликатов и ссылками

#### Игнорирование путей (.dupignore)

Вы можете указать файлы и папки, которые нужно игнорировать при сканировании и при очистке, создав файл `.dupignore` в корне сканируемой директории. Формат аналогичен `.gitignore`:

- `ignored/` — игнорировать папку и всё внутри
- `*.bak` — игнорировать все файлы с расширением `.bak`
- `temp/*` — игнорировать все файлы внутри `temp`

Дополнительно поддерживаются исключающие правила `!` (последнее совпадение побеждает):

- `*` — игнорировать всё
- `!*.jpg` — разрешить файлы с расширением `.jpg`
- Комбинируйте несколько `!`-правил, чтобы разрешить нужные типы.

Комментарии начинаются с `#`.

Пример файла:

```
# Игнорируем временные файлы и кэш
temp/
cache/
*.bak
```

Если `.dupignore` присутствует, программа автоматически применит его как при поиске дубликатов, так и при очистке.

Пример: игнорировать всё, кроме фотографий и видео

```
# Игнорировать все файлы
*

# Разрешить фото
!*.jpg
!*.jpeg
!*.png
!*.gif
!*.bmp
!*.webp
!*.tiff
!*.tif
!*.heic
!*.heif

# Разрешить распространённые RAW-форматы
!*.cr2
!*.cr3
!*.nef
!*.arw
!*.dng

# Разрешить видео
!*.mp4
!*.mov
!*.mkv
!*.avi
!*.webm
```

Готовые шаблоны и полное руководство:

- Полное руководство: [docs/DUPIGNORE.md](docs/DUPIGNORE.md)
- Фото/Видео: [docs/dupignore-photos-videos.example](docs/dupignore-photos-videos.example)
- Документы: [docs/dupignore-docs.example](docs/dupignore-docs.example)
- AutoCAD: [docs/dupignore-autocad.example](docs/dupignore-autocad.example)

 

### 2️⃣ Режим очистки (CLEAN)

Удаление дубликатов с сохранением одного экземпляра и созданием симлинков.

```bash
# С помощью Make
make run ARGS="-mode clean -json duplicates.json -target /path/to/storage"

# Напрямую
./bin/duplicatelink -mode clean -json duplicates.json -target /path/to/storage
```

**Параметры очистки:**

| Флаг | Описание | Обязательный |
|------|---------|-------------|
| `-json` | JSON файл со списком дубликатов | ✅ |
| `-target` | Папка назначения для хранения файлов | ✅ |
| `-mode` | Установить в значение "clean" | ✅ |

**Что происходит при очистке:**

1. 📂 Создаются папки по категориям (image, video, audio и т.д.)
2. 📋 Один экземпляр каждого дубликата копируется в папку назначения
3. 🗑️ Все остальные дубликаты удаляются из исходных мест
4. 🔗 На месте удалённых файлов создаются символические ссылки
5. 📄 Генерируется Markdown отчёт `cleanup_report.md` (включает хеш-алгоритм и ошибки, если были)
6. 💾 Создаются JSON файлы восстановления: `restore.json` (все действия + ошибки) и `restore_errors_only.json` (только проблемные файлы)

**Пример:**

```bash
./duplicatelink -mode clean -json download.json -target /mnt/storage/deduplicated
```

**Результат:**

Структура папки назначения:
```
/mnt/storage/deduplicated/
├── image/
│   ├── photo1.jpg
│   ├── photo2.png
│   └── ...
├── video/
│   ├── movie.mp4
│   └── ...
├── audio/
│   ├── song.mp3
│   └── ...
├── cleanup_report.md
├── restore.json
└── restore_errors_only.json
```

**Дополнительные сценарии очистки:**

```bash
# Очистка с пользовательским отчётом и отдельной папкой назначения
make run ARGS="-mode clean -json scan.json -target /mnt/dedup -scan-report /mnt/dedup/cleanup_report.md"

# Очистка со скрытыми файлами (скан отчёт уже получен ранее)
./bin/duplicatelink -mode clean -json scan.json -target /mnt/dedup_hidden -include-hidden

# Очистка нескольких папок (сканируйте по очереди, затем чистите общий таргет)
./bin/duplicatelink -mode clean -json downloads.json -target /mnt/dedup/all
./bin/duplicatelink -mode clean -json documents.json -target /mnt/dedup/all
```

### 3️⃣ Режим восстановления (RESTORE)

Возврат файловой системы в исходное состояние.

```bash
# С помощью Make
make run ARGS="-mode restore -restore /path/to/storage/restore.json"

# Напрямую
./bin/duplicatelink -mode restore -restore /path/to/storage/restore.json
```

**Параметры восстановления:**

| Флаг | Описание | Обязательный |
|------|---------|-------------|
| `-restore` | JSON файл восстановления | ✅ |
| `-mode` | Установить в значение "restore" | ✅ |

**Что происходит при восстановлении:**

1. 🔗 Все символические ссылки удаляются
2. 📂 Оригинальные файлы копируются обратно в исходные места
3. ✅ Файловая структура возвращается в исходное состояние

**Пример:**

```bash
# С помощью Make
make run ARGS="-mode restore -restore /mnt/storage/deduplicated/restore.json"

# Напрямую
./bin/duplicatelink -mode restore -restore /mnt/storage/deduplicated/restore.json
```

**Дополнительные сценарии восстановления:**

```bash
# Восстановление и проверка отчёта очистки
make run ARGS="-mode restore -restore /mnt/dedup/restore.json" && cat /mnt/dedup/cleanup_report.md | head

# Восстановление только после ручной проверки JSON (пример с другой директории)
./bin/duplicatelink -mode restore -restore /backups/dedup_run_2026-01-21/restore.json

# Повторное восстановление на случай, если часть файлов была заблокирована в прошлый раз
./bin/duplicatelink -mode restore -restore /mnt/dedup/restore_errors_only.json
```

## Примеры использования

### Полный цикл работы

```bash
# 1. Сборка проекта
make build

# 2. Сканирование папки
make run ARGS="-hash sha1 -output scan.json /home/user/Downloads/"
# или напрямую
./bin/duplicatelink -hash sha1 -output scan.json /home/user/Downloads/

# Просмотр результатов (опционально)
cat scan.json | jq '.duplicates | length'

# 3. Очистка дубликатов
make run ARGS="-mode clean -json scan.json -target /mnt/deduplicated"
# или напрямую
./bin/duplicatelink -mode clean -json scan.json -target /mnt/deduplicated

# Просмотр отчёта
cat /mnt/deduplicated/cleanup_report.md
# (опционально) только ошибки
cat /mnt/deduplicated/restore_errors_only.json

# 4. При необходимости - восстановление
make run ARGS="-mode restore -restore /mnt/deduplicated/restore.json"
# или напрямую
./bin/duplicatelink -mode restore -restore /mnt/deduplicated/restore.json
```

### Сканирование нескольких папок

```bash
# Сборка (если еще не собрано)
make build

# Сканирование Downloads
make run ARGS="-output downloads.json ~/Downloads/"

# Сканирование Documents
make run ARGS="-hash md5 -output documents.json ~/Documents/"

# Сканирование со скрытыми файлами
make run ARGS="-include-hidden -output full.json ~/"
```

### Работа на удаленных серверах

```bash
# На сервере - создать отчет
ssh user@server "cd /data && /path/to/duplicatelink -output scan.json ./"

# На локальной машине - скачать отчет
scp user@server:/data/scan.json .

# Очистить на сервере
ssh user@server "cd /data && /path/to/duplicatelink -mode clean -json scan.json -target /mnt/storage"
```

## Категории файлов

При очистке файлы автоматически сортируются по следующим категориям:

| Категория | Расширения |
|-----------|-----------|
| **image** | jpg, jpeg, png, gif, bmp, webp, svg, ico, tiff, tif |
| **video** | mp4, avi, mkv, mov, wmv, flv, webm, m4v, mpg, mpeg |
| **audio** | mp3, wav, flac, aac, ogg, wma, m4a, opus |
| **document** | pdf, doc, docx, txt, rtf, odt, tex, md |
| **archive** | zip, rar, 7z, tar, gz, bz2, xz, iso |
| **code** | go, py, js, java, cpp, c, h, rs, rb, php, html, css |
| **data** | json, xml, yaml, yml, csv, sql, db |
| **other** | прочие файлы |

## JSON форматы

### Формат сканирования (duplicates.json)

```json
{
  "folder_path": "/home/user/Downloads",
  "algorithm": "SHA1",
  "duplicates": [
    {
      "hash": "da39a3ee5e6b4b0d3255bfef95601890afd80709",
      "size": 1024,
      "algorithm": "SHA1",
      "files": [
        {
          "path": "/home/user/Downloads/file1.jpg",
          "hash": "da39a3ee5e6b4b0d3255bfef95601890afd80709",
          "size": 1024
        },
        {
          "path": "/home/user/Downloads/file2.jpg",
          "hash": "da39a3ee5e6b4b0d3255bfef95601890afd80709",
          "size": 1024
        }
      ]
    }
  ]
}
```

### Формат восстановления (restore.json)

```json
{
  "timestamp": "deduplicated_backup",
  "source_json": "",
  "target_dir": "/mnt/deduplicated",
  "actions": [
    {
      "original_path": "/home/user/Downloads/file1.jpg",
      "moved_to": "/mnt/deduplicated/image/file1.jpg",
      "symlink_path": "",
      "is_symlink": false
    },
    {
      "original_path": "/home/user/Downloads/file2.jpg",
      "moved_to": "/mnt/deduplicated/image/file1.jpg",
      "symlink_path": "/home/user/Downloads/file2.jpg",
      "is_symlink": true
    }
  ],
  "errors": [
    {
      "file_path": "/home/user/Downloads/file2.jpg",
      "operation": "symlink",
      "error": "file exists",
      "timestamp": "0",
      "error_type": "symlink_failed"
    }
  ],
  "report_file": "/mnt/deduplicated/cleanup_report.md"
}
```

### Формат файла только с ошибками (restore_errors_only.json)

```json
{
  "timestamp": "deduplicated_backup",
  "source_json": "",
  "target_dir": "/mnt/deduplicated",
  "actions": [],
  "errors": [
    {
      "file_path": "/home/user/Downloads/file2.jpg",
      "operation": "symlink",
      "error": "file exists",
      "timestamp": "0",
      "error_type": "symlink_failed"
    }
  ],
  "report_file": "/mnt/deduplicated/cleanup_report.md"
}
```

## Отчёты

### Отчёт очистки (cleanup_report.md)

Автоматически генерируемый Markdown отчёт содержит:

- 📋 Информацию о дате, папке источника и алгоритме хеширования
- 🔢 Количество групп дубликатов
- 📁 Категорию каждого файла
- 🔗 Активные ссылки на файлы
- 📊 Статистику по каждой группе
- ⚠️ Таблицу ошибок (если были: операция, тип ошибки, описание)

**Пример:**

```markdown
# Отчет об очистке дубликатов

**Дата:** deduplicated_backup

**Исходная папка:** /home/user/Downloads

**Алгоритм:** SHA1

**Групп дубликатов:** 5

---

## Группа 1

**Хеш (SHA1):** `da39a3ee5e6b4b0d3255bfef95601890afd80709`

**Размер:** 1024 байт

**Категория:** image

**Основной файл:** [file1.jpg](file:///mnt/deduplicated/image/file1.jpg)

**Дубликаты (заменены симлинками):**

- [/home/user/Downloads/file2.jpg](file:///home/user/Downloads/file2.jpg)
- [/home/user/Downloads/file3.jpg](file:///home/user/Downloads/file3.jpg)

### Отчёт сканирования (scan_report.md)

- 📋 Папка сканирования и алгоритм
- 🔢 Общее число проверенных файлов
- 🔗 Список групп дубликатов с активными ссылками
- 🧭 Указывается хеш с явным алгоритмом (SHA1/MD5)
```

## Безопасность и рекомендации

⚠️ **Важно!**

1. **Бэкапирование:** Всегда создавайте резервную копию перед очисткой
2. **Тестирование:** Протестируйте на небольшой папке перед обработкой больших данных
3. **Проверка отчёта:** Внимательно проверьте `cleanup_report.md` перед удалением
4. **Восстановление:** JSON файл восстановления должен храниться в безопасном месте

## Примеры команд

```bash
# Вывод справки
./duplicatelink

# Быстрое сканирование текущей папки
./duplicatelink .

# Сканирование с заданным алгоритмом
./duplicatelink -hash md5 ~/Downloads

# Очистка с минимальным вводом
./duplicatelink -mode clean -json duplicates.json -target ./storage

# Подробная справка по флагам
./duplicatelink -h
```

## Возможные ошибки и решения

| Ошибка | Причина | Решение |
|--------|---------|---------|
| "ошибка: неподдерживаемый режим" | Неправильный режим | Используйте scan, clean или restore |
| "не удалось открыть папку" | Папка не существует | Проверьте путь |
| "ошибка при создании симлинка" | Файловая система не поддерживает симлинки | Используйте другую папку назначения |
| "ошибка при разборе JSON" | Повреждённый JSON файл | Пересканируйте папку |

## Производительность

- 📊 Скорость сканирования: зависит от количества файлов и скорости диска
- 💾 Использование памяти: минимальное (хеши хранятся в памяти)
- 🔄 Очистка: примерно столько же времени, сколько скопировать файлы

**Рекомендации для больших папок:**

- Используйте SSD для лучшей производительности
- Закройте другие программы, использующие диск
- Запустите в фоновом режиме для больших объёмов

## Версия и информация

**Версия:** 1.0.0  
**Язык:** Go  
**Лицензия:** MIT  
**Автор:** Tolik Proh

## Обратная связь

Если у вас есть предложения или нашли баги, создайте issue в репозитории.

---

Спасибо за использование DuplicateLink! 🙏
