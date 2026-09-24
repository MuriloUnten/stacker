package main

import (
	"bytes"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"image"
    _ "image/jpeg"
    _ "image/png"
	"io"
	"log"
	"slices"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/shopspring/decimal"
)

func parseSQLiteTimestamp(str string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", str)
}

//go:embed db/migrations/*.sql
var embedMigrations embed.FS

const (
	migrationsPath = "db/migrations"
	sqliteOptions  = "?_journal_mode=WAL&_foreign_keys=on"
)

type Store struct {
	db *sql.DB
}

func StoreInit(path string) *Store {
	db, err := Connect("file:" + path + sqliteOptions)
	if err != nil {
		log.Fatal(err)
	}

	return &Store{
		db: db,
	}
}

func (s *Store) Deinit() {
	s.db.Close()
}

func Connect(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}

	// TODO figure out better way of disabling logging only for testing purposes
	goose.SetLogger(goose.NopLogger())

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, err
	}

	if err := goose.Up(db, migrationsPath); err != nil {
		return nil, err
	}

	return db, nil
}

func getItems(db *sql.DB) ([]Item, error) {
	q := `
	select it.item_id, it.name, it.sku, it.kind, it.thumbnail_id,
	inv.uom, inv.available, inv.reserved,
	iv.version_id, iv.version_code, iv.notes, iv.status, iv.created_at, iv.published_at
	from item it join inventory inv on it.item_id = inv.item_id
	left join item_version iv on it.current_version_id = iv.version_id
	`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}

	return scanItems(rows, 10)
}

func getItemById(db *sql.DB, id int) (Item, error) {
	q := `
	select it.item_id, it.name, it.sku, it.kind, it.thumbnail_id,
	inv.uom, inv.available, inv.reserved,
	iv.version_id, iv.version_code, iv.notes, iv.status, iv.created_at, iv.published_at
	from item it join inventory inv on it.item_id = inv.item_id
	left join item_version iv on it.current_version_id = iv.version_id
	where it.item_id = ?`

	row := db.QueryRow(q, id)
	var item Item
	err := scanItem(row, &item)
	return item, err
}

func getItemsWithKind(db *sql.DB, kind ItemKind) ([]Item, error) {
	q := `
	select it.item_id, it.name, it.sku, it.kind, it.thumbnail_id,
	inv.uom, inv.available, inv.reserved,
	iv.version_id, iv.version_code, iv.notes, iv.status, iv.created_at, iv.published_at
	from item it join inventory inv on it.item_id = inv.item_id
	left join item_version iv on it.current_version_id = iv.version_id
	where it.kind = ?
	`
	rows, err := db.Query(q, kind.String())
	if err != nil {
		return nil, err
	}

	return scanItems(rows, 10)
}

func getItemByIdWithKind(db *sql.DB, id int, kind ItemKind) (Item, error) {
	q := `
	select it.item_id, it.name, it.sku, it.kind, it.thumbnail_id,
	inv.uom, inv.available, inv.reserved,
	iv.version_id, iv.version_code, iv.notes, iv.status, iv.created_at, iv.published_at
	from item it join inventory inv on it.item_id = inv.item_id
	left join item_version iv on it.current_version_id = iv.version_id
	where it.item_id = ? and it.kind = ?
	`

	row := db.QueryRow(q, id, kind.String())
	var item Item
	err := scanItem(row, &item)
	return item, err
}

func getItemsByIdList(tx *sql.Tx, itemIds []int) ([]Item, error) {
	if len(itemIds) == 0 {
		return []Item{}, nil
	}

	q := `
	select it.item_id, it.name, it.sku, it.kind, it.thumbnail_id,
	inv.uom, inv.available, inv.reserved,
	iv.version_id, iv.version_code, iv.notes, iv.status, iv.created_at, iv.published_at
	from item it join inventory inv on it.item_id = inv.item_id
	left join item_version iv on it.current_version_id = iv.version_id
	where it.item_id in (%s)
	`

	args := make([]any, 0, len(itemIds))
	args = append(args, itemIds[0])
	var placeholders strings.Builder
	placeholders.WriteString("?")
	for i := 1; i < len(itemIds); i++ {
		placeholders.WriteString(", ?")
		args = append(args, itemIds[i])
	}
	itemsSelect := fmt.Sprintf(q, placeholders.String())

	itemsStmt, err := tx.Prepare(itemsSelect)
	if err != nil {
		return nil, err
	}
	rows, err := itemsStmt.Query(args...)
	if err != nil {
		return nil, err
	}

	return scanItems(rows, len(itemIds))
}

