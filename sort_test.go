package topo_test

import (
	"cmp"
	"iter"
	"slices"
	"testing"
	"time"

	"github.com/crhntr/topo"
)

type Recipe struct {
	ID          int
	Ingredients []int
	CookTime    time.Duration
	IsBad       bool
}

func identifiers[T any, ID comparable](in []T, id func(T) ID) []ID {
	ids := make([]ID, len(in))
	for i := range in {
		ids[i] = id(in[i])
	}
	return ids
}

func (p Recipe) Edges() iter.Seq[int] { return slices.Values(p.Ingredients) }
func (p Recipe) Identifier() int      { return p.ID }

func TestSort(t *testing.T) {
	t.Run("no dependencies", func(t *testing.T) {
		t.Run("increasing", func(t *testing.T) {
			recipes := []Recipe{
				{ID: 1},
				{ID: 2},
				{ID: 3},
			}
			err := topo.Sort(recipes, Recipe.Identifier, Recipe.Edges)
			if err != nil {
				t.Fatal(err)
			}
			if exp, got := []int{1, 2, 3}, identifiers(recipes, Recipe.Identifier); !slices.Equal(got, exp) {
				t.Errorf("expected recipes to be sorted like %v, got %v", exp, got)
			}
		})
		t.Run("decreasing", func(t *testing.T) {
			recipes := []Recipe{
				{ID: 3},
				{ID: 2},
				{ID: 1},
			}
			slices.SortFunc(recipes, func(a, b Recipe) int {
				return cmp.Compare(a.ID, b.ID)
			})
			err := topo.Sort(recipes, Recipe.Identifier, Recipe.Edges)
			if err != nil {
				t.Fatal(err)
			}
			if exp, got := []int{1, 2, 3}, identifiers(recipes, Recipe.Identifier); !slices.Equal(got, exp) {
				t.Errorf("expected recipes to be sorted like %v, got %v", exp, got)
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
			err := topo.Sort(recipes, Recipe.Identifier, Recipe.Edges)
			if err != nil {
				t.Fatal(err)
			}
			if exp, got := []int{4, 5, 2, 3, 1}, identifiers(recipes, Recipe.Identifier); !slices.Equal(got, exp) {
				t.Errorf("expected recipes to be sorted like %v, got %v", exp, got)
			}
		})
		t.Run("self reference", func(t *testing.T) {
			recipes := []Recipe{
				{ID: 1, Ingredients: []int{1}},
			}
			err := topo.Sort(recipes, Recipe.Identifier, Recipe.Edges)
			if err == nil {
				t.Error("expected an error")
			}
		})
		t.Run("loop", func(t *testing.T) {
			recipes := []Recipe{
				{ID: 1, Ingredients: []int{2}},
				{ID: 2, Ingredients: []int{3}},
				{ID: 3, Ingredients: []int{1}},
			}
			err := topo.Sort(recipes, Recipe.Identifier, Recipe.Edges)
			if err == nil {
				t.Error("expected an error")
			}
		})
	})
}
