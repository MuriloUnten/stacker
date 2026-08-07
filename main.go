package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

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

	componentParams := CreateComponentParams{
		name: "Parafuso Allen M3 10mm",
		available: 100,
	}

	component, err := createComponent(db, componentParams)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("component 1: %#v\n", component)

	componentParams.name = "Porca 1"
	componentParams.available = 3
	component, err = createComponent(db, componentParams)
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
}

type ItemKind string
func (k ItemKind) String() string {
	return string(k)
}
const (
	COMPONENT ItemKind = "component"
	ASSEMBLY  ItemKind = "assembly"
)

type BaseItem struct {
	id      int
	name    string
	kind    ItemKind
	imageId *int

	available int
	reserved  int
}

type Item struct {
	BaseItem
	children []BoMLine
}

type BoMLine struct {
	assemblyItemId int
	childItemId    int
	ammount        int
}

type Component struct {
	BaseItem
}

func (c Component) toItem() Item {
	var item Item
	item.BaseItem = c.BaseItem
	item.children = nil
	return item
}

type Assembly struct {
	Item
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

/**
 * Params for the creation of the base item shared between Assembly and Component
 * (which is basically the Component itself)
 */
type CreateGenericItemParams struct {
	name      string
	kind      ItemKind
	available int
}

/**
 * When item is a component, children must be empty or nil
 * All children must already exist
 * 
 */
func createItem(db *sql.DB, params CreateItemParams) (Item, error) {
	var item Item
	if params.kind == COMPONENT {
		if len(params.children) != 0 {
			return item, errors.New("children must be nil or empty when creating a component")
		}
		componentParams := CreateComponentParams{
			name: params.name,
			available: params.available,
		}
		component, err := createComponent(db, componentParams) 
		return component.toItem(), err
	}

	assemblyParams := CreateAssemblyParams{
		name: params.name,
		available: params.available,
		children: params.children,
	}
	assembly, err := createAssembly(db, assemblyParams)
	return assembly.Item, err
}

func createComponent(db *sql.DB, params CreateComponentParams) (Component, error) {
	created := Component{}

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
	created.BaseItem, err = insertBaseItem(tx, itemParams)
	if err != nil {
		return created, err
	}

	if err = tx.Commit(); err != nil {
		return created, err
	}

	return created, err
}

func getComponents(db *sql.DB) ([]Component, error) {
	kind := COMPONENT
	q := `
	select it.item_id, it.name, it.kind, it.image_id, inv.available, inv.reserved
	from item it join inventory inv on it.item_id = inv.item_id
	where it.kind = ?
	`
	rows, err := db.Query(q, kind.String())
	if err != nil {
		return nil, err
	}

	comps := make([]Component, 0, 10)
	return scanComponents(rows,comps)
}

func getComponentById(db *sql.DB, id int) (Component, error) {
	q := `
	select it.item_id, it.name, it.kind, it.image_id, inv.available, inv.reserved
	from item it join inventory inv on it.item_id = inv.item_id
	where it.item_id = ?`

	row := db.QueryRow(q, id)
	var c Component
	err := scanComponent(row, &c)
	return c, err
}

/**
 * allows for scan from both *sql.Row and *sql.Rows
 * always assumes all fields, and in the order they are defined
 */
type RowScanner interface {
	Scan(...any) error
}
 
func scanComponent(scanner RowScanner, comp *Component) error {
	err := scanner.Scan(&comp.id, &comp.name, &comp.kind, &comp.imageId, &comp.available, &comp.reserved)
	return err
}

func scanComponents(rows *sql.Rows, comps []Component) ([]Component, error) {
	if comps == nil {
		return nil, errors.New("must give pre-allocated slice")
	}
	var c Component
	for rows.Next() {
		err := scanComponent(rows, &c)
		if err != nil {
			return comps, err
		}
		comps = append(comps, c)
	}
	if err := rows.Err(); err != nil {
		return comps, err
	}
	return comps, nil
}

func insertBaseItem(tx *sql.Tx, params CreateGenericItemParams) (BaseItem, error) {
	created := BaseItem{}
	kind     := COMPONENT
	reserved := 0

	insertItem      := `
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

	row := itemStatement.QueryRow(params.name, kind)
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

func createAssembly(db *sql.DB, params CreateAssemblyParams) (Assembly, error) {
	// insert assembly
	// check if children reference parent
	// insert relations to bom_line

	var created Assembly

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
	created.Item.BaseItem = item

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
