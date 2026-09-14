package main

import (
	"database/sql"
	"encoding/json"
	"errors"
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

func getItems(db *sql.DB) ([]Item, error) {
	q := `
	select it.item_id, it.name, it.sku, it.kind, it.thumbnail_id,
	inv.uom, inv.available, inv.reserved
	from item it join inventory inv on it.item_id = inv.item_id
	`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, 10)
	return scanItems(rows, items)
}

func getItemById(db *sql.DB, id int) (Item, error) {
	q := `
	select it.item_id, it.name, it.sku, it.kind, it.thumbnail_id,
	inv.uom, inv.available, inv.reserved
	from item it join inventory inv on it.item_id = inv.item_id
	where it.item_id = ?`

	row := db.QueryRow(q, id)
	var item Item
	err := scanItem(row, &item)
	return item, err
}

func getItemsWithKind(db *sql.DB, kind ItemKind) ([]Item, error) {
	q := `
	select it.item_id, it.name, it.sku, it.kind, it.thumbnail_id,
	inv.uom, inv.available, inv.reserved
	from item it join inventory inv on it.item_id = inv.item_id
	where it.kind = ?
	`
	rows, err := db.Query(q, kind.String())
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, 10)
	return scanItems(rows, items)
}

func getItemByIdWithKind(db *sql.DB, id int, kind ItemKind) (Item, error) {
	q := `
	select it.item_id, it.name, it.sku, it.kind, it.thumbnail_id, inv.uom, inv.available, inv.reserved
	from item it join inventory inv on it.item_id = inv.item_id
	where it.item_id = ? and it.kind = ?
	`

	row := db.QueryRow(q, id, kind.String())
	var item Item
	err := scanItem(row, &item)
	return item, err
}

func insertBaseItem(tx *sql.Tx, kind ItemKind, params CreateBaseItemParams) (Item, error) {
	created  := Item{}

	insertItem := `
	insert into item (name, sku, kind, description)
	values (?, ?, ?, ?) returning item_id, name, sku, kind, description, thumbnail_id
	`

	insertInventory := `
	insert into inventory (item_id, uom, available, reserved)
	values (?, ?, ?, 0) returning uom, available, reserved
	`

	itemStatement, err := tx.Prepare(insertItem)
	if err != nil {
		return created, err
	}
	inventoryStatement, err := tx.Prepare(insertInventory)
	if err != nil {
		return created, err
	}

	row := itemStatement.QueryRow(params.Name, params.Sku, kind, params.Description)
	err = row.Scan(
		&created.Id,
		&created.Name,
		&created.Sku,
		&created.Kind,
		&created.Description,
		&created.ThumbnailId,
	)
	if err != nil {
		return created, err
	}

	row = inventoryStatement.QueryRow(created.Id, params.Unit, params.Available)
	err = row.Scan(
		&created.Inventory.Unit,
		&created.Inventory.Available,
		&created.Inventory.Reserved,
	)
	if err != nil {
		return created, err
	}

	created.CurrentVersion = nil
	return created, nil
}

/**
 * allows for scan from both *sql.Row and *sql.Rows
 * always assumes all fields, and in the order they are defined
 */
type RowScanner interface {
	Scan(...any) error
}

func scanItem(scanner RowScanner, item *Item) error {
	err := scanner.Scan(
		&item.Id,
		&item.Name,
		&item.Sku,
		&item.Kind,
		&item.ThumbnailId,
		&item.Inventory.Unit,
		&item.Inventory.Available,
		&item.Inventory.Reserved,
	)
	return err
}

func scanItems(rows *sql.Rows, items []Item) ([]Item, error) {
	if items == nil {
		return nil, errors.New("must give pre-allocated slice")
	}
	var item Item
	for rows.Next() {
		err := scanItem(rows, &item)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return items, err
	}
	return items, nil
}

