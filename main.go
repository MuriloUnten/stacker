package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

/**
 * TODO
 * - Implement Store
 * - Prepare all statements beforehand
 * - Read about prepared statements in transactions
 * - Enable write ahead logging
 * - Think about assembly versioning and implement it
 * - Implement recursive read of BoM tree
 * - Implement migration system
 * - Implement connection pool
 * - Add better errors
 * - think about having timestamps in more things
 * - Test stuff
 * - Think about ensuring all statement call will take the right ammount of arguments
 * - Study and think about context and timeouts
 * - Add more fields to image table (mime_type, width, height, etc)
 * - Add optional image input to item creation functions
 * - Think about how to implement a tracker scanner goroutine
 * - Implement manufacture orders
 * - Think about manufacture orders having another state maybe called refurbished or something that restores inventory
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
		item_id  integer primary key,
		name     text not null,
		kind     text not null check (kind in ('assembly', 'component')),
		image_id integer references image(image_id)
	);

	create table if not exists inventory (
		item_id   integer primary key references item(item_id),
		available integer not null check (available >= 0),
		reserved  integer not null check (reserved >= 0)
	);

	create table if not exists bom_line (
		assembly_item_id integer not null references item(item_id),
		child_item_id    integer not null references item(item_id),
		ammount          integer not null check (ammount > 0),

		primary key (assembly_item_id, child_item_id)
	);

	create table if not exists manufacture_order (
		order_id          integer primary key,
		item_id           integer not null references item(item_id),
		ammount_requested integer not null check (ammount_requested > 0),
		ammount_completed integer not null check (ammount_completed >= 0),
		status            text not null check (status in ('pending', 'started', 'done', 'cancelled')),
		created_at        datetime not null,
		started_at        datetime,
		completed_at      datetime
	);

	create table if not exists tracker (
		tracker_id integer primary key,
		item_id    integer not null references item(item_id),
		threshold  integer not null check (threshold >= 0)
	);

	create table if not exists image (
		image_id integer primary key,
		content  blob not null
	);
	`

	_, err = db.Exec(bootstrapStmt)
	if err != nil {
		log.Fatal(err)
	}

	// componentParams := CreateComponentParams{
	// 	name: "Parafuso Allen M3 10mm",
	// 	available: 100,
	// }
	//
	// component, err := createComponent(db, componentParams)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("component 1: %#v\n", component)
	//
	// componentParams.name = "Porca 1"
	// componentParams.available = 3
	// component, err = createComponent(db, componentParams)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("component 2: %#v\n", component)

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
	COMPONENT ItemKind = "component"
	ASSEMBLY  ItemKind = "assembly"
)

type Item struct {
	id      int
	name    string
	kind    ItemKind
	imageId *int

	available int
	reserved  int
}

type BoMLine struct {
	assemblyItemId int
	childItemId    int
	ammount        int
}

type CreateComponentParams struct {
	name      string
	available int
}

type CreateItemChildParams struct {
	itemId int
	ammount int
}

type CreateItemParams struct {
	name      string
	kind      ItemKind
	available int
	children  []CreateItemChildParams
}

type CreateAssemblyParams struct {
	name      string
	available int
	children  []CreateItemChildParams
}

type CreateAssemblyResult struct {
	Item
	children []BoMLine
}

/**
 * Params for the creation of the base item shared between Assembly and Component
 * (which is basically the Component itself)
 */
type CreateGenericItemParams struct {
	name      string
	kind      ItemKind
	available int
}

type AssemblyNode struct {
	Item
	ammount int
	children []AssemblyNode
}

type BoMRow struct {
	Item
	assemblyItemId int
	childItemId int
	ammount int
}

/**
 * allows for scan from both *sql.Row and *sql.Rows
 * always assumes all fields, and in the order they are defined
 */
type RowScanner interface {
	Scan(...any) error
}

/**
 * When item is a component, children must be empty or nil
 * All children must already exist
 * 
 */
func createItem(db *sql.DB, params CreateItemParams) (Item, error) {
	var item Item
	switch params.kind {
	case COMPONENT:
		if len(params.children) != 0 {
			return item, errors.New("children must be nil or empty when creating a component")
		}
		componentParams := CreateComponentParams{
			name: params.name,
			available: params.available,
		}
		component, err := createComponent(db, componentParams) 
		return component, err
	case ASSEMBLY:
		assemblyParams := CreateAssemblyParams{
			name: params.name,
			available: params.available,
			children: params.children,
		}
		assembly, err := createAssembly(db, assemblyParams)
		return assembly.Item, err
	default:
		return item, errors.New("invalid kind")
	}
}

func getItems(db *sql.DB) ([]Item, error) {
	q := `
	select it.item_id, it.name, it.kind, it.image_id, inv.available, inv.reserved
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
	select it.item_id, it.name, it.kind, it.image_id, inv.available, inv.reserved
	from item it join inventory inv on it.item_id = inv.item_id
	where it.item_id = ?`

	row := db.QueryRow(q, id)
	var item Item
	err := scanItem(row, &item)
	return item, err
}

func getItemsWithKind(db *sql.DB, kind ItemKind) ([]Item, error) {
	q := `
	select it.item_id, it.name, it.kind, it.image_id, inv.available, inv.reserved
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
	select it.item_id, it.name, it.kind, it.image_id, inv.available, inv.reserved
	from item it join inventory inv on it.item_id = inv.item_id
	where it.item_id = ? and it.kind = ?
	`

	row := db.QueryRow(q, id, kind.String())
	var item Item
	err := scanItem(row, &item)
	return item, err
}

