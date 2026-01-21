package topo_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/crhntr/topo"
)

type Ingredient struct {
	RecipeID int
	Done     bool
}

func (i Ingredient) RecipeIdentifier() int {
	return i.RecipeID
}

var _ topo.TaskFunc[Recipe, Ingredient] = Recipe.Cook

var ErrBadRecipe = errors.New("bad recipe")

func (p Recipe) Cook(ctx context.Context, in []Ingredient) (Ingredient, error) {
	if p.IsBad {
		return Ingredient{}, ErrBadRecipe
	}
	for i, requirement := range p.Ingredients {
		if i >= len(in) || in[i].RecipeID != requirement {
			return Ingredient{}, fmt.Errorf("missing requirement %d: got %v", requirement, in)
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
		ctx := t.Context()
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
		_, err := topo.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("self reference", func(t *testing.T) {
		recipes := []Recipe{
			{ID: 1, Ingredients: []int{1}},
		}
		ctx := t.Context()
		_, err := topo.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if err == nil {
			t.Error("expected and error")
		}
	})
	t.Run("one task", func(t *testing.T) {
		recipes := []Recipe{
			{ID: 1, CookTime: time.Second / 20},
		}
		ctx := t.Context()
		_, err := topo.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
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
		ctx := t.Context()
		_, err := topo.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("deeply nested", func(t *testing.T) {
		sleep := time.Second / 100
		recipes := []Recipe{
			{ID: 1, CookTime: sleep},
			{ID: 2, CookTime: sleep},
			{ID: 3, CookTime: sleep, Ingredients: []int{1, 2}},
			{ID: 4, CookTime: sleep, Ingredients: []int{1, 3}},
			{ID: 5, CookTime: sleep, Ingredients: []int{1, 4}},
			{ID: 6, CookTime: sleep, Ingredients: []int{1, 2, 5}},
			{ID: 7, CookTime: sleep, Ingredients: []int{1, 2, 5}},
			{ID: 8, CookTime: sleep, Ingredients: []int{1, 2, 3, 4, 5, 6}},
			{ID: 9, CookTime: sleep},
			{ID: 10, CookTime: sleep, Ingredients: []int{8}},
			{ID: 11, CookTime: sleep, Ingredients: []int{1}},
			{ID: 12, CookTime: sleep, Ingredients: []int{9}},
			{ID: 13, CookTime: sleep, Ingredients: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
		}
		ctx := t.Context()
		_, err := topo.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, func(recipe Recipe, ctx context.Context, vs []Ingredient) (Ingredient, error) {
			t.Log(recipe.ID, vs)
			return recipe.Cook(ctx, vs)
		})
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("missing parent", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("expected panic")
			}
		}()
		sleep := time.Second / 100
		recipes := []Recipe{
			{ID: 1, CookTime: sleep},
			{ID: 2, CookTime: sleep, Ingredients: []int{999}},
		}
		ctx := t.Context()
		_, _ = topo.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, func(recipe Recipe, ctx context.Context, vs []Ingredient) (Ingredient, error) {
			t.Log(recipe.ID, vs)
			return recipe.Cook(ctx, vs)
		})
	})
	t.Run("context canceled after start", func(t *testing.T) {
		recipes := []Recipe{
			{ID: 1, CookTime: time.Minute},
			{ID: 2, CookTime: time.Minute},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second/20)
		t.Cleanup(cancel)
		_, err := topo.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected deadline exceeded error, got: %v", err)
		}
	})
	t.Run("context canceled before start", func(t *testing.T) {
		recipes := []Recipe{
			{ID: 1, CookTime: time.Minute},
			{ID: 2, CookTime: time.Minute},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second/20)
		cancel()
		_, err := topo.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected deadline exceeded error, got: %v", err)
		}
	})
	t.Run("context canceled in function", func(t *testing.T) {
		recipes := []Recipe{
			{ID: 1},
			{ID: 2, Ingredients: []int{1}},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second/20)
		_, err := topo.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, func(t Recipe, ctx context.Context, vs []Ingredient) (Ingredient, error) {
			cancel()
			return Recipe.Cook(t, ctx, vs)
		})
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected deadline exceeded error, got: %v", err)
		}
	})
	t.Run("task failure", func(t *testing.T) {
		recipes := []Recipe{
			{ID: 1, CookTime: time.Minute, Ingredients: []int{2}},
			{ID: 2, CookTime: time.Minute, IsBad: true},
		}
		ctx := t.Context()
		_, err := topo.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if !errors.Is(err, ErrBadRecipe) {
			t.Errorf("expected bad recipe error, got: %v", err)
		}
	})
}