func scanBomLine(scanner RowScanner, bomLine *BomLine) error {
	err := scanner.Scan(
		&bomLine.Id,
		&bomLine.ParentVersionId,
		&bomLine.ChildItemId,
		&bomLine.ChildVersionId,
		&bomLine.Quantity,
		&bomLine.Position,
	)
	return err
}

func scanItemVersion(scanner RowScanner, iv *BaseItemVersion) error {
	var createdAt   string
	var publishedAt sql.NullString
	err := scanner.Scan(
		&iv.Id,
		&iv.ItemId,
		&iv.VersionCode,
		&iv.Notes,
		&iv.Status,
		&createdAt,
		&publishedAt,
	)
	if err != nil {
		return err
	}

	iv.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
	if err != nil {
		return err
	}
	if publishedAt.Valid {
		*iv.PublishedAt, err = time.Parse("2006-01-02 15:04:05", publishedAt.String)
		if err != nil {
			return err
		}
	}

	return nil
}

func createComponent(db *sql.DB, params CreateComponentParams) (Item, error) {
	created := Item{}

	tx, err := db.Begin()
	if err != nil {
		return created, err
	}
	defer tx.Rollback()

	created, err = insertBaseItem(tx, COMPONENT, params.CreateBaseItemParams)
	if err != nil {
		return created, err
	}

	if err = tx.Commit(); err != nil {
		return created, err
	}

	return created, err
}

func getComponents(db *sql.DB) ([]Item, error) {
	return getItemsWithKind(db, COMPONENT)
}

func getComponentById(db *sql.DB, id int) (Item, error) {
	return getItemByIdWithKind(db, id, COMPONENT)
}

func createAssembly(db *sql.DB, params CreateAssemblyParams) (CreateAssemblyResult, error) {
	var created CreateAssemblyResult

	tx, err := db.Begin()
	if err != nil {
		return created, err
	}
	defer tx.Rollback()

	item, err := insertBaseItem(tx, ASSEMBLY, params.CreateBaseItemParams)
	if err != nil {
		return created, err
	}
	created.Item = item

	version, err := createItemVersion(tx, created.Id, params.CreateItemVersionParams)
	if err != nil {
		return created, err
	}
	created.Version = version
	
	if err = tx.Commit(); err != nil {
		return created, err
	}

	return created, nil
}

func getAssemblies(db *sql.DB) ([]Item, error) {
	return getItemsWithKind(db, ASSEMBLY)
}

func getAssemblyById(db *sql.DB, id int) (Item, error) {
	return getItemByIdWithKind(db, id, ASSEMBLY)
}

