package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type recipe struct {
	ID          int64    `json:"id"`
	Title       string   `json:"title"`
	Difficulty  string   `json:"difficulty"`
	PrepPeriod  string   `json:"prep_period"`
	Method      string   `json:"method"`
	Categories  []string `json:"categories"`
	Ingredients []string `json:"ingredients"`
	Images      []string `json:"images"`
}

// save used for Create or Update
func (r *recipe) save(c *redis.Client) error {
	var save bool

	if r.ID == 0 {
		save = true
		id, err := c.Incr("recipe_id").Result()
		if err != nil {
			return err
		}
		r.ID = id
	}

	_, err := c.TxPipelined(func(pipe redis.Pipeliner) error {
		if save {
			if err := pipe.RPush("recipes", r.ID).Err(); err != nil {
				return err
			}
		}
		pipe.HMSet(fmt.Sprintf("recipe:%d", r.ID),
			"id", r.ID,
			"title", r.Title,
			"difficulty", r.Difficulty,
			"prep_period", r.PrepPeriod,
			"method", r.Method,
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

		saveList(r.ID, "categories", r.Categories, c, pipe)
		saveList(r.ID, "ingredients", r.Ingredients, c, pipe)
		saveList(r.ID, "images", r.Images, c, pipe)

		return nil
	})

	return err
}

func (r *recipe) load(id int64, c *redis.Client) error {
	if id <= 0 {
		return errors.New("invalid id")
	}

	r.ID = id

	var hgetAllCmd *redis.StringStringMapCmd
	var listCmds [3]*redis.StringSliceCmd

	_, err := c.Pipelined(func(pipe redis.Pipeliner) error {
		hgetAllCmd = pipe.HGetAll(fmt.Sprintf("recipe:%d", r.ID))

		for i, l := range []string{"categories", "ingredients", "images"} {
			listCmds[i] = pipe.LRange(fmt.Sprintf("recipe:%d:%s", r.ID, l), 0, -1)
		}
		return nil
	})
	if err != nil {
		return err
	}

	result, err := hgetAllCmd.Result()
	if err != nil {
		return err
	}
	r.Title = result["title"]
	r.Difficulty = result["difficulty"]
	r.PrepPeriod, _ = result["prep_period"]
	r.Method = result["method"]

	loadList := func(list ...*[]string) error {
		for i := range list {
			strings, err := listCmds[i].Result()
			if err != nil {
				return err
			}
			*list[i] = strings
		}
		return nil
	}
	err = loadList(&r.Categories, &r.Ingredients, &r.Images)
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
			cmds = append(cmds, pipe.HMGet(fmt.Sprintf("recipe:%s", recipeId), "Title"))
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

var rdb *redis.Client

func main() {

	rdb = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	r := mux.NewRouter()

	r.Path("/recipe").Methods("POST").HandlerFunc(createHandler)
	r.Path("/recipe/{id}").Methods("PUT").HandlerFunc(updateHandler)
	r.Path("/recipe/{id}").Methods("GET").HandlerFunc(viewHandler)
	r.Path("/recipes").Methods("GET").HandlerFunc(listHandler)

	log.Fatal(http.ListenAndServe(":8080", r))
}

func createHandler(w http.ResponseWriter, r *http.Request) {

	var recipe recipe

	err := json.NewDecoder(r.Body).Decode(&recipe)
	if err != nil {
		handleError(w, err)
		return
	}

	err = recipe.save(rdb)
	if err != nil {
		handleError(w, err)
		return
	}
}

func updateHandler(w http.ResponseWriter, r *http.Request) {

}

func viewHandler(w http.ResponseWriter, r *http.Request) {

}

func listHandler(w http.ResponseWriter, r *http.Request) {

}

func handleError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(err.Error()))
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
		Title:       "PanCake",
		Difficulty:  "easy",
		PrepPeriod:  "10m",
		Method:      "",
		Categories:  []string{"breakfast", "eastern"},
		Ingredients: []string{"eggs", "corn"},
		Images:      []string{"url1", "url2"},
	}

	for i := 0; i < 100; i++ {
		r.ID = 0
		r.Title = fmt.Sprintf("PanCake-%d", i)
		if err := r.save(client); err != nil {
			log.Fatal(err)
		}
	}
}
