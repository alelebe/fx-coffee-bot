package main

import (
	"strings"
	"testing"
)

const filePath string = "./data/benugo.json"

func TestLoadBewerages(t *testing.T) {
	benugo, err := loadBewerages(filePath)
	if err != nil {
		t.Errorf("'%s' can't be loaded: %v", filePath, err)
		return
	}

	if false == strings.HasPrefix(benugo.Title, "Benugo") {
		t.Errorf("'%s' has wrong 'Title'", filePath)
	}

	size := len(benugo.Entry.Items)
	// total := numberOfDrinks(benugo.Entry.Items)
	// if total <= size {
	// 	t.Errorf("'%s' has weird number of drinks, size: %d, total: %d", filePath, size, total)
	// }

	all := benugo.Entry.getAllEntries()
	if len(all) <= size {
		t.Errorf("'%s' has weird number of _all_ drinks, all: %d, total: %d", filePath, len(all), size)
	}
}

func TestGetDrinkItem(t *testing.T) {
	benugo, err := loadBewerages(filePath)
	if err != nil {
		t.Errorf("'%s' can't be loaded: %v", filePath, err)
		return
	}

	all := benugo.Entry.getAllEntries()
	size := len(all)
	var item *Drink

	item = benugo.Entry.getDrinkByID(all[0].ID)
	if item == nil {
		t.Errorf("'%s' can't find first drink by ID: %s", filePath, all[0].ID)
		return
	}
	item = benugo.Entry.getDrinkByID(all[size-1].ID)
	if item == nil {
		t.Errorf("'%s' can't find last drink by ID: %s", filePath, all[size-1].ID)
		return
	}
	item = benugo.Entry.getDrinkByID("")
	if item != nil {
		t.Errorf("'%s' found item with empty ID: %+v", filePath, *item)
		return
	}
}