func getBom(db *sql.DB, versionId int) (AssemblyNode, error) {
	var root AssemblyNode
	v, err := getBaseItemVersion(db, versionId)
	if err != nil {
		return root, fmt.Errorf("fetch root node: %w", err)
	}
	root.Version = &v

	rootItem, err := getItemById(db, v.ItemId)
	if err != nil {
		return root, fmt.Errorf("fetch root item: %w", err)
	}
	root.BaseItem = rootItem.BaseItem
	root.Inventory = rootItem.Inventory


	q := `
	WITH RECURSIVE bom_tree AS (
		SELECT
			bl.parent_version_id,
			bl.child_item_id,
			bl.child_version_id,
			bl.quantity,
			bl.position,
			1 AS depth,
			'/' || iv.item_id || '/' || bl.child_item_id || '/' AS path
		FROM bom_line bl
		JOIN item_version iv ON bl.parent_version_id = iv.version_id
		WHERE bl.parent_version_id = ?

		UNION ALL

		SELECT
			bl.parent_version_id,
			bl.child_item_id,
			bl.child_version_id,
			bl.quantity,
			bl.position,
			bt.depth + 1,
			bt.path || bl.child_item_id || '/'
		FROM bom_line bl
		-- recurse using the specific child VERSION from the parent row,
		-- not just any version of the same item
		JOIN bom_tree bt ON bl.parent_version_id = bt.child_version_id
		WHERE bt.depth < 50
		  AND bt.path NOT LIKE '%/' || bl.child_item_id || '/%'
	)
	SELECT
		bt.parent_version_id,
		bt.child_item_id,
		bt.child_version_id,
		bt.quantity,
		bt.position,
		it.name,
		it.sku,
		it.kind,
		it.description,
		it.thumbnail_id,
		it.current_version_id,
		iv.version_code,
		iv.notes,
		iv.status,
		iv.created_at,
		iv.published_at,
		inv.uom,
		inv.available,
		inv.reserved
	FROM bom_tree bt
	JOIN item it ON bt.child_item_id = it.item_id
	-- components have NULL child_version_id / no item_version row at all
	LEFT JOIN item_version iv ON bt.child_version_id = iv.version_id
	JOIN inventory inv ON it.item_id = inv.item_id
	ORDER BY bt.depth, bt.parent_version_id, bt.position;
	`

	rows, err := db.Query(q, versionId)
	if err != nil {
		return AssemblyNode{}, fmt.Errorf("query bom rows: %w", err)
	}
	defer rows.Close()

	// Keyed by parent_version_id, NOT item_id — a single item can have
	// multiple versions, each with a different BOM.
	childrenOf := make(map[int][]AssemblyNode)

	for rows.Next() {
		var (
			parentVersionId int
			childItemId     int
			childVersionId  *int
			quantity        decimal.Decimal
			position        int

			name             string
			sku              string
			kind             string
			description      string
			thumbnailId      *int
			currentVersionId *int

			versionCode sql.NullString
			notes       sql.NullString
			status      sql.NullString
			createdAt   sql.NullString
			publishedAt sql.NullString

			uom       UnitOfMeasurement
			available decimal.Decimal
			reserved  decimal.Decimal
		)

		if err := rows.Scan(
			&parentVersionId, &childItemId, &childVersionId, &quantity, &position,
			&name, &sku, &kind, &description, &thumbnailId, &currentVersionId,
			&versionCode, &notes, &status, &createdAt, &publishedAt,
			&uom, &available, &reserved,
		); err != nil {
			return AssemblyNode{}, fmt.Errorf("scan bom rows: %w", err)
		}

		node := AssemblyNode{
			Quantity: quantity,
			Position: position,
			BaseItem: BaseItem{
				Id:          childItemId,
				Name:        name,
				Sku:         sku,
				Kind:        ItemKind(kind),
				Description: description,
				ThumbnailId: thumbnailId,
			},
		}
		// if currentVersionId.Valid {
		// 	id := int(currentVersionId.Int64)
		// 	node.item.currentVersionId = &id
		// }
		node.Inventory = Inventory{
			ItemId:    childItemId,
			Unit:      uom,
			Available: available,
			Reserved:  reserved,
		}

		// only assemblies (kind='a') get a version populated
		if childVersionId != nil {
			bv := BaseItemVersion{
				Id:          *childVersionId,
				ItemId:      childItemId,
				VersionCode: versionCode.String,
				Notes:       notes.String,
				Status:      BomStatus(status.String),
			}
			if createdAt.Valid {
				if t, err := time.Parse("2006-01-02 15:04:05", createdAt.String); err == nil {
					bv.CreatedAt = t
				}
			}
			if publishedAt.Valid {
				if t, err := time.Parse("2006-01-02 15:04:05", publishedAt.String); err == nil {
					bv.PublishedAt = &t
				}
			}
			node.Version = &bv
		}

		childrenOf[parentVersionId] = append(childrenOf[parentVersionId], node)
	}
	if err := rows.Err(); err != nil {
		return AssemblyNode{}, fmt.Errorf("iterate bom rows: %w", err)
	}

	buildAssemblyRecursive(&root, childrenOf)
	return root, nil
}

