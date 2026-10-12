// Package migrations содержит SQL-миграции, вшитые в бинарник.
package migrations

import "embed"

// FS — встроенная файловая система с .sql-файлами миграций.
// Используется в main при инициализации migrate.
//
//go:embed *.sql
var FS embed.FS
