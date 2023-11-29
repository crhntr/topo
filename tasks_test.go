package topo_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
		_, err := topo.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("self reference", func(t *testing.T) {
		recipes := []Recipe{
			{ID: 1, Ingredients: []int{1}},
		}
		ctx := context.Background()
		_, err := topo.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if err == nil {
			t.Error("expected and error")
		}
	})
	t.Run("one task", func(t *testing.T) {
		recipes := []Recipe{
			{ID: 1, CookTime: time.Second / 20},
		}
		ctx := context.Background()
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
		ctx := context.Background()
		_, err := topo.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if err != nil {
			t.Error(err)
		}
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
		ctx := context.Background()
		_, err := topo.Tasks(ctx, recipes, Recipe.Identifier, Recipe.Edges, Recipe.Cook)
		if !strings.Contains(err.Error(), "function returned error") {
			t.Errorf("expected error to contain 'function returned error', got: %v", err)
		}
		if !errors.Is(err, ErrBadRecipe) {
			t.Errorf("expected bad recipe error, got: %v", err)
		}
	})
}