func insertBaseItem(tx *sql.Tx, kind ItemKind, params CreateBaseItemParams) (Item, error) {
	created  := Item{}

	if params.Unit == EACH && !params.Available.IsInteger() {
		return created, errors.New("item with kind EACH cannot have non integer available stock")
	}

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
	var (
		versionId   sql.NullInt64
		versionCode sql.NullString
		notes       sql.NullString
		status      sql.NullString
		createdAt   sql.NullString
		publishedAt sql.NullString
	)

	err := scanner.Scan(
		&item.Id,
		&item.Name,
		&item.Sku,
		&item.Kind,
		&item.ThumbnailId,
		&item.Inventory.Unit,
		&item.Inventory.Available,
		&item.Inventory.Reserved,
		&versionId,
		&versionCode,
		&notes,
		&status,
		&createdAt,
		&publishedAt,
	)
	if err != nil {
		return err
	}

	if !versionId.Valid {
		item.CurrentVersion = nil
		return nil
	}

	// if version_id field is not null,
	// then all other version fields must not be null
	if !versionCode.Valid || !notes.Valid || !status.Valid || !createdAt.Valid || !publishedAt.Valid {
		return errors.New("inconsistent state: there is a version_id, but other fields are missing")
	}

	createdAtTime, err := parseSQLiteTimestamp(createdAt.String)
	if err != nil {
		return err
	}
	publishedAtTime, err := parseSQLiteTimestamp(publishedAt.String)
	if err != nil {
		return err
	}

	item.CurrentVersion = &BaseItemVersion{
		Id: int(versionId.Int64),
		ItemId: item.Id,
		VersionCode: versionCode.String,
		Notes: notes.String,
		Status: BomStatus(status.String),
		CreatedAt: createdAtTime,
		PublishedAt: &publishedAtTime,
	}

	return nil
}

func scanItems(rows *sql.Rows, initialCapacity int) ([]Item, error) {
	capacity := max(initialCapacity, 5)
	items := make([]Item, 0, capacity)

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

	iv.CreatedAt, err = parseSQLiteTimestamp(createdAt)
	if err != nil {
		return err
	}
	if publishedAt.Valid {
		publishedAtTime, err := parseSQLiteTimestamp(publishedAt.String)
		if err != nil {
			return err
		}
		iv.PublishedAt = &publishedAtTime
	}

	return nil
}

