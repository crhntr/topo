package topological_test

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/crhntr/topological"
)

type Recipe struct {
	ID          int
	Ingredients []int
	CookTime    time.Duration
	IsBad       bool
}

func identifiers[T any, ID cmp.Ordered](in []T, id func(T) ID) []ID {
	ids := make([]ID, len(in))
	for i := range in {
		ids[i] = id(in[i])
	}
	return ids
}

func (p Recipe) Edges() []int {
	return slices.Clone(p.Ingredients)
}

func (p Recipe) Identifier() int {
	return p.ID
}

func TestSort(t *testing.T) {
	t.Run("no dependencies", func(t *testing.T) {
		t.Run("increasing", func(t *testing.T) {
			recipes := []Recipe{
				{ID: 1},
				{ID: 2},
				{ID: 3},
			}
			err := topological.Sort(recipes, Recipe.Identifier, Recipe.Edges)
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
			err := topological.Sort(recipes, Recipe.Identifier, Recipe.Edges)
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
			err := topological.Sort(recipes, Recipe.Identifier, Recipe.Edges)
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
			err := topological.Sort(recipes, Recipe.Identifier, Recipe.Edges)
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
			err := topological.Sort(recipes, Recipe.Identifier, Recipe.Edges)
			if err == nil {
				t.Error("expected an error")
			}
		})
	})
}

type Ingredient struct {
	RecipeID int
	Done     bool
}

func (i Ingredient) RecipeIdentifier() int {
	return i.RecipeID
}

var _ topological.TaskFunc[Recipe, Ingredient] = Recipe.Cook

var ErrBadRecipe = errors.New("bad recipe")

func (p Recipe) Cook(ctx context.Context, in []Ingredient) (Ingredient, error) {
	if p.IsBad {
		return Ingredient{}, ErrBadRecipe
	}
	for i, requirement := range p.Ingredients {
		if i >= len(in) || in[i].RecipeID != requirement {
			return Ingredient{}, fmt.Errorf("missing requirement %d", requirement)
		}
	}
	select {
	case <-ctx.Done():
		return Ingredient{RecipeID: p.ID}, ctx.Err()
	case <-time.After(p.CookTime):
		return Ingredient{RecipeID: p.ID, Done: true}, nil
	}
}

func TestTasks(t *testing.T) {
	t.Run("no dependencies", func(t *testing.T) {
		ctx := context.Background()
		s := time.Second / 6
		recipes := []Recipe{
			{ID: 1, CookTime: s},
			{ID: 2, CookTime: s},
			{ID: 3, CookTime: s},
			{ID: 4, CookTime: s},
			{ID: 5, CookTime: s},
			{ID: 6, CookTime: s},
		}
		ctx, cancel := context.WithTimeout(ctx, s*time.Duration(len(recipes))/2)
		t.Cleanup(cancel)
		err := topological.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("self reference", func(t *testing.T) {
		recipes := []Recipe{
			{ID: 1, Ingredients: []int{1}},
		}
		ctx := context.Background()
		err := topological.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if err == nil {
			t.Error("expected and error")
		}
	})
	t.Run("one task", func(t *testing.T) {
		recipes := []Recipe{
			{ID: 1, CookTime: time.Second / 20},
		}
		ctx := context.Background()
		err := topological.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if err != nil {
			t.Error("expected and error")
		}
	})
	t.Run("two requirements", func(t *testing.T) {
		sleep := time.Second / 20
		recipes := []Recipe{
			{ID: 1, CookTime: sleep},
			{ID: 2, CookTime: sleep, Ingredients: []int{1, 3}},
			{ID: 3, CookTime: sleep},
		}
		ctx := context.Background()
		err := topological.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("context canceled", func(t *testing.T) {
		recipes := []Recipe{
			{ID: 1, CookTime: time.Minute},
			{ID: 2, CookTime: time.Minute},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second/20)
		t.Cleanup(cancel)
		err := topological.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected deadline exceeded error, got: %v", err)
		}
	})
	t.Run("task failure", func(t *testing.T) {
		recipes := []Recipe{
			{ID: 1, CookTime: time.Minute, Ingredients: []int{2}},
			{ID: 2, CookTime: time.Minute, IsBad: true},
		}
		ctx := context.Background()
		err := topological.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if !errors.Is(err, ErrBadRecipe) {
			t.Errorf("expected bad recipe error, got: %v", err)
		}
	})
}
