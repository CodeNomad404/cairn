package model

import (
	"reflect"
	"testing"
)

func TestAccountingExpenseCategoryValid(t *testing.T) {
	for category := AccountingExpenseCategoryFood; category <= AccountingExpenseCategoryOther; category++ {
		if !category.Valid() {
			t.Fatalf("category %d rejected", category)
		}
	}
	for _, category := range []AccountingExpenseCategory{0, 9, -1} {
		if category.Valid() {
			t.Fatalf("category %d accepted", category)
		}
	}
}

func TestAccountingModelsDeclareNoJSONTags(t *testing.T) {
	models := []any{AccountingBook{}, AccountingExpense{}}
	for _, value := range models {
		typ := reflect.TypeOf(value)
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if tag := field.Tag.Get("json"); tag != "" {
				t.Errorf("%s.%s declares JSON tag %q", typ.Name(), field.Name, tag)
			}
		}
	}
}