func scanItemVersions(rows *sql.Rows, initialCapacity int) ([]BaseItemVersion, error) {
	capacity := max(initialCapacity, 5)
	versions := make([]BaseItemVersion, 0, capacity)

	var version BaseItemVersion
	for rows.Next() {
		err := scanItemVersion(rows, &version)
		if err != nil {
			return versions, err
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return versions, err
	}
	return versions, nil
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
				if t, err := parseSQLiteTimestamp(createdAt.String); err == nil {
					bv.CreatedAt = t
				}
			}
			if publishedAt.Valid {
				if t, err := parseSQLiteTimestamp(publishedAt.String); err == nil {
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

	err = validateAssemblyChildren(tx, params.Children, itemId)
	if err != nil {
		return created, err
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

func createItemVersionWrapper(db *sql.DB, itemId int, params CreateItemVersionParams) (ItemVersion, error) {
	var iv ItemVersion
	tx, err := db.Begin()
	if err != nil {
		return iv, err
	}
	defer tx.Rollback()

	iv, err = createItemVersion(tx, itemId, params)
	if err != nil {
		return iv, err
	}

	err = tx.Commit()
	return iv, err
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
	where parent_version_id = ?
	order by position asc
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

func getItemVersionsByItem(db *sql.DB, itemId int) ([]BaseItemVersion, error) {
	q := `
	select version_id, item_id, version_code, notes, status, created_at, published_at
	from item_version v where v.item_id = ?
	order by created_at desc
	`

	stmt, err := db.Prepare(q)
	if err != nil {
		return []BaseItemVersion{}, err
	}
	rows, err := stmt.Query(itemId)
	if err != nil {
		return []BaseItemVersion{}, err
	}

	return scanItemVersions(rows, 10)
}

func getItemVersionsByIdList(tx *sql.Tx, versionIds []int) ([]BaseItemVersion, error) {
	if len(versionIds) == 0 {
		return []BaseItemVersion{}, nil
	}

	q := `
	select version_id, item_id, version_code, notes, status, created_at, published_at
	from item_version v where v.version_id in (%s)
	order by created_at desc
	`
	args := make([]any, 0, len(versionIds))
	args = append(args, versionIds[0])
	var placeholders strings.Builder
	placeholders.WriteString("?")
	for i := 1; i < len(versionIds); i++ {
		placeholders.WriteString(", ?")
		args = append(args, versionIds[i])
	}
	itemsSelect := fmt.Sprintf(q, placeholders.String())

	itemsStmt, err := tx.Prepare(itemsSelect)
	if err != nil {
		return nil, err
	}
	rows, err := itemsStmt.Query(args...)
	if err != nil {
		return nil, err
	}

	return scanItemVersions(rows, len(versionIds))
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

func publishVersion(db *sql.DB, versionId int) error {
	iv, err := getItemVersionById(db, versionId)
	if err != nil {
		return err
	}

	if iv.Status != DRAFT {
		return errors.New("can't publish non draft version")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	versionQuery := `
	update item_version set status = 'published', published_at = datetime('now')
	where version_id = ?
	returning item_id
	`
	itemQuery := `
	update item set current_version_id = ?
	where item_id = ?
	`

	versionStmt, err := tx.Prepare(versionQuery)
	if err != nil {
		return err
	}
	itemStmt, err := tx.Prepare(itemQuery)
	if err != nil {
		return err
	}

	row := versionStmt.QueryRow(versionId)
	var itemId int
	err = row.Scan(&itemId)
	if err != nil {
		return err
	}

	result, err := itemStmt.Exec(versionId, itemId)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return errors.New("no rows affected")
	}

	return tx.Commit()
}

func deprecateVersion(db *sql.DB, versionId int) error {
	iv, err := getItemVersionById(db, versionId)
	if err != nil {
		return err
	}

	if iv.Status != PUBLISHED {
		return errors.New("can't deprecate non published version")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	versionQuery := `
	update item_version set status = 'deprecated'
	where version_id = ?
	returning item_id
	`
	publishedItemVersionsQuery := `
	select version_id from item_version
	where item_id = ? and status = 'published'
	order by published_at desc limit 1
	`
	itemQuery := `
	update item set current_version_id = ?
	where item_id = ?
	`

	versionStmt, err := tx.Prepare(versionQuery)
	if err != nil {
		return err
	}
	publishedVersions, err := tx.Prepare(publishedItemVersionsQuery)
	if err != nil {
		return err
	}
	itemStmt, err := tx.Prepare(itemQuery)
	if err != nil {
		return err
	}

	row := versionStmt.QueryRow(versionId)
	var itemId int
	err = row.Scan(&itemId)
	if err != nil {
		return err
	}

	row = publishedVersions.QueryRow(itemId)
	var newPublishedVersionId *int
	err = row.Scan(newPublishedVersionId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			newPublishedVersionId = nil
		} else {
			return err
		}
	}

	result, err := itemStmt.Exec(newPublishedVersionId, itemId)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return errors.New("no rows affected")
	}

	return tx.Commit()
}

func validateAssemblyChildren(tx *sql.Tx, children []CreateAssemblyChildParams, parentId int) error {
	if len(children) == 0 {
		return errors.New("there must be at least one child")
	}
	itemIds := make([]int, 0 , len(children))
	versionIds := make([]int, 0 , len(children))
	for _, child := range(children) {
		if !slices.Contains(itemIds, child.ItemId) {
			itemIds = append(itemIds, child.ItemId)
		}
		if child.ItemVersionId != nil {
			versionId := *child.ItemVersionId
			if !slices.Contains(versionIds, versionId) {
				versionIds = append(versionIds, versionId)
			}
		}

		if child.ItemId == parentId {
			return errors.New("attempted to create self referencing assembly")
		}
		if child.Quantity.LessThanOrEqual(decimal.NewFromInt(0)) {
			return errors.New("assembly child must have quantity greater than zero")
		}
	}

	items, err := getItemsByIdList(tx, itemIds)
	if err != nil {
		return err
	}
	if len(items) != len(itemIds) {
		return errors.New("invalid child: item id does not exist")
	}
	itemMap := make(map[int]Item)
	for _, it := range(items) {
		itemMap[it.Id] = it
	}

	versions, err := getItemVersionsByIdList(tx, versionIds)
	if err != nil {
		return err
	}
	if len(versions) != len(versionIds) {
		return errors.New("invalid child: version id does not exist")
	}
	versionMap := make(map[int]BaseItemVersion)
	for _, v := range(versions) {
		versionMap[v.Id] = v
	}

	for _, child := range(children) {
		childItem, ok := itemMap[child.ItemId]
		if !ok {
			return errors.New("child error: item not found")
		}

		if childItem.Inventory.Unit == EACH && !child.Quantity.IsInteger() {
			return errors.New("child error: item unit is each and quantity is not integer")
		}

		if childItem.Kind == COMPONENT && child.ItemVersionId != nil {
			return errors.New("child error: child is component with version")
		}
		if childItem.Kind == ASSEMBLY && child.ItemVersionId == nil {
			return errors.New("child error: assembly child without its version")
		}

		if child.ItemVersionId != nil {
			childVersion, ok := versionMap[*child.ItemVersionId]
			if !ok {
				return errors.New("child error: item version not found")
			}
			if childVersion.ItemId == parentId {
				return errors.New("child error: attemping to create recursive version")
			}
			if childVersion.Status != PUBLISHED {
				return errors.New("child error: item version is not published")
			}
			if childVersion.ItemId != child.ItemId {
				return errors.New("child error: item and version do not match")
			}
		}
	}

	return nil
}

func getItemThumbnail(db *sql.DB, itemId int) (Image, io.ReadCloser, error) {
	var img Image
	var content []byte
	q := `
	select im.image_id, im.mime_type, im.size_bytes, im.width, im.height,
	im.created_at, im.content
	from item it join image im on it.thumbnail_id = im.image_id
	where it.item_id = ?
	`

	stmt, err := db.Prepare(q)
	if err != nil {
		return img, nil, err
	}

	row := stmt.QueryRow(itemId)
	err = row.Scan(
		&img.Id,
		&img.MimeType,
		&img.SizeBytes,
		&img.Width,
		&img.Height,
		&img.CreatedAt,
		&content,
	)
	if err != nil {
		return img, nil, err
	}

	return img, io.NopCloser(bytes.NewReader(content)), nil
}

func uploadItemThumbnail(db *sql.DB, itemId int, content io.Reader) (Image, error) {
	var img Image
	data, err := io.ReadAll(io.LimitReader(content, 10<<20))

	tx, err := db.Begin()
	if err != nil {
		return img, err
	}
	defer tx.Rollback()

	img.MimeType = "image/jpeg"
	config, _, err :=image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return img, err
	}
	img.Width = config.Width
	img.Height = config.Height

	q := `
	insert into image (mime_type, size_bytes, width, height, content)
	values (?, ?, ?, ?, ?)
	returning image_id, created_at
	`
	insertStmt, err := tx.Prepare(q)
	if err != nil {
		return img, err
	}

	createdAtString := ""
	row := insertStmt.QueryRow(
		img.MimeType,
		len(data),
		img.Width,
		img.Height,
		data,
	)
	err = row.Scan(
		&img.Id,
		&createdAtString,
	)
	if err != nil {
		return img, err
	}
	img.CreatedAt, err = parseSQLiteTimestamp(createdAtString)
	if err != nil {
		return img, err
	}

	itemUpdate := `update item set thumbnail_id = ? where item_id = ?`
	updatedStmt, err := tx.Prepare(itemUpdate)
	if err != nil {
		return img, err
	}

	updateResult, err := updatedStmt.Exec(img.Id, itemId)
	if err != nil {
		return img, err
	}

	affected, err := updateResult.RowsAffected()
	if err != nil {
		return img, err
	}
	if affected != 1 {
		return img, errors.New("image upload error: failed to update item thumbnail id")
	}

	return img, tx.Commit()
}

func getImageById(db *sql.DB, imageId int) (Image, io.ReadCloser, error) {
	var img Image
	var content []byte
	q := `
	select im.image_id, im.mime_type, im.size_bytes, im.width, im.height,
	im.created_at, im.content
	from image im where it.item_id = ?
	`

	stmt, err := db.Prepare(q)
	if err != nil {
		return img, nil, err
	}

	row := stmt.QueryRow(imageId)
	err = row.Scan(
		&img.Id,
		&img.MimeType,
		&img.SizeBytes,
		&img.Width,
		&img.Height,
		&img.CreatedAt,
		&content,
	)
	if err != nil {
		return img, nil, err
	}

	return img, io.NopCloser(bytes.NewReader(content)), nil
}
