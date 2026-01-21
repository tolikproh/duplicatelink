.PHONY: all build clean install run test help

# Переменные
BINARY_NAME=duplicatelink
BINARY_PATH=./bin/$(BINARY_NAME)
CMD_PATH=./cmd/duplicatelink
GO=go
GOFLAGS=-v

# Цвета для красивого вывода
COLOR_RESET=\033[0m
COLOR_BOLD=\033[1m
COLOR_GREEN=\033[32m
COLOR_YELLOW=\033[33m

all: build ## Собрать проект (по умолчанию)

build: ## Собрать бинарный файл
	@echo "$(COLOR_BOLD)$(COLOR_GREEN)Сборка $(BINARY_NAME)...$(COLOR_RESET)"
	@mkdir -p bin
	@$(GO) build $(GOFLAGS) -o $(BINARY_PATH) $(CMD_PATH)
	@echo "$(COLOR_GREEN)✓ Сборка завершена: $(BINARY_PATH)$(COLOR_RESET)"

clean: ## Очистить собранные файлы
	@echo "$(COLOR_YELLOW)Очистка...$(COLOR_RESET)"
	@rm -f $(BINARY_PATH)
	@rm -f duplicates.json scan_report.md
	@echo "$(COLOR_GREEN)✓ Очистка завершена$(COLOR_RESET)"

install: build ## Установить бинарный файл в $GOPATH/bin
	@echo "$(COLOR_BOLD)$(COLOR_GREEN)Установка $(BINARY_NAME)...$(COLOR_RESET)"
	@$(GO) install $(CMD_PATH)
	@echo "$(COLOR_GREEN)✓ Установка завершена$(COLOR_RESET)"

run: build ## Собрать и запустить с параметрами (использовать: make run ARGS="...")
	@echo "$(COLOR_BOLD)$(COLOR_GREEN)Запуск $(BINARY_NAME)...$(COLOR_RESET)"
	@$(BINARY_PATH) $(ARGS)

test: ## Запустить тесты
	@echo "$(COLOR_BOLD)$(COLOR_GREEN)Запуск тестов...$(COLOR_RESET)"
	@$(GO) test -v ./...

fmt: ## Форматировать код
	@echo "$(COLOR_YELLOW)Форматирование кода...$(COLOR_RESET)"
	@$(GO) fmt ./...
	@echo "$(COLOR_GREEN)✓ Форматирование завершено$(COLOR_RESET)"

vet: ## Проверить код с помощью go vet
	@echo "$(COLOR_YELLOW)Проверка кода...$(COLOR_RESET)"
	@$(GO) vet ./...
	@echo "$(COLOR_GREEN)✓ Проверка завершена$(COLOR_RESET)"

deps: ## Скачать зависимости
	@echo "$(COLOR_YELLOW)Загрузка зависимостей...$(COLOR_RESET)"
	@$(GO) mod download
	@echo "$(COLOR_GREEN)✓ Зависимости загружены$(COLOR_RESET)"

tidy: ## Очистить зависимости
	@echo "$(COLOR_YELLOW)Очистка зависимостей...$(COLOR_RESET)"
	@$(GO) mod tidy
	@echo "$(COLOR_GREEN)✓ Зависимости очищены$(COLOR_RESET)"

help: ## Показать эту справку
	@echo "$(COLOR_BOLD)Доступные команды:$(COLOR_RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(COLOR_GREEN)%-15s$(COLOR_RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(COLOR_BOLD)Примеры использования:$(COLOR_RESET)"
	@echo "  make build                          # Собрать проект"
	@echo "  make run ARGS=\"/path/to/folder\"     # Запустить сканирование"
	@echo "  make run ARGS=\"-hash md5 /path\"     # Запустить с MD5"
	@echo "  make clean                          # Очистить собранные файлы"
	@echo "  make install                        # Установить в системе"
