package webdriver

import (
	"testing"

	"github.com/tebeka/selenium"
)

type interactionElement struct {
	selenium.WebElement
	selected bool
	clicks   int
	children []selenium.WebElement
}

func (e *interactionElement) IsSelected() (bool, error) { return e.selected, nil }
func (e *interactionElement) Click() error {
	e.clicks++
	e.selected = !e.selected
	return nil
}
func (e *interactionElement) FindElements(by, value string) ([]selenium.WebElement, error) {
	return e.children, nil
}

func TestCheckAndUncheckAreIdempotent(t *testing.T) {
	service := &service{}
	element := &interactionElement{}
	if err := service.Check(element); err != nil {
		t.Fatal(err)
	}
	if err := service.Check(element); err != nil {
		t.Fatal(err)
	}
	if element.clicks != 1 {
		t.Fatalf("check clicks=%d, wanted 1", element.clicks)
	}
	if err := service.Uncheck(element); err != nil {
		t.Fatal(err)
	}
	if err := service.Uncheck(element); err != nil {
		t.Fatal(err)
	}
	if element.clicks != 2 {
		t.Fatalf("total clicks=%d, wanted 2", element.clicks)
	}
}

func TestSelectOptionClicksFirstMatch(t *testing.T) {
	service := &service{}
	option := &interactionElement{}
	selectElement := &interactionElement{children: []selenium.WebElement{option}}
	if err := service.SelectOption(selectElement, "US"); err != nil {
		t.Fatal(err)
	}
	if option.clicks != 1 {
		t.Fatalf("option clicks=%d, wanted 1", option.clicks)
	}
}
