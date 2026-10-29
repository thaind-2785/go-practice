package models

type ExportFile struct {
	Format string `json:"format"`
	Path   string `json:"path"`
}

type ExportResult struct {
	TotalEmployees int          `json:"totalEmployees"`
	Files          []ExportFile `json:"files"`
	Duration       string       `json:"duration"`
}
