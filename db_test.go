package main

import (
	"testing"

	"github.com/shopspring/decimal"
)

func testStoreInit(t *testing.T) *Store {
	t.Helper()
	store := StoreInit(":memory:")
	t.Cleanup(func() { store.Deinit() })

	return store
}

func seedComponentFixture(t *testing.T, s *Store, name string) Item {
	t.Helper()

	var params CreateComponentParams
	params.Name = name
	params.Sku = name
	params.Description = name
	params.Unit = EACH
	params.Available = decimal.NewFromInt(100)
	component, err := createComponent(s.db, params)
	if err != nil {
		t.Fatal(err)
	}

	return component
}

func TestCreateComponent(t *testing.T) {
	s := testStoreInit(t)
	db := s.db

	var params CreateComponentParams
	params.Name = "10k tht resistor"
	params.Sku  = "1234"
	params.Description = ""
	params.Unit = EACH
	params.Available = decimal.NewFromInt(1000)

	item, err := createComponent(db, params)
	if err != nil {
		t.Errorf("failed to create fine component: %s", err.Error())
	}

	if item.Kind != COMPONENT {
		t.Errorf("expected kind == COMPONENT, got %s", item.Kind.String())
	}
}

func TestInsertMatchingSKU(t *testing.T) {
	s := testStoreInit(t)
	db := s.db

	var params CreateComponentParams
	params.Name = "10k tht resistor"
	params.Sku  = "1234"
	params.Description = ""
	params.Unit = EACH
	params.Available = decimal.NewFromInt(1000)

	_, err := createComponent(db, params)
	if err != nil {
		t.Errorf("failed to create fine component: %s", err.Error())
	}

	params.Name = "2.2k tht resistor"
	params.Sku  = "1234"
	params.Description = ""
	params.Unit = EACH
	params.Available = decimal.NewFromInt(1000)

	_, err = createComponent(db, params)
	if err == nil {
		t.Errorf("created item with matching sku")
	}
}

func TestCreateItemWithNegativeAvailable(t *testing.T) {
	s := testStoreInit(t)
	db := s.db

	var params CreateComponentParams
	params.Name = "10k tht resistor"
	params.Sku  = "1234"
	params.Description = ""
	params.Unit = EACH
	params.Available = decimal.NewFromInt(-1)

	_, err := createComponent(db, params)
	if err == nil {
		t.Errorf("created item with negative available stock")
	}
}

func TestCreateItemWithUomEachAndNonIntAvailable(t *testing.T) {
	s := testStoreInit(t)
	db := s.db

	var params CreateComponentParams
	params.Name = "bolt"
	params.Sku  = "1298"
	params.Description = "a bolt"
	params.Unit = EACH
	params.Available = decimal.NewFromFloat(10.5)

	_, err := createComponent(db, params)
	if err == nil {
		t.Errorf("should not be able to create item with uom EACH and non integer available stock")
	}
}

func TestCreateAssembly(t *testing.T) {
	s := testStoreInit(t)

	var params CreateComponentParams
	params.Name = "bolt"
	params.Sku = "1234"
	params.Description = "its just a bolt"
	params.Unit = EACH
	params.Available = decimal.NewFromInt(100)
	component, err := createComponent(s.db, params)
	if err != nil {
		t.Errorf("failed to create fine component: %s", err.Error())
	}
	boltId := component.Id

	params.Name = "Tire"
	params.Sku = "110324"
	params.Description = "Pirelli tire"
	params.Unit = EACH
	params.Available = decimal.NewFromInt(20)
	component, err = createComponent(s.db, params)
	if err != nil {
		t.Errorf("failed to create fine component: %s", err.Error())
	}
	tireId := component.Id

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
		t.Errorf("failed to create fine assembly: %s", err.Error())
	}

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
		t.Errorf("failed to create fine assembly: %s", err.Error())
	}

	_, err = getBom(s.db, carResult.Version.Id)
	if err != nil {
		t.Errorf("failed to read fine BOM: %s", err.Error())
	}
}

func TestCreateRecursiveItemVersion(t *testing.T) {
	s := testStoreInit(t)

	var params CreateComponentParams
	params.Name = "bolt"
	params.Sku = "1234"
	params.Description = "its just a bolt"
	params.Unit = EACH
	params.Available = decimal.NewFromInt(100)
	component, err := createComponent(s.db, params)
	if err != nil {
		t.Errorf("failed to create fine component: %s", err.Error())
	}
	boltId := component.Id

	params.Name = "Tire"
	params.Sku = "110324"
	params.Description = "Pirelli tire"
	params.Unit = EACH
	params.Available = decimal.NewFromInt(20)
	component, err = createComponent(s.db, params)
	if err != nil {
		t.Errorf("failed to create fine component: %s", err.Error())
	}
	tireId := component.Id

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
		t.Errorf("failed to create fine assembly: %s", err.Error())
	}

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
		t.Errorf("failed to create fine assembly: %s", err.Error())
	}

	_, err = getBom(s.db, carResult.Version.Id)
	if err != nil {
		t.Errorf("failed to read fine BOM: %s", err.Error())
	}
}

func TestAttemptComponentVersion(t *testing.T) {
	s := testStoreInit(t)

	component := seedComponentFixture(t, s, "bolt")
	child     := seedComponentFixture(t, s, "child component")

	var params CreateItemVersionParams
	params.VersionCode = "1.0"
	params.VersionNotes = "first revision"
	params.Children = []CreateAssemblyChildParams{
		{
			ItemId: child.Id,
			ItemVersionId: nil,
			Quantity: decimal.NewFromInt(10),
		},
	}

	_, err :=createItemVersionWrapper(s.db, component.Id, params)
	if err == nil {
		t.Errorf("should not be able to create version for component")
	}
}
