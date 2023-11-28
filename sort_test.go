package topological

import (
	"slices"
	"testing"
)

func TestSort(t *testing.T) {
	t.Run("no dependencies", func(t *testing.T) {
		t.Run("increasing", func(t *testing.T) {
			recipes := []Recipe{
				{ID: 1},
				{ID: 2},
				{ID: 3},
			}
			sorted, err := Sort(recipes, func(r Recipe) int { return r.ID }, func(r Recipe) []int { return r.Edges() })
			if err != nil {
				t.Fatal(err)
			}
			if len(sorted) != len(recipes) {
				t.Fatalf("expected %d recipes, got %d", len(recipes), len(sorted))
			}
			if exp := []int{1, 2, 3}; !slices.Equal(identifiers(sorted), exp) {
				t.Errorf("expected recipes to be sorted like %v, got %v", exp, identifiers(sorted))
			}
		})
		t.Run("decreasing", func(t *testing.T) {
			recipes := []Recipe{
				{ID: 3},
				{ID: 2},
				{ID: 1},
			}
			sorted, err := Sort(recipes, func(r Recipe) int { return r.ID }, func(r Recipe) []int { return r.Edges() })
			if err != nil {
				t.Fatal(err)
			}
			if exp := []int{1, 2, 3}; !slices.Equal(identifiers(sorted), exp) {
				t.Errorf("expected recipes to be sorted like %v, got %v", exp, identifiers(sorted))
			}
		})
	})

	t.Run("with dependencies", func(t *testing.T) {
		t.Run("no cycles", func(t *testing.T) {
			recipes := []Recipe{
				{ID: 1, Ingredients: []int{2, 3}},
				{ID: 2, Ingredients: []int{4, 5}},
				{ID: 3, Ingredients: []int{4, 5}},
				{ID: 4, Ingredients: []int{}},
				{ID: 5, Ingredients: []int{}},
			}
			sorted, err := Sort(recipes, func(r Recipe) int { return r.ID }, func(r Recipe) []int { return r.Edges() })
			if err != nil {
				t.Fatal(err)
			}
			if exp := []int{4, 5, 2, 3, 1}; !slices.Equal(identifiers(sorted), exp) {
				t.Errorf("expected recipes to be sorted like %v, got %v", exp, identifiers(sorted))
			}
		})
		t.Run("self reference", func(t *testing.T) {
			recipes := []Recipe{
				{ID: 1, Ingredients: []int{1}},
			}
			_, err := Sort(recipes, func(r Recipe) int { return r.ID }, func(r Recipe) []int { return r.Edges() })
			if err == nil {
				t.Error(err)
			}
		})
		t.Run("loop", func(t *testing.T) {
			recipes := []Recipe{
				{ID: 1, Ingredients: []int{2}},
				{ID: 2, Ingredients: []int{3}},
				{ID: 3, Ingredients: []int{1}},
			}
			_, err := Sort(recipes, func(r Recipe) int { return r.ID }, func(r Recipe) []int { return r.Edges() })
			if err == nil {
				t.Error(err)
			}
		})
	})
}

func identifiers(recipes []Recipe) []int {
	ids := make([]int, len(recipes))
	for i, r := range recipes {
		ids[i] = r.ID
	}
	return ids
}

type Recipe struct {
	ID          int
	Ingredients []int
}

func (p Recipe) Edges() []int {
	return slices.Clone(p.Ingredients)
}

func (p Recipe) Identifier() int {
	return p.ID
}
