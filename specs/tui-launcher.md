# TUI Launcher Architecture (Unified with Portalis & Warp)

## Overview

**TUI Launcher** — объединённое приложение для запуска задач в терминале с использованием **Portalis** терминального эмулятора и **Warp flex layout**.

```
┌─────────────────┐
│  TUI Launcher    │
│  -s myproject    │
│  (Portalis + Warp)│
└─────────────────┘
         │
         │ PTY + ANSI
         ▼
┌─────────────────┐
│  Portalis        │
│  Terminal Emulator│
└─────────────────┘
         │
         │ Flex Layout
         ▼
┌─────────────────┐
│  Warp TabGroup   │
│  [Tasks] [Term]  │
└─────────────────┘
```

## Architecture

### Unified Components

1. **Portalis** — терминальный эмулятор:
   - PTY wrapper (creack/pty)
   - ANSI parser (flat cell slice)
   - Screen buffer
   - Cursor blink
   - Auto-responder для запросов позиции курсора

2. **Warp Flex Layout**:
   - Left panel (30%) — список задач
   - Right panel (70%) — терминал Portalis
   - TabGroup для вкладок

3. **Bubbletea + Lipgloss** — TUI фреймворк и стильизация

### Workflow

1. **Запуск приложения**:
   ```bash
   ./bin/tui-launcher -s myproject
   ```

2. **Выбор задачи** (↑↓ навигация)

3. **Запуск** (Enter):
   - Задача запускается в Portalis терминале
   - Терминал отображается в правой панели

4. **Завершение**:
   - `q` или `Ctrl+C` — выход

## Idempotency

Задачи автоматически классифицируются:

| Задача | Idempotent | Поведение |
|--------|------------|-----------|
| `build` | ✅ true | Перезапуск терминала |
| `test` | ✅ true | Перезапуск терминала |
| `lint` | ✅ true | Перезапуск терминала |
| `dev` | ✅ true | Перезапуск терминала |
| `watch` | ✅ true | Перезапуск терминала |
| `serve` | ✅ true | Перезапуск терминала |
| `run` | ✅ true | Перезапуск терминала |
| `migrate` | ❌ false | Новый терминал |
| `deploy` | ❌ false | Новый терминал |
| `setup` | ❌ false | Новый терминал |

## File Structure

```
tui-launcher/
├── bin/              # Бинарники
│   └── tui-launcher
├── cmd/              # CLI команды
├── config/           # Конфигурация
├── executor/         # Выполнение задач
├── project/          # Работа с проектами
├── taskfile/         # Парсинг Taskfile.yml
├── packagejson/      # Парсинг package.json
├── ui/               # TUI интерфейс
├── viewer/           # Viewer компоненты
├── warp/             # Warp интеграция (flex layout)
├── main.go           # Entry point
├── go.mod            # Go модули
├── go.sum            # Зависимости
├── README.md         # Документация
├── USAGE.md          # Инструкция
└── SPEC.md           # Архитектура
```

## Dependencies

### Go
- `github.com/charmbracelet/bubbletea` — TUI фреймворк
- `github.com/charmbracelet/lipgloss` — стильизация
- `github.com/Starframe/portalis` — терминальный эмулятор

### Tools
- `cue-tty` — Playwright-like CLI для TUI

## CLI

```bash
tui-launcher -s <name>              # Start launcher
tui-launcher -s <name> help        # Show help
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `TERM` | Terminal type (default: `xterm-256color`) |
| `PROMPT_COMMAND` | Запуск при входе в терминал (для bash) |

## Best Practices

### Всегда указывай размер терминала

```go
cue.Launch(cmd, cue.WithSize(80, 24))
```

### Используй WaitStable после действий

```go
page.Press("Enter")
page.WaitStable(100 * time.Millisecond)
cue.Expect(page).ToContain("result")
```

### Включай мышь до рендеринга

```go
page := app.Page()
page.EnableMouse()
page.WaitStable(300 * time.Millisecond)
```

### Сохраняй артефакты при падении

```go
t.Cleanup(func() {
    if t.Failed() {
        page.SaveArtifact("test-artifacts", t.Name())
    }
})
```

## Status

- ✅ Сборка успешна
- ✅ Тестирование пройдено
- ✅ Документация создана
- ✅ Готов к использованию

---

**Разработчик**: Starframe Dev
**Лицензия**: MIT
