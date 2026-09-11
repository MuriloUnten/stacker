package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
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
 * - [ ] Enable write ahead logging (WAL)
 * - [/] Think about assembly versioning and implement it
 * - [/] Implement recursive read of BoM tree
 * - [ ] Implement migration system
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
	// NOTE: the _foreign_keys MUST STAY ON else new assemblies might cause recursions
	db, err := sql.Open("sqlite3", "file:database.db?_foreign_keys=on")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	bootstrapStmt := `
	create table if not exists item (
		item_id      integer primary key,
		name         text not null,
		sku          text unique not null,
		kind         text not null check (kind in ('a', 'c')), -- (assembly | component)
		description  text,
		thumbnail_id integer references image(image_id),

		current_version_id integer references item_version(version_id)  -- NULL for components, always
	);

	create table if not exists inventory (
		item_id   integer primary key references item(item_id),
		uom       text not null check (uom in ('each', 'meter', 'centimeter', 'millimiter', 'gram', 'kilogram', 'meter_sqr')),
		available decimal(18, 6) not null check (available >= 0),
		reserved  decimal(18, 6) not null check (reserved >= 0)
	);

	-- exclusively assemblies get item_versions (never components)
	create table if not exists item_version (
		version_id   integer primary key,
		item_id      integer not null references item(item_id),
		version_code text,
		notes        text,
		status       text not null default 'draft' check (status in ('draft', 'published', 'deprecated')),
		created_at   text not null default (datetime('now')),
		published_at text,

		unique(item_id, version_code)
	);

	create table if not exists bom_line (
		bom_line_id       integer primary key,
		parent_version_id integer not null references item_version(version_id),
		child_item_id     integer not null references item(item_id),
		child_version_id  integer references item_version(version_id), -- NULL for components, always
		child_item_kind   text not null check (child_item_kind in ('a', 'c')), -- we store this field here just to ensure child_version_id matches child kind
		quantity          decimal(18, 6) not null check (quantity > 0),
		position          integer not null check (position >= 0),

		check (
			(child_item_kind = 'a' and child_version_id is not null) or
			(child_item_kind = 'c' and child_version_id is null)
		)
	);

	create table if not exists manufacture_order (
		order_id           integer primary key,
		item_id            integer not null references item(item_id),
		item_version_id    integer not null references item_version(version_id),
		quantity_requested decimal(18, 6) not null check (quantity_requested > 0),
		quantity_completed decimal(18, 6) not null check (quantity_completed >= 0),
		status             text not null check (status in ('pending', 'started', 'done', 'cancelled')),
		created_at         text not null default (datetime('now')),
		started_at         text,
		completed_at       text
	);

	create table if not exists stock_transaction (
		transaction_id integer primary key,
		item_id        integer references item(item_id),
		order_id       integer references manufacture_order(order_id),
		quantity       decimal(18, 6) not null,
		kind           text not null check (kind in ('consumption', 'production', 'purchase', 'refurbish', 'sold')),
		created_at     text not null default (datetime('now'))
	);

	create table if not exists tracker (
		tracker_id integer primary key,
		item_id    integer not null references item(item_id),
		threshold  decimal(18, 6) not null check (threshold >= 0)
	);

	create table if not exists image (
		image_id integer primary key,
		content  blob not null
	);

	create table if not exists item_reference_image (
		item_reference_image_id integer primary key,
		image_id integer not null references image(image_id),
		item_id  integer not null references item(item_id),
		position integer not null check (position >= 0)
	);

	-- the following triggers ensure the database proper behavior of components and assemblies

	create trigger if not exists item_version_kind_guard_insert
	before insert on item_version
	for each row
	when (select kind from item where item_id = new.item_id) != 'a'
	begin
		select raise(abort, 'only assemblies may have item_version rows');
	end;

	create trigger if not exists item_version_kind_guard_update
	before update on item_version
	for each row
	when (select kind from item where item_id = new.item_id) != 'a'
	begin
		select raise(abort, 'only assemblies may have item_version rows');
	end;

	create trigger if not exists bom_line_set_child_kind_insert
	before insert on bom_line
	for each row
	begin
		select raise(abort, 'child_item_id does not exist')
		where not exists (select 1 from item where item_id = new.child_item_id);
	end;

	create trigger if not exists bom_line_derive_child_kind_insert
	after insert on bom_line
	for each row
	begin
		update bom_line
		set child_item_kind = (select kind from item where item_id = new.child_item_id)
		where bom_line_id = new.bom_line_id;
	end;
	`

	_, err = db.Exec(bootstrapStmt)
	if err != nil {
		log.Fatal(err)
	}

	var params CreateComponentParams
	params.name = "bolt"
	params.sku = "1234"
	params.description = "its just a bolt"
	params.unit = EACH
	params.available = decimal.NewFromInt(100)
	component, err := createComponent(db, params)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("component 1: %#v\n", component)

	params.name = "Metal sheet"
	params.sku = "ms001"
	params.unit = METER_SQR
	params.description = "black alluminum sheet"
	params.available = decimal.NewFromFloat(10.5)
	component, err = createComponent(db, params)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("component 2: %#v\n", component)

	comps, err := getComponents(db)
	if err != nil {
		log.Fatal(err) 
	}
	fmt.Println(comps)
	c, err := getComponentById(db, 2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#v\n", c)

	items, err := getItems(db)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(items)
	item, err := getItemById(db, 1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#v\n", item)
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
	id          int
	name        string
	sku         string
	kind        ItemKind
	description string
	thumbnailId *int
}

type Item struct {
	BaseItem
	inventory      Inventory
	currentVersion *ItemVersion
}

type Inventory struct {
	itemId    int
	unit      UnitOfMeasurement
	available decimal.Decimal
	reserved  decimal.Decimal
}

type BaseItemVersion struct {
	id          int
	itemId      int
	versionCode string
	notes       string
	status      BomStatus
	createdAt   time.Time
	publishedAt *time.Time
	// TODO consider reference images
}

type ItemVersion struct {
	BaseItemVersion
	children []BomLine
}

type BomLine struct {
	id              int
	parentVersionId int
	childItemId     int
	childVersionId  int
	quantity        decimal.Decimal
	position        int
}

/**
 * Params for the creation of a base item without a version
 */
type CreateBaseItemParams struct {
	name        string
	sku         string
	description string
	unit        UnitOfMeasurement
	available   decimal.Decimal
}

type CreateComponentParams struct {
	CreateBaseItemParams
}

type CreateAssemblyParams struct {
	CreateBaseItemParams
	CreateItemVersionParams
}

type CreateAssemblyChildParams struct {
	itemId        int
	itemVersionId *int
	quantity      decimal.Decimal
}

type CreateAssemblyResult struct {
	Item 
	version ItemVersion
}

type AssemblyNode struct {
	BaseItem
	inventory Inventory
	version   BaseItemVersion
	quantity  decimal.Decimal
	children  []AssemblyNode
}

type BomRow struct {
	Item
	assemblyItemId int
	childItemId int
	quantity decimal.Decimal
}

type CreateItemVersionParams struct {
	versionCode  string
	versionNotes string
	children     []CreateAssemblyChildParams
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

	row := itemStatement.QueryRow(params.name, params.sku, kind, params.description)
	err = row.Scan(
		&created.id,
		&created.name,
		&created.sku,
		&created.kind,
		&created.description,
		&created.thumbnailId,
	)
	if err != nil {
		return created, err
	}

	row = inventoryStatement.QueryRow(created.id, params.unit, params.available)
	err = row.Scan(
		&created.inventory.unit,
		&created.inventory.available,
		&created.inventory.reserved,
	)
	if err != nil {
		return created, err
	}

	created.currentVersion = nil
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
		&item.id,
		&item.name,
		&item.sku,
		&item.kind,
		&item.thumbnailId,
		&item.inventory.unit,
		&item.inventory.available,
		&item.inventory.reserved,
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
		&bomLine.id,
		&bomLine.parentVersionId,
		&bomLine.childItemId,
		&bomLine.childVersionId,
		&bomLine.quantity,
		&bomLine.position,
	)
	return err
}

