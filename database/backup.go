package database

import (
	"errors"
	"os"
	"strings"
)

// Backup writes a consistent snapshot of the open database.
// VACUUM INTO uses SQLite's online backup API, so a copy taken mid-write stays readable.
func Backup(dest string) error {
	if GetDB() == nil {
		return errors.New("数据库还没有打开")
	}
	if strings.ContainsAny(dest, "\r\n") {
		return errors.New("备份路径无效")
	}
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	escaped := strings.ReplaceAll(dest, "'", "''")
	return GetDB().Exec("VACUUM INTO '" + escaped + "'").Error
}
