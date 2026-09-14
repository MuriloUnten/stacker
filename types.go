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
	Id          int
	Name        string
	Sku         string
	Kind        ItemKind
	Description string
	ThumbnailId *int
}

type Item struct {
	BaseItem
	Inventory      Inventory
	CurrentVersion *ItemVersion
}

type Inventory struct {
	ItemId    int
	Unit      UnitOfMeasurement
	Available decimal.Decimal
	Reserved  decimal.Decimal
}

type BaseItemVersion struct {
	Id          int
	ItemId      int
	VersionCode string
	Notes       string
	Status      BomStatus
	CreatedAt   time.Time
	PublishedAt *time.Time
	// TODO consider reference images
}

type ItemVersion struct {
	BaseItemVersion
	Children []BomLine
}

type BomLine struct {
	Id              int
	ParentVersionId int
	ChildItemId     int
	ChildVersionId  *int
	Quantity        decimal.Decimal
	Position        int
}

/**
 * Params for the creation of a base item without a version
 */
type CreateBaseItemParams struct {
	Name        string
	Sku         string
	Description string
	Unit        UnitOfMeasurement
	Available   decimal.Decimal
}

type CreateComponentParams struct {
	CreateBaseItemParams
}

type CreateAssemblyParams struct {
	CreateBaseItemParams
	CreateItemVersionParams
}

type CreateAssemblyChildParams struct {
	ItemId        int
	ItemVersionId *int
	Quantity      decimal.Decimal
}

type CreateAssemblyResult struct {
	Item 
	Version ItemVersion
}

type AssemblyNode struct {
	BaseItem
	Inventory Inventory
	Version   *BaseItemVersion
	Quantity  decimal.Decimal
	Position  int
	Children  []AssemblyNode
}

type BomRow struct {
	Item
	AssemblyItemId int
	ChildItemId int
	Quantity decimal.Decimal
}

type CreateItemVersionParams struct {
	VersionCode  string
	VersionNotes string
	Children     []CreateAssemblyChildParams
}

