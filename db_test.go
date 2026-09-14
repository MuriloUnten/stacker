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

func TestCreateItemRecursive(t *testing.T) {
	// TODO implement
}
