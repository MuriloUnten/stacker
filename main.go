package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/shopspring/decimal"
)

/**
 * TODO
 * - [ ] implement endpoints for /items/{id}/versions
 * - [x] use decimal types instead of decimal.Decimal's (https://github.com/shopspring/decimal)
 * - [x] Position in bom_line
 * - [ ] Treat possible active version on item related functions
 * - [ ] Implement Store
 * - [ ] Prepare all statements beforehand
 * - [ ] Read about prepared statements in transactions
 * - [x] Enable write ahead logging (WAL)
 * - [/] Think about assembly versioning and implement it
 * - [/] Implement recursive read of BoM tree
 * - [/] Implement migration system
 * - [ ] Implement connection pool
 * - [ ] Add better errors
 * - [ ] think about having timestamps in more things
 * - [ ] Write testing framework
 * - [ ] Add Tests
 * - [ ] Consider using sqlc
 * - [ ] Study and think about context and timeouts
 * - [ ] Add more fields to image table (mime_type, width, height, etc)
 * - [ ] Add optional image input to item creation functions
 * - [ ] Think about how to implement a tracker scanner goroutine
 * - [ ] Implement manufacture orders
 * - [ ] Think about manufacture orders having another state maybe called refurbished or something that restores inventory
 * - [ ] Rewatch the McMaster-Carr video (https://www.youtube.com/watch?v=-Ln-8QM8KhQ)
 */
func main() {
	s := StoreInit("db/database.db")
	defer s.Deinit()

	var params CreateComponentParams
	params.Name = "bolt"
	params.Sku = "1234"
	params.Description = "its just a bolt"
	params.Unit = EACH
	params.Available = decimal.NewFromInt(100)
	component, err := createComponent(s.db, params)
	if err != nil {
		log.Fatal(err)
	}
	boltId := component.Id
	fmt.Printf("bolt: %#v\n", component)

	// params.name = "Metal sheet"
	// params.sku = "ms001"
	// params.unit = METER_SQR
	// params.description = "black alluminum sheet"
	// params.available = decimal.NewFromFloat(10.5)
	// component, err = createComponent(s.db, params)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("component 2: %#v\n", component)
	//
	// comps, err := getComponents(s.db)
	// if err != nil {
	// 	log.Fatal(err) 
	// }
	// fmt.Println(comps)
	// c, err := getComponentById(s.db, 2)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("%#v\n", c)
	//
	// items, err := getItems(s.db)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(items)
	// item, err := getItemById(s.db, 1)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("%#v\n", item)

	params.Name = "Tire"
	params.Sku = "110324"
	params.Description = "Pirelli tire"
	params.Unit = EACH
	params.Available = decimal.NewFromInt(20)
	component, err = createComponent(s.db, params)
	if err != nil {
		log.Fatal(err)
	}
	tireId := component.Id
	fmt.Printf("tire: %#v\n", component)

	var wheelParams CreateAssemblyParams
	wheelParams.Name = "Wheel"
	wheelParams.Sku  = "1010"
	wheelParams.Description = "17 inch wheel"
	wheelParams.Unit = EACH
	wheelParams.Available = decimal.NewFromInt(0)
	wheelParams.VersionCode = "1.0"
	wheelParams.VersionNotes = ""
	wheelParams.Children = []CreateAssemblyChildParams{
		{
			ItemId: tireId,
			ItemVersionId: nil,
			Quantity: decimal.NewFromInt(1),
		},
		{
			ItemId: boltId,
			ItemVersionId: nil,
			Quantity: decimal.NewFromInt(4),
		},
	}

	wheelResult, err := createAssembly(s.db, wheelParams)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Wheel: %#v\n", wheelResult)

	var carParams CreateAssemblyParams
	carParams.Name = "Car"
	carParams.Sku  = "1111"
	carParams.Description = "Random ass car"
	carParams.Unit = EACH
	carParams.Available = decimal.NewFromInt(0)
	carParams.VersionCode = "1.0"
	carParams.VersionNotes = ""
	carParams.Children = []CreateAssemblyChildParams{
		{
			ItemId: wheelResult.Id,
			ItemVersionId: &wheelResult.Version.Id,
			Quantity: decimal.NewFromInt(4),
		},
	}
	carResult, err := createAssembly(s.db, carParams)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Car: %#v\n", carResult)
	fmt.Println("")

	carBom, err := getBom(s.db, carResult.Version.Id)
	if err != nil {
		log.Fatal(err)
	}

	data, err := json.MarshalIndent(carBom, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	
	fmt.Println(string(data))
}

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

