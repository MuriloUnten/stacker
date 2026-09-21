package main

import (
	"time"
	"github.com/shopspring/decimal"
)

type ItemKind string
func (k ItemKind) String() string {
	return string(k)
}
const (
	COMPONENT ItemKind = "c"
	ASSEMBLY  ItemKind = "a"
)

type TransactionKind string
const (
	CONSUMPTION TransactionKind = "consumption"
	PRODUCTION  TransactionKind = "production"
	PURCHASE    TransactionKind = "purchase"
	REFURBISH   TransactionKind = "refurbish"
	SOLD        TransactionKind = "sold"
)

type UnitOfMeasurement string
const (
	EACH       UnitOfMeasurement = "each"
	METER      UnitOfMeasurement = "meter"
	CENTIMETER UnitOfMeasurement = "centimeter"
	MILLIMITER UnitOfMeasurement = "millimiter"
	GRAM       UnitOfMeasurement = "gram"
	KILOGRAM   UnitOfMeasurement = "kilogram"
	METER_SQR  UnitOfMeasurement = "meter_sqr"
)

type OrderStatus string
const (
	PENDING   OrderStatus = "pending"
	STARTED   OrderStatus = "started"
	DONE      OrderStatus = "done"
	CANCELLED OrderStatus = "cancelled"
)

type BomStatus string
const (
	DRAFT      BomStatus = "draft"
	PUBLISHED  BomStatus = "published"
	DEPRECATED BomStatus = "deprecated"
)

type BaseItem struct {
	Id          int      `json:"id"`
	Name        string   `json:"name"`
	Sku         string   `json:"sku"`
	Kind        ItemKind `json:"kind"`
	Description string   `json:"description"`
	ThumbnailId *int     `json:"thumbnail_id"`
}

type Item struct {
	BaseItem
	Inventory      Inventory    `json:"inventory"`
	CurrentVersion *BaseItemVersion `json:"current_version"`
}

type Inventory struct {
	ItemId    int               `json:"item_id"`
	Unit      UnitOfMeasurement `json:"unit"`
	Available decimal.Decimal   `json:"available"`
	Reserved  decimal.Decimal   `json:"reserved"`
}

type BaseItemVersion struct {
	Id          int        `json:"id"`
	ItemId      int        `json:"item_id"`
	VersionCode string     `json:"version_code"`
	Notes       string     `json:"notes"`
	Status      BomStatus  `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	PublishedAt *time.Time `json:"published_at"`
	// TODO consider reference images
}

type ItemVersion struct {
	BaseItemVersion
	Children []BomLine `json:"children"`
}

type BomLine struct {
	Id              int             `json:"id"`
	ParentVersionId int             `json:"parent_version_id"`
	ChildItemId     int             `json:"child_item_id"`
	ChildVersionId  *int            `json:"child_version_id"`
	Quantity        decimal.Decimal `json:"quantity"`
	Position        int             `json:"position"`
}

/**
 * Params for the creation of a base item without a version
 */
type CreateBaseItemParams struct {
	Name        string            `json:"name"`
	Sku         string            `json:"sku"`
	Description string            `json:"description"`
	Unit        UnitOfMeasurement `json:"unit"`
	Available   decimal.Decimal   `json:"available"`
}

type CreateComponentParams struct {
	CreateBaseItemParams
}

type CreateAssemblyParams struct {
	CreateBaseItemParams
	CreateItemVersionParams
}

type CreateAssemblyChildParams struct {
	ItemId        int             `json:"item_id"`
	ItemVersionId *int            `json:"item_version_id"`
	Quantity      decimal.Decimal `json:"quantity"`
}

type CreateAssemblyResult struct {
	Item
	Version ItemVersion `json:"version"`
}

type AssemblyNode struct {
	BaseItem
	Inventory Inventory        `json:"inventory"`
	Version   *BaseItemVersion `json:"version"`
	Quantity  decimal.Decimal  `json:"quantity"`
	Position  int              `json:"position"`
	Children  []AssemblyNode   `json:"children"`
}

type BomRow struct {
	Item
	AssemblyItemId int
	ChildItemId int
	Quantity decimal.Decimal
}

type CreateItemVersionParams struct {
	VersionCode  string                      `json:"version_code"`
	VersionNotes string                      `json:"version_notes"`
	Children     []CreateAssemblyChildParams `json:"children"`
}

type Image struct {
	Id        int       `json:"id"`
	MimeType  string    `json:"mime_type"`
	SizeBytes int       `json:"size_bytes"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"created_at"`
}