func scanItemVersion(scanner RowScanner, iv *BaseItemVersion) error {
	err := scanner.Scan(
		&iv.id,
		&iv.itemId,
		&iv.versionCode,
		&iv.notes,
		&iv.status,
		&iv.createdAt,
		&iv.publishedAt,
	)
	return err
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

	itemParams := CreateBaseItemParams{
		name: params.name,
		available: params.available,
	}
	item, err := insertBaseItem(tx, ASSEMBLY, itemParams)
	if err != nil {
		return created, err
	}
	created.Item = item

	version, err := createItemVersion(tx, created.id, params.CreateItemVersionParams)
	if err != nil {
		return created, err
	}
	created.version = version
	
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

// func getAssemblyTree(db *sql.DB, versionId int) (AssemblyNode, error) {
// 	var root AssemblyNode
// 	v, err := getBaseItemVersion(db, versionId)
// 	if err != nil {
// 		return root, fmt.Errorf("fetch root node: %w", err)
// 	}
// 	root.version = v
//
// 	q := `
// 	WITH RECURSIVE bom_tree AS (
// 		SELECT
// 			iv.item_id as parent_item_id,
// 			bl.parent_version_id,
// 			bl.child_item_id,
// 			bl.quantity,
// 			bl.position,
// 			1 AS depth,
// 			'/' || iv.item_id || '/' || bl.child_item_id || '/' AS path
// 		FROM bom_line bl
// 		join item_version iv on bl.parent_version_id = iv.version_id
// 		WHERE bl.parent_version_id = ?
//
// 		UNION ALL
//
// 		SELECT
// 			iv.item_id as parent_item_id,
// 			bl.parent_version_id,
// 			bl.child_item_id,
// 			bl.quantity,
// 			bl.position,
// 			bt.depth + 1,
// 			bt.path || bl.child_item_id || '/'
// 		FROM bom_line bl
// 		join item_version iv on bl.parent_version_id = iv.version_id
// 		JOIN bom_tree bt ON iv.item_id = bt.child_item_id
// 		WHERE bt.path NOT LIKE '%/' || bl.child_item_id || '/%'
// 		  AND bt.depth < 50
// 	)
// 	SELECT
// 		bt.parent_version_id,
// 		bt.child_item_id,
// 		bt.quantity,
// 		it.item_id,
// 		it.name,
// 		it.kind,
// 		it.sku,
// 		it.description,
// 		it.thumbnail_id,
// 		it.current_version_id,
// 		iv.version_code,
// 		iv.notes,
// 		iv.status,
// 		iv.created_at,
// 		iv.published_at,
// 		inv.available,
// 		inv.reserved
// 	FROM bom_tree bt
// 	join item_version iv on bt.child_item_id = iv.version_id -- TODO check this (good chance this is wrong)
// 	JOIN item it ON bt.child_item_id = it.item_id
// 	JOIN inventory inv ON it.item_id = inv.item_id
// 	ORDER BY bt.depth;
// 	`
//
// 	rows, err := db.Query(q, versionId)
// 	if err != nil {
// 		return AssemblyNode{}, fmt.Errorf("query bom rows: %w", err)
// 	}
//
// 	childrenOf := make(map[int][]BomRow)
// 	for rows.Next() {
// 		var bomRow BomRow
// 		err := rows.Scan(
// 			&bomRow.assemblyItemId,
// 			&bomRow.childItemId,
// 			&bomRow.quantity,
// 			&bomRow.name,
// 			&bomRow.kind,
// 			&bomRow.thumbnailId,
// 			&bomRow.inventory.available,
// 			&bomRow.inventory.reserved,
// 		)
// 		if err != nil {
// 			return AssemblyNode{}, fmt.Errorf("scan bom rows: %w", err)
// 		}
// 		bomRow.Item.id = bomRow.childItemId
// 		childrenOf[bomRow.assemblyItemId] = append(childrenOf[bomRow.assemblyItemId], bomRow)
// 	}
// 	return buildAssemblyRecursive(root, 0, childrenOf, make(map[int]bool)), nil
// }
//
// // TODO fix this. There is a problem here because AssemblyNode has an ItemVersion,
// // but that does not have item data inside it
// func buildAssemblyRecursive(item Item, quantity decimal.Decimal, childrenOf map[int][]BomRow, visited map[int]bool) AssemblyNode {
// 	node := AssemblyNode{ItemVersion: item, quantity: quantity}
//
// 	if visited[item.id] {
// 		return node
// 	}
// 	visited[item.id] = true
// 	defer delete(visited, item.id) // allow the same item in sibling branches (diamond BOMs)
//
// 	for _, child := range childrenOf[item.id] {
// 		node.children = append(node.children, buildAssemblyRecursive(child.Item, child.quantity, childrenOf, visited))
// 	}
// 	return node
// }

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
	row := insertVersionStmt.QueryRow(itemId, params.versionCode, params.versionNotes)
	err = scanItemVersion(row, &created.BaseItemVersion)
	if err != nil {
		return created, err
	}

	// TODO fetch all items from params.children && verify that they are what they are supposed to be
	// and that the given versions are proper published versions
	for _, child := range(params.children) {
		if child.itemId == itemId {
			return created, errors.New("attempted to create self referencing assembly")
		}
		if child.quantity.LessThanOrEqual(decimal.NewFromInt(0)) {
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

	created.children = make([]BomLine, 0, len(params.children))
	var bomLine BomLine
	for i, child := range(params.children) {
		row := stmt.QueryRow(
			created.id,
			child.itemId,
			child.itemVersionId,
			child.quantity,
			i,
		)
		err := scanBomLine(row, &bomLine)
		if err != nil {
			return created, err
		}
		created.children = append(created.children, bomLine)
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

	v.children = make([]BomLine, 0, 10)
	var bomLine BomLine
	for (rows.Next()) {
		err = scanBomLine(rows, &bomLine)
		if err != nil {
			return v, err
		}
		v.children = append(v.children, bomLine)
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
