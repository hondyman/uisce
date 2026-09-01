package reporting

import (
	"github.com/google/uuid"
)

type TOCEntry struct {
	SectionOrder int    `json:"sectionOrder"`
	ChapterTitle string `json:"chapterTitle"`
	StartPageNum int    `json:"startPageNum"`
	PageCount    int    `json:"pageCount"`
}

type BookSectionConfig struct {
	SectionID           uuid.UUID `json:"sectionId"`
	ReportDefinitionID  uuid.UUID `json:"reportDefinitionId"`
	SectionOrder        int       `json:"sectionOrder"`
	ChapterTitle        string    `json:"chapterTitle"`
	InsertDividerTab    bool      `json:"insertDividerTab"`
	DividerSubtitle     string    `json:"dividerSubtitle"`
	ConditionExpression string    `json:"conditionExpression"`
}

type ReportBookSpec struct {
	BookID             uuid.UUID           `json:"bookId"`
	TenantID           uuid.UUID           `json:"tenantId"`
	BookName           string              `json:"bookName"`
	IncludeTOC         bool                `json:"includeToc"`
	TOCTitle           string              `json:"tocTitle"`
	PageNumberingStyle string              `json:"pageNumberingStyle"` // CONTINUOUS, FRONT_MATTER_ROMAN
	PageFormat         PageFormat          `json:"pageFormat"`
	Sections           []BookSectionConfig `json:"sections"`
}

type CompiledBookletArtifact struct {
	ArtifactID     uuid.UUID  `json:"artifactId"`
	ClientID       string     `json:"clientId"`
	TotalPages     int        `json:"totalPages"`
	TOCEntries     []TOCEntry `json:"tocEntries"`
	PDFBinary      []byte     `json:"-"`
	SHA256Checksum string     `json:"sha256Checksum"`
	FileSizeBytes  int64      `json:"fileSizeBytes"`
}