func buildAssemblyRecursive(node *AssemblyNode, childrenOf map[int][]AssemblyNode) {
	if node.Version == nil {
		// components have no version_id, hence no bom_line rows of their own
		return
	}
	children := childrenOf[node.Version.Id]
	for i := range children {
		buildAssemblyRecursive(&children[i], childrenOf)
	}
	node.Children = children
}

func createItemVersion(tx *sql.Tx, itemId int, params CreateItemVersionParams) (ItemVersion, error) {
	var created ItemVersion
	insertVersionQuery := `
	insert into item_version (item_id, version_code, notes)
	values (?, ?, ?)
	returning version_id, item_id, version_code, notes, status, created_at, published_at
	`
	insertVersionStmt, err := tx.Prepare(insertVersionQuery)
	if err != nil {
		return created, err
	}
	row := insertVersionStmt.QueryRow(itemId, params.VersionCode, params.VersionNotes)
	err = scanItemVersion(row, &created.BaseItemVersion)
	if err != nil {
		return created, err
	}

	// TODO fetch all items from params.children && verify that they are what they are supposed to be
	// and that the given versions are proper published versions
	for _, child := range(params.Children) {
		if child.ItemId == itemId {
			return created, errors.New("attempted to create self referencing assembly")
		}
		if child.Quantity.LessThanOrEqual(decimal.NewFromInt(0)) {
			return created, errors.New("assembly child must have quantity greater than zero")
		}
	}

	q := `insert into bom_line (parent_version_id, child_item_id, child_version_id, quantity, position)
	values (?, ?, ?, ?, ?)
	returning bom_line_id, parent_version_id, child_item_id, child_version_id, quantity, position`
	stmt, err := tx.Prepare(q)
	if err != nil {
		return created, err
	}

	created.Children = make([]BomLine, 0, len(params.Children))
	var bomLine BomLine
	for i, child := range(params.Children) {
		row := stmt.QueryRow(
			created.Id,
			child.ItemId,
			child.ItemVersionId,
			child.Quantity,
			i,
		)
		err := scanBomLine(row, &bomLine)
		if err != nil {
			return created, err
		}
		created.Children = append(created.Children, bomLine)
	}

	return created, nil
}

func getItemVersionById(db *sql.DB, versionId int) (ItemVersion, error) {
	var v ItemVersion
	var baseItemVersion BaseItemVersion
	baseItemVersion, err := getBaseItemVersion(db, versionId)
	if err != nil {
		return v, err
	}
	v.BaseItemVersion = baseItemVersion

	q := `
	select bom_line_id, parent_version_id, child_item_id, child_version_id, quantity, position from bom_line
	order by position asc
	where parent_version_id = ?
	`
	stmt, err := db.Prepare(q)
	if err != nil {
		return v, err
	}
	rows, err := stmt.Query(versionId)
	if err != nil {
		return v, err
	}

	v.Children = make([]BomLine, 0, 10)
	var bomLine BomLine
	for (rows.Next()) {
		err = scanBomLine(rows, &bomLine)
		if err != nil {
			return v, err
		}
		v.Children = append(v.Children, bomLine)
	}

	return v, nil
}

// TODO implement
func getItemVersionsByItem(tx *sql.DB, itemId int) ([]ItemVersion, error) {
	// versionQuery := `
	// select version_id, item_id, version_code, notes, status, created_at, published_at from item_version v 
	// where v.item_id = ?
	// order by created_at desc
	// `
	return nil, fmt.Errorf("not implemented")
}

func getBaseItemVersion(db *sql.DB, versionId int) (BaseItemVersion, error) {
	var v BaseItemVersion
	q := `
	select version_id, item_id, version_code, notes, status, created_at, published_at
	from item_version v where v.version_id = ?
	`
	stmt, err := db.Prepare(q)
	if err != nil {
		return v, err
	}
	row := stmt.QueryRow(versionId)
	err = scanItemVersion(row, &v)
	if err != nil {
		return v, err
	}

	return v, nil
}
