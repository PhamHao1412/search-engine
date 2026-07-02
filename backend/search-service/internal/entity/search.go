package entity

import "time"

// --- Schema: product ---

type Product struct {
	ID               string    `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID         string    `gorm:"type:uuid;not null" json:"tenant_id"`
	Name             string    `gorm:"type:varchar(255);not null" json:"name"`
	Description      string    `gorm:"type:text" json:"description"`
	CategoryID       *string   `gorm:"type:uuid" json:"category_id,omitempty"`
	Brand            string    `gorm:"type:varchar(100)" json:"brand"`
	Price            float64   `gorm:"type:decimal(15,2);not null;default:0.00" json:"price"`
	ImageURL         string    `gorm:"type:varchar(500)" json:"image_url"`
	Inventory        int       `gorm:"type:integer;not null;default:0" json:"inventory"`
	Status           string    `gorm:"type:varchar(50);not null;default:'active'" json:"status"`
	Featured         bool      `gorm:"type:boolean;not null;default:false" json:"featured"`
	OriginalLanguage string    `gorm:"type:varchar(10);not null;default:'vi'" json:"original_language"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Product) TableName() string {
	return "product_svc.products"
}

type ProductTranslation struct {
	ID                    string    `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID              string    `gorm:"type:uuid;not null" json:"tenant_id"`
	ProductID             string    `gorm:"type:uuid;not null" json:"product_id"`
	LanguageCode          string    `gorm:"type:varchar(10);not null" json:"language_code"`
	NameTranslated        string    `gorm:"type:varchar(255);not null" json:"name_translated"`
	DescriptionTranslated string    `gorm:"type:text" json:"description_translated"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (ProductTranslation) TableName() string {
	return "product_svc.product_translations"
}

// --- Schema: search ---

type SearchSynonym struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID  string    `gorm:"type:uuid;not null" json:"tenant_id"`
	Keyword   string    `gorm:"type:varchar(255);not null" json:"keyword"`
	Synonym   string    `gorm:"type:varchar(255);not null" json:"synonym"`
	Status    string    `gorm:"type:varchar(50);not null;default:'active'" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SearchSynonym) TableName() string {
	return "search_svc.search_synonyms"
}

type SearchLog struct {
	ID              string    `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID        string    `gorm:"type:uuid;not null" json:"tenant_id"`
	Query           string    `gorm:"type:varchar(255);not null" json:"query"`
	NormalizedQuery string    `gorm:"type:varchar(255);not null" json:"normalized_query"`
	ResultCount     int       `gorm:"type:integer;not null;default:0" json:"result_count"`
	SearchedAt      time.Time `json:"searched_at"`
}

func (SearchLog) TableName() string {
	return "search_svc.search_logs"
}

type ClickLog struct {
	ID            string    `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID      string    `gorm:"type:uuid;not null" json:"tenant_id"`
	SearchLogID   string    `gorm:"type:uuid;not null" json:"search_log_id"`
	Query         string    `gorm:"type:varchar(255);not null" json:"query"`
	ProductID     string    `gorm:"type:uuid;not null" json:"product_id"`
	ClickPosition int       `gorm:"type:integer;not null;default:1" json:"position"`
	ClickedAt     time.Time `json:"clicked_at"`
}

func (ClickLog) TableName() string {
	return "search_svc.click_logs"
}

type SpellcheckDictionary struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID    string    `gorm:"type:uuid;not null" json:"tenant_id"`
	TypoWord    string    `gorm:"type:varchar(255);not null" json:"typo_word"`
	CorrectWord string    `gorm:"type:varchar(255);not null" json:"correct_word"`
	Status      string    `gorm:"type:varchar(50);not null;default:'active'" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (SpellcheckDictionary) TableName() string {
	return "search_svc.spellcheck_dictionary"
}

type AISuggestion struct {
	ID              string    `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID        string    `gorm:"type:uuid;not null" json:"tenant_id"`
	SuggestionType  string    `gorm:"type:varchar(100);not null" json:"suggestion_type"`
	SourceValue     string    `gorm:"type:varchar(255);not null" json:"source_value"`
	SuggestedValue  string    `gorm:"type:varchar(255);not null" json:"suggested_value"`
	ConfidenceScore float64   `gorm:"type:decimal(5,4);not null;default:0.0000" json:"confidence_score"`
	Status          string    `gorm:"type:varchar(50);not null;default:'pending'" json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

func (AISuggestion) TableName() string {
	return "search_svc.ai_suggestions"
}

type ProductEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	TenantID  string    `json:"tenant_id"`
	Timestamp time.Time `json:"timestamp"`
	Data      Product   `json:"data"`
}

type SearchSyncJob struct {
	ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID     string    `gorm:"type:uuid;not null" json:"tenant_id"`
	ProductID    string    `gorm:"type:uuid;not null;uniqueIndex" json:"product_id"`
	Status       string    `gorm:"type:varchar(50);not null;default:'pending'" json:"status"`
	ErrorMessage *string   `gorm:"type:text" json:"error_message,omitempty"`
	RetryCount   int       `gorm:"type:integer;not null;default:0" json:"retry_count"`
	TextHash     string    `gorm:"type:varchar(64)" json:"text_hash"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (SearchSyncJob) TableName() string {
	return "search_svc.search_sync_jobs"
}

type Suggestion struct {
	ID            string  `json:"id"`
	Text          string  `json:"text"`
	Brand         string  `json:"brand"`
	Price         float64 `json:"price"`
	ProductNameVI string  `json:"product_name_vi"`
	ProductNameEN string  `json:"product_name_en"`
	ProductNameTH string  `json:"product_name_th"`
	DescriptionVI string  `json:"description_vi"`
	DescriptionEN string  `json:"description_en"`
	DescriptionTH string  `json:"description_th"`
	ImageURL      string  `json:"image_url"`
	Inventory     int     `json:"inventory"`
}

type Tenant struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Tenant) TableName() string {
	return "product_svc.tenants"
}

type SearchTranslation struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID    string    `gorm:"type:uuid;not null" json:"tenant_id"`
	Keyword     string    `gorm:"type:varchar(255);not null" json:"keyword"`
	LangCode    string    `gorm:"type:varchar(10);not null" json:"lang_code"`
	Translation string    `gorm:"type:varchar(255);not null" json:"translation"`
	Status      string    `gorm:"type:varchar(50);not null;default:'active'" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (SearchTranslation) TableName() string {
	return "search_svc.search_translations"
}
