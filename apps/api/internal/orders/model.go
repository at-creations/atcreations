// Package orders implements order placement, tracking and fulfilment.
// Payment is handled offline; the API never accepts prices from clients.
package orders

import (
	"slices"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/atcreations/atcreations/apps/api/internal/catalog"
)

type Status string

const (
	Pending    Status = "pending"
	Confirmed  Status = "confirmed"
	Processing Status = "processing"
	Shipped    Status = "shipped"
	Delivered  Status = "delivered"
	Completed  Status = "completed"
	Cancelled  Status = "cancelled"
	Refunded   Status = "refunded"
)

var transitions = map[Status][]Status{
	Pending:    {Confirmed, Cancelled},
	Confirmed:  {Processing, Cancelled},
	Processing: {Shipped, Cancelled},
	Shipped:    {Delivered},
	Delivered:  {Completed, Refunded},
	Completed:  {Refunded},
}

// CanTransition reports whether an order may move from one status to another.
func CanTransition(from, to Status) bool { return slices.Contains(transitions[from], to) }

// ShippingEditable reports whether the address may still change.
func ShippingEditable(s Status) bool { return s == Pending || s == Confirmed || s == Processing }

type Customer struct {
	Name  string `bson:"name" json:"name" validate:"required,max=100"`
	Email string `bson:"email" json:"email" validate:"required,email,max=254"`
	Phone string `bson:"phone,omitempty" json:"phone,omitempty" validate:"max=40"`
}

type Address struct {
	Name       string `bson:"name" json:"name" validate:"required,max=100"`
	Phone      string `bson:"phone" json:"phone" validate:"required,max=40"`
	Line1      string `bson:"line1" json:"line1" validate:"required,max=200"`
	Line2      string `bson:"line2,omitempty" json:"line2,omitempty" validate:"max=200"`
	City       string `bson:"city" json:"city" validate:"required,max=100"`
	Region     string `bson:"region,omitempty" json:"region,omitempty" validate:"max=100"`
	PostalCode string `bson:"postalCode" json:"postalCode" validate:"required,max=20"`
	Country    string `bson:"country" json:"country" validate:"required,len=2,uppercase,alpha"`
}

type Item struct {
	ProductID       bson.ObjectID         `bson:"productId" json:"productId"`
	Slug            string                `bson:"slug" json:"slug"`
	Name            catalog.LocalizedText `bson:"name" json:"name"`
	UnitPriceCents  int64                 `bson:"unitPriceCents" json:"unitPriceCents"`
	DiscountPercent int                   `bson:"discountPercent" json:"discountPercent"`
	Quantity        int                   `bson:"quantity" json:"quantity"`
	LineTotalCents  int64                 `bson:"lineTotalCents" json:"lineTotalCents"`
	ImageURL        string                `bson:"imageUrl,omitempty" json:"imageUrl,omitempty"`
	// ImageKey lets the media sweep keep images that orders still show.
	ImageKey string `bson:"imageKey,omitempty" json:"-"`
}

type Event struct {
	Type       string    `bson:"type" json:"type"`
	At         time.Time `bson:"at" json:"at"`
	Actor      string    `bson:"actor" json:"actor"`
	FromStatus Status    `bson:"fromStatus,omitempty" json:"fromStatus,omitempty"`
	ToStatus   Status    `bson:"toStatus,omitempty" json:"toStatus,omitempty"`
	Note       string    `bson:"note,omitempty" json:"note,omitempty"`
}

type Order struct {
	ID             bson.ObjectID `bson:"_id" json:"id"`
	Number         string        `bson:"number" json:"number"`
	AccessCodeHash string        `bson:"accessCodeHash" json:"-"`
	Status         Status        `bson:"status" json:"status"`
	Customer       Customer      `bson:"customer" json:"customer"`
	Shipping       Address       `bson:"shipping" json:"shipping"`
	Items          []Item        `bson:"items" json:"items"`
	SubtotalCents  int64         `bson:"subtotalCents" json:"subtotalCents"`
	DiscountCents  int64         `bson:"discountCents" json:"discountCents"`
	ShippingCents  int64         `bson:"shippingCents" json:"shippingCents"`
	TaxCents       int64         `bson:"taxCents" json:"taxCents"`
	TotalCents     int64         `bson:"totalCents" json:"totalCents"`
	Currency       string        `bson:"currency" json:"currency"`
	CustomerNote   string        `bson:"customerNote,omitempty" json:"customerNote,omitempty"`
	StaffNote      string        `bson:"staffNote,omitempty" json:"staffNote,omitempty"`
	TrackingNumber string        `bson:"trackingNumber,omitempty" json:"trackingNumber,omitempty"`
	History        []Event       `bson:"history" json:"history"`
	CreatedAt      time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time     `bson:"updatedAt" json:"updatedAt"`
}

type CreateInput struct {
	Customer     Customer    `json:"customer"`
	Shipping     Address     `json:"shipping"`
	Items        []ItemInput `json:"items" validate:"required,min=1,max=50,dive"`
	CustomerNote string      `json:"customerNote" validate:"max=1000"`
}

type ItemInput struct {
	ProductID bson.ObjectID `json:"productId" validate:"required"`
	Quantity  int           `json:"quantity" validate:"required,min=1,max=99"`
}

type Tracking struct {
	Number         string    `json:"number"`
	Status         Status    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	TrackingNumber string    `json:"trackingNumber,omitempty"`
	Items          []struct {
		Name     catalog.LocalizedText `json:"name"`
		Quantity int                   `json:"quantity"`
	} `json:"items"`
}
