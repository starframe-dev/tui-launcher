# Project Context

[2026-05-31] Проблема: `go test`/`go vet` падали на `fmt.Println` с raw string, заканчивающимся newline → Решение: заменить `fmt.Println` на `fmt.Print` в help output (`main.go`, `viewer/main.go`).

[2026-10-14] Проблема: абсолютные `replace` в `go.mod` и CueTTY-тест, зависящий от предварительно собранного бинарника, делали публикацию/чистый checkout невоспроизводимыми; `Init()` также менял Bubbletea-модель из фоновой команды и иногда оставлял экран на `Loading tasks...` → Решение: использовать опубликованные module versions, собирать тестовый бинарник в `t.TempDir()` и передавать загруженные задачи typed message для обновления модели только в `Update()`. Перед публикацией проверять root и nested Go-модули отдельно, включая `go test -race`, `go vet`, `go mod verify` и `go mod tidy -diff`.