func insertBaseItem(tx *sql.Tx, params CreateGenericItemParams) (Item, error) {
	created  := Item{}
	reserved := 0

	insertItem := `
	insert into item (name, kind)
	values (?, ?) returning item_id, name, kind, image_id
	`

	insertInventory := `
	insert into inventory (item_id, available, reserved)
	values (?, ?, ?) returning available, reserved
	`

	itemStatement, err := tx.Prepare(insertItem)
	if err != nil {
		return created, err
	}
	inventoryStatement, err := tx.Prepare(insertInventory)
	if err != nil {
		return created, err
	}

	row := itemStatement.QueryRow(params.name, params.kind)
	err = row.Scan(&created.id, &created.name, &created.kind, &created.imageId)
	if err != nil {
		return created, err
	}

	row = inventoryStatement.QueryRow(created.id, params.available, reserved)
	err = row.Scan(&created.available, &created.reserved)
	if err != nil {
		return created, err
	}

	return created, nil
}

func scanItem(scanner RowScanner, item *Item) error {
	err := scanner.Scan(&item.id, &item.name, &item.kind, &item.imageId, &item.available, &item.reserved)
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

func createComponent(db *sql.DB, params CreateComponentParams) (Item, error) {
	created := Item{}

	tx, err := db.Begin()
	if err != nil {
		return created, err
	}
	defer tx.Rollback()

	itemParams := CreateGenericItemParams{
		name: params.name,
		kind: COMPONENT,
		available: params.available,
	}
	created, err = insertBaseItem(tx, itemParams)
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

	itemParams := CreateGenericItemParams{
		name: params.name,
		kind: ASSEMBLY,
		available: params.available,
	}
	item, err := insertBaseItem(tx, itemParams)
	if err != nil {
		return created, err
	}
	created.Item = item

	for _, child := range(params.children) {
		if child.itemId == created.id {
			return created, errors.New("attempted to create self referencing assembly")
		}
		if child.ammount <= 0 {
			return created, errors.New("assembly child must have ammount of at least 1")
		}
	}

	q := `insert into bom_line (assembly_item_id, child_item_id, ammount)
	values (?, ?, ?) returning assembly_item_id, child_item_id, ammount`
	stmt, err := db.Prepare(q)
	if err != nil {
		return created, err
	}

	created.children = make([]BoMLine, 0, len(params.children))
	var bomLine BoMLine
	for _, child := range(params.children) {
		row := stmt.QueryRow(created.id, child.itemId, child.ammount)
		err = row.Scan(&bomLine.assemblyItemId, &bomLine.childItemId, &bomLine.ammount)
		if err != nil {
			return created, err
		}
		created.children = append(created.children, bomLine)
	}
	
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

func getAssemblyTree(db *sql.DB, id int) (AssemblyNode, error) {
	root, err := getAssemblyById(db, id)
	if err != nil {
		return AssemblyNode{}, fmt.Errorf("fetch root node: %w", err)
	}

	q := `
	WITH RECURSIVE bom_tree AS (
		SELECT
			bl.assembly_item_id,
			bl.child_item_id,
			bl.ammount,
			1 AS depth,
			'/' || bl.assembly_item_id || '/' || bl.child_item_id || '/' AS path
		FROM bom_line bl
		WHERE bl.assembly_item_id = ?

		UNION ALL

		SELECT
			bl.assembly_item_id,
			bl.child_item_id,
			bl.ammount,
			bt.depth + 1,
			bt.path || bl.child_item_id || '/'
		FROM bom_line bl
		JOIN bom_tree bt ON bl.assembly_item_id = bt.child_item_id
		WHERE bt.path NOT LIKE '%/' || bl.child_item_id || '/%'
		  AND bt.depth < 50
	)
	SELECT
		bt.assembly_item_id,
		bt.child_item_id,
		bt.ammount,
		it.name,
		it.kind,
		it.image_id,
		inv.available,
		inv.reserved
	FROM bom_tree bt
	JOIN item it ON it.item_id = bt.child_item_id
	JOIN inventory inv ON it.item_id = inv.item_id
	ORDER BY bt.depth;
	`

	rows, err := db.Query(q, id)
	if err != nil {
		return AssemblyNode{}, fmt.Errorf("query bom rows: %w", err)
	}

	childrenOf := make(map[int][]BoMRow)
	for rows.Next() {
		var bomRow BoMRow
		err := rows.Scan(
			&bomRow.assemblyItemId,
			&bomRow.childItemId,
			&bomRow.ammount,
			&bomRow.name,
			&bomRow.kind,
			&bomRow.imageId,
			&bomRow.available,
			&bomRow.reserved,
		)
		if err != nil {
			return AssemblyNode{}, fmt.Errorf("scan bom rows: %w", err)
		}
		bomRow.Item.id = bomRow.childItemId
		childrenOf[bomRow.assemblyItemId] = append(childrenOf[bomRow.assemblyItemId], bomRow)
	}
	return buildAssemblyRecursive(root, 0, childrenOf, make(map[int]bool)), nil
}

func buildAssemblyRecursive(item Item, ammount int, childrenOf map[int][]BoMRow, visited map[int]bool) AssemblyNode {
	node := AssemblyNode{Item: item, ammount: ammount}

	if visited[item.id] {
		return node
	}
	visited[item.id] = true
	defer delete(visited, item.id) // allow the same item in sibling branches (diamond BOMs)

	for _, child := range childrenOf[item.id] {
		node.children = append(node.children, buildAssemblyRecursive(child.Item, child.ammount, childrenOf, visited))
	}
	return node
}
