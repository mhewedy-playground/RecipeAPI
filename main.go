package main

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v7"
	"log"
	"time"
)

type recipe struct {
	id          int64
	title       string
	difficulty  string
	prepPeriod  time.Duration
	method      string
	categories  []string
	ingredients []string
	images      []string
}

// save used for Create or Update
func (r *recipe) save(c *redis.Client) error {
	var save bool

	if r.id == 0 {
		save = true
		id, err := c.Incr("recipe_id").Result()
		if err != nil {
			return err
		}
		r.id = id
	}

	_, err := c.TxPipelined(func(pipe redis.Pipeliner) error {
		if save {
			if err := pipe.RPush("recipes", r.id).Err(); err != nil {
				return err
			}
		}
		pipe.HMSet(fmt.Sprintf("recipe:%d", r.id),
			"id", r.id,
			"title", r.title,
			"difficulty", r.difficulty,
			"prep_period", r.prepPeriod.String(),
			"method", r.method,
		)

		saveList := func(recipeId int64, name string, values []string, c *redis.Client, pipe redis.Pipeliner) {
			if values == nil {
				return
			}
			key := fmt.Sprintf("recipe:%d:%s", recipeId, name)
			if c.Exists(key).Val() == 1 {
				return
			}
			pipe.RPush(key, values)
		}

		saveList(r.id, "categories", r.categories, c, pipe)
		saveList(r.id, "ingredients", r.ingredients, c, pipe)
		saveList(r.id, "images", r.images, c, pipe)

		return nil
	})

	return err
}

func (r *recipe) load(id int64, c *redis.Client) error {
	if id <= 0 {
		return errors.New("invalid id")
	}

	r.id = id

	var hgetAllCmd *redis.StringStringMapCmd
	var categoriesCmd *redis.StringSliceCmd
	var ingredientsCmd *redis.StringSliceCmd
	var imagesCmd *redis.StringSliceCmd

	_, err := c.Pipelined(func(pipe redis.Pipeliner) error {

		hgetAllCmd = pipe.HGetAll(fmt.Sprintf("recipe:%d", r.id))
		categoriesCmd = pipe.LRange(fmt.Sprintf("recipe:%d:categories", r.id), 0, -1)
		ingredientsCmd = pipe.LRange(fmt.Sprintf("recipe:%d:ingredients", r.id), 0, -1)
		imagesCmd = pipe.LRange(fmt.Sprintf("recipe:%d:images", r.id), 0, -1)

		return nil
	})
	if err != nil {
		return err
	}

	result, err := hgetAllCmd.Result()
	if err != nil {
		return err
	}
	r.title = result["title"]
	r.difficulty = result["difficulty"]
	r.prepPeriod, _ = time.ParseDuration(result["prep_period"])
	r.method = result["method"]

	r.categories, err = categoriesCmd.Result()
	if err != nil {
		return err
	}
	r.ingredients, err = ingredientsCmd.Result()
	if err != nil {
		return err
	}
	r.images, err = imagesCmd.Result()
	if err != nil {
		return err
	}

	return nil
}

func list(page int, c *redis.Client) ([]string, error) {
	if page <= 0 {
		return nil, errors.New("invalid page")
	}

	const pageSize int64 = 20
	from, to := (int64(page)-1)*pageSize, int64(page)*pageSize-1

	recipeIds, err := c.LRange("recipes", from, to).Result()
	if err != nil {
		return nil, err
	}

	var cmds []*redis.SliceCmd
	_, err = c.Pipelined(func(pipe redis.Pipeliner) error {
		for _, recipeId := range recipeIds {
			cmds = append(cmds, pipe.HMGet(fmt.Sprintf("recipe:%s", recipeId), "title"))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var titles []string
	for _, c := range cmds {
		titles = append(titles, c.Val()[0].(string))
	}

	return titles, nil
}

func main() {

	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	//initDB(client)

	//fmt.Println(list(1, client))

	load(1, client)
}

func load(id int64, client *redis.Client) {
	var recipe = &recipe{}
	if err := recipe.load(id, client); err != nil {
		log.Fatal(err)
	}
	fmt.Println(recipe)
}

func initDB(client *redis.Client) {
	r := &recipe{
		title:       "PanCake",
		difficulty:  "easy",
		prepPeriod:  10 * time.Minute,
		method:      "",
		categories:  []string{"breakfast", "eastern"},
		ingredients: []string{"eggs", "corn"},
		images:      []string{"url1", "url2"},
	}

	for i := 0; i < 100; i++ {
		r.id = 0
		r.title = fmt.Sprintf("PanCake-%d", i)
		if err := r.save(client); err != nil {
			log.Fatal(err)
		}
	}
}
