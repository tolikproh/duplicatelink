package models

// FileHash хранит информацию о файле и его хешу
type FileHash struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// DuplicateGroup группа дубликатов
type DuplicateGroup struct {
	Hash      string     `json:"hash"`
	Size      int64      `json:"size"`
	Files     []FileHash `json:"files"`
	Algorithm string     `json:"algorithm"`
}

// ScanResult результат сканирования
type ScanResult struct {
	FolderPath string           `json:"folder_path"`
	Algorithm  string           `json:"algorithm"`
	Duplicates []DuplicateGroup `json:"duplicates"`
}

// RestoreAction действие для восстановления
type RestoreAction struct {
	OriginalPath string `json:"original_path"`
	MovedTo      string `json:"moved_to"`
	SymlinkPath  string `json:"symlink_path,omitempty"`
	IsSymlink    bool   `json:"is_symlink"`
}

// ErrorInfo информация об ошибке при обработке файла
type ErrorInfo struct {
	FilePath  string `json:"file_path"`
	Operation string `json:"operation"`
	Error     string `json:"error"`
	Timestamp string `json:"timestamp"`
	ErrorType string `json:"error_type"`
}

// RestoreData данные для восстановления
type RestoreData struct {
	Timestamp  string          `json:"timestamp"`
	SourceJSON string          `json:"source_json"`
	TargetDir  string          `json:"target_dir"`
	Actions    []RestoreAction `json:"actions"`
	Errors     []ErrorInfo     `json:"errors,omitempty"`
	ReportFile string          `json:"report_file"`
}
