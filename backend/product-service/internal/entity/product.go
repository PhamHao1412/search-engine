package entity

import "time"

type Tenant struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TenantConfig struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID    string    `gorm:"type:uuid;not null" json:"tenant_id"`
	ConfigKey   string    `gorm:"type:varchar(100);not null" json:"config_key"`
	ConfigValue string    `gorm:"type:text;not null" json:"config_value"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Category struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TenantID  string    `gorm:"type:uuid;not null" json:"tenant_id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	ParentID  *string   `gorm:"type:uuid" json:"parent_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

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

type ProductEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"` // "ProductCreated", "ProductUpdated", "ProductDeleted"
	TenantID  string    `json:"tenant_id"`
	Timestamp time.Time `json:"timestamp"`
	Data      Product   `json:"data"`
}
