// Package catalog implements products and categories.
package catalog

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type LocalizedText struct {
	EN string `bson:"en" json:"en"`
	VI string `bson:"vi,omitempty" json:"vi,omitempty"`
}

type Image struct {
	ID     string         `bson:"id"`
	Key    string         `bson:"key"`
	Width  int            `bson:"w"`
	Height int            `bson:"h"`
	Alt    *LocalizedText `bson:"alt,omitempty"`
}

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"

	MaxProductImages = 10
	Currency         = "CAD"
)

type Product struct {
	ID              bson.ObjectID   `bson:"_id"`
	Slug            string          `bson:"slug"`
	Status          string          `bson:"status"`
	Name            LocalizedText   `bson:"name"`
	Description     LocalizedText   `bson:"description"`
	PriceCents      *int64          `bson:"priceCents"`
	DiscountPercent int             `bson:"discountPercent"`
	FinalPriceCents *int64          `bson:"finalPriceCents"`
	HasPrice        bool            `bson:"hasPrice"`
	Stock           int             `bson:"stock"`
	Featured        bool            `bson:"featured"`
	CategoryIDs     []bson.ObjectID `bson:"categoryIds"`
	Images          []Image         `bson:"images"`
	// PendingImages counts uploads still being processed (they reserve room under the limit).
	PendingImages int   `bson:"pendingImages,omitempty"`
	Views         int64 `bson:"views"`
	RecentViews   int64 `bson:"recentViews"`
	// Version increments on every edit of the editable fields and on every
	// stock change, enabling optimistic concurrency for admin edits.
	Version int64 `bson:"version"`
	// Accent-folded copies used for search and locale-aware name sorting. SearchWords (names
	// first) backs word-prefix search with a multikey index.
	SearchWords []string  `bson:"searchWords"`
	SortNameEN  string    `bson:"sortNameEn"`
	SortNameVI  string    `bson:"sortNameVi"`
	CreatedAt   time.Time `bson:"createdAt"`
	UpdatedAt   time.Time `bson:"updatedAt"`
	CreatedBy   string    `bson:"createdBy,omitempty"`
	UpdatedBy   string    `bson:"updatedBy,omitempty"`
}

type Category struct {
	ID          bson.ObjectID `bson:"_id"`
	Slug        string        `bson:"slug"`
	Name        LocalizedText `bson:"name"`
	Description LocalizedText `bson:"description"`
	Image       *Image        `bson:"image,omitempty"`
	SortOrder   int           `bson:"sortOrder"`
	// ProductCount: products (any status) in the category; maintained on writes (see counts.go).
	ProductCount int64     `bson:"productCount,omitempty"`
	CreatedAt    time.Time `bson:"createdAt"`
	UpdatedAt    time.Time `bson:"updatedAt"`
}

// FinalPrice applies a percentage discount, rounding half up to the cent.
func FinalPrice(priceCents *int64, discountPercent int) *int64 {
	if priceCents == nil {
		return nil
	}
	v := (*priceCents*int64(100-discountPercent) + 50) / 100
	return &v
}

// ---- API representations ----

type ImageDTO struct {
	ID     string         `json:"id"`
	Key    string         `json:"key"`
	URL    string         `json:"url"`
	Width  int            `json:"width"`
	Height int            `json:"height"`
	Alt    *LocalizedText `json:"alt,omitempty"`
}

type CategoryRef struct {
	ID   bson.ObjectID `json:"id"`
	Slug string        `json:"slug"`
	Name LocalizedText `json:"name"`
}

type ProductDTO struct {
	ID              bson.ObjectID   `json:"id"`
	Slug            string          `json:"slug"`
	Status          string          `json:"status"`
	Name            LocalizedText   `json:"name"`
	Description     LocalizedText   `json:"description"`
	PriceCents      *int64          `json:"priceCents"`
	FinalPriceCents *int64          `json:"finalPriceCents"`
	Currency        string          `json:"currency"`
	DiscountPercent int             `json:"discountPercent"`
	Stock           int             `json:"stock"`
	Featured        bool            `json:"featured"`
	CategoryIDs     []bson.ObjectID `json:"categoryIds"`
	Categories      []CategoryRef   `json:"categories"`
	Images          []ImageDTO      `json:"images"`
	Views           int64           `json:"views"`
	Version         int64           `json:"version"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

type CategoryDTO struct {
	ID          bson.ObjectID `json:"id"`
	Slug        string        `json:"slug"`
	Name        LocalizedText `json:"name"`
	Description LocalizedText `json:"description"`
	Image       *ImageDTO     `json:"image"`
	SortOrder   int           `json:"sortOrder"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

// ---- Inputs ----

type NameInput struct {
	EN string `json:"en" validate:"required,max=200"`
	VI string `json:"vi" validate:"max=200"`
}

type DescriptionInput struct {
	EN string `json:"en" validate:"max=5000"`
	VI string `json:"vi" validate:"max=5000"`
}

type ProductInput struct {
	Slug            string           `json:"slug" validate:"omitempty,max=120"`
	Status          string           `json:"status" validate:"required,oneof=draft published archived"`
	Name            NameInput        `json:"name"`
	Description     DescriptionInput `json:"description"`
	PriceCents      *int64           `json:"priceCents" validate:"omitempty,min=0,max=100000000"`
	DiscountPercent int              `json:"discountPercent" validate:"min=0,max=90"`
	Stock           int              `json:"stock" validate:"min=0,max=1000000"`
	Featured        bool             `json:"featured"`
	CategoryIDs     []bson.ObjectID  `json:"categoryIds" validate:"max=20"`
	// Version the client loaded; required on update (409 if it changed).
	Version *int64 `json:"version"`
}

type CategoryInput struct {
	Slug        string           `json:"slug" validate:"omitempty,max=80"`
	Name        NameInput        `json:"name"`
	Description DescriptionInput `json:"description"`
	SortOrder   int              `json:"sortOrder" validate:"min=0,max=10000"`
}
