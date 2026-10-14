# 🚀 TUI Launcher (Unified with Portalis & Warp)

**Объединённое TUI приложение** для запуска задач в терминале с использованием Portalis терминального эмулятора и Warp flex layout.

## ✨ Возможности

- ✅ **Portalis** — терминальный эмулятор (PTY, ANSI парсинг, screen buffer)
- ✅ **Warp Flex Layout** — организация панелей (левая: задачи, правая: терминал)
- ✅ **Автоопределение** пакетных менеджеров (pnpm > yarn > bun > npm)
- ✅ **Idempotency detection** — классификация задач
- ✅ **Taskfile.yml / package.json** парсинг
- ✅ **MacOS совместимость** (session ID вместо process group)

## 🚀 Быстрый старт

### Сборка

```bash
cd ~/Space/Projects/Starframe/tui-launcher
go build .
```

Бинарник: `bin/tui-launcher` (~5.4 MB)

### Запуск

```bash
./bin/tui-launcher [-s session-name]
```

**Или через go run:**

```bash
go run . -s session-name
```

### Управление

| Клавиша | Действие |
|---------|----------|
| `↑↓` | Навигация между задачами |
| `Enter` | Запустить выбранную задачу |
| `q` | Выход |
| `Ctrl+C` | Выход |

## 📋 Поддерживаемые задачи

- `build` — сборка проекта
- `test` — запуск тестов
- `lint` — линтинг
- `dev` — режим разработки
- `serve` — запуск сервера
- `run` — выполнение

## 🎨 Архитектура

```
tui-launcher/
├── bin/              # Бинарники
├── cmd/              # CLI команды
├── config/           # Конфигурация
├── executor/         # Выполнение задач
├── project/          # Работа с проектами
├── taskfile/         # Парсинг Taskfile.yml
├── packagejson/      # Парсинг package.json
├── ui/               # TUI интерфейс
├── viewer/           # Viewer компоненты
└── warp/             # Warp интеграция (flex layout)
```

**Layout:**
- Левая панель (30%) — список задач
- Правая панель (70%) — терминальный эмулятор Portalis

## 📦 Поддержка пакетных менеджеров

Автоматическое определение через lock-файлы:
- `pnpm-lock.yaml` (приоритет)
- `yarn.lock`
- `bun.lockb`
- `package-lock.json`

## ⚠️ Важное замечание

**Приложение требует интерактивный запуск в TTY!**

Не работает в фоновом режиме:

```bash
# ❌ Не работает (в фоне)
nohup ./bin/tui-launcher &

# ✅ Работает (напрямую в терминале)
./bin/tui-launcher -s session-name
```

## 📚 Документация

- [USAGE.md](USAGE.md) — подробная инструкция
- [SPEC.md](SPEC.md) — архитектура и спецификации
- [CONTEXT.md](CONTEXT.md) — контекст проекта

## 🛠️ Технологии

- **Bubbletea** — TUI фреймворк
- **Lipgloss** — стильизация
- **Portalis** — терминальный эмулятор
- **Warp** — layout engine

## 📊 Статус

- ✅ Сборка успешна
- ✅ Тестирование пройдено
- ✅ Документация создана
- ✅ Готов к использованию

---

**Разработчик**: Starframe Dev
**Лицензия**: MIT
