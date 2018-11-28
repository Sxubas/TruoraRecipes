package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	_ "github.com/lib/pq"       //Postgres driver
	"github.com/satori/go.uuid" //Golang UUID management for postgres
)

var (
	//Port the web services will bind to, defaults to 8085
	port = flag.String("port", "8085", "Port to listen on")
)

type Recipe struct {
	Name        string   `json:"name"`
	Difficulty  int      `json:"difficulty"`
	Ingredients []string `json:"ingredients"`
	Procedure   string   `json:"procedure"`
	Image       string   `json:"image"`
}

type SimpleRecipe struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Difficulty int    `json:"difficulty"`
	Procedure  string `json:"procedure"`
	Image      string `json:"image"`
}

func prepareDB() {
	//Connecting to the 'recipes' db
	db, err := sql.Open("postgres", "postgresql://maxroach@localhost:26257/recipes?sslmode=disable")
	if err != nil {
		log.Fatal("Error connecting to the db: ", err)
	}

	// Create the "recipes" table
	if _, err := db.Exec(
		"CREATE TABLE IF NOT EXISTS recipes (id UUID PRIMARY KEY, name TEXT, difficulty INT, procedure TEXT, image TEXT)"); err != nil {
		log.Fatal(err)
	}

	// Create the "ingredients" table
	if _, err := db.Exec(
		"CREATE TABLE IF NOT EXISTS ingredients (name TEXT PRIMARY KEY)"); err != nil {
		log.Fatal(err)
	}

	// Create the "recipes_ingredients" table (many to many)
	if _, err := db.Exec(
		"CREATE TABLE IF NOT EXISTS recipes_ingredients (ingredient TEXT REFERENCES ingredients(name), recipeID UUID REFERENCES recipes(id) )"); err != nil {
		log.Fatal(err)
	}
}

func insertRecipe(recipe *Recipe) {
	//Connect to the 'recipes' db
	db, err := sql.Open("postgres", "postgresql://maxroach@localhost:26257/recipes?sslmode=disable")
	if err != nil {
		log.Fatal("Error connecting to the db: ", err)
	}

	id, err := uuid.NewV4()

	if err != nil {
		log.Fatal("Error generating UID: ", err)
		return
	}

	if _, err := db.Exec(
		"INSERT INTO recipes (name, difficulty, procedure, image, id) values ( $1, $2, $3, $4, $5)",
		recipe.Name, recipe.Difficulty, recipe.Procedure, recipe.Image, id); err != nil {
		log.Fatal(err)
		return
	}

	for i := 0; i < len(recipe.Ingredients); i++ {
		//Insert into ingredients table
		if _, err := db.Exec(
			"INSERT INTO ingredients (name) values ($1) ON CONFLICT DO NOTHING", recipe.Ingredients[i]); err != nil {
			log.Fatal(err)
			return
		}
		//Then, into many-to-many table
		if _, err := db.Exec(
			"INSERT INTO recipes_ingredients (ingredient, recipeID) values ($1, $2) ON CONFLICT DO NOTHING", recipe.Ingredients[i], id); err != nil {
			log.Fatal(err)
			return
		}
	}
}

func updateRecipe(recipe *Recipe, id string) {
	//Connect to the 'recipes' db
	db, err := sql.Open("postgres", "postgresql://maxroach@localhost:26257/recipes?sslmode=disable")
	if err != nil {
		log.Fatal("Error connecting to the db: ", err)
	}

	if _, err := db.Exec(
		"UPDATE recipes SET name = $1, difficulty = $2, procedure = $3, image = $4 WHERE id = $5",
		recipe.Name, recipe.Difficulty, recipe.Procedure, recipe.Image, id); err != nil {
		log.Fatal(err)
		return
	}

	if _, err := db.Exec(
		"DELETE FROM recipes_ingredients WHERE recipeID = $1", id); err != nil {
		log.Fatal(err)
		return
	}

	for i := 0; i < len(recipe.Ingredients); i++ {
		//Insert into ingredients table
		if _, err := db.Exec(
			"INSERT INTO ingredients (name) values ($1) ON CONFLICT DO NOTHING", recipe.Ingredients[i]); err != nil {
			log.Fatal(err)
			return
		}
		//Then, into many-to-many table
		if _, err := db.Exec(
			"INSERT INTO recipes_ingredients (ingredient, recipeID) values ($1, $2) ON CONFLICT DO NOTHING", recipe.Ingredients[i], id); err != nil {
			log.Fatal(err)
			return
		}
	}
}

func deleteRecipe(id string) {
	//Connect to the 'recipes' db
	db, err := sql.Open("postgres", "postgresql://maxroach@localhost:26257/recipes?sslmode=disable")
	if err != nil {
		log.Fatal("Error connecting to the db: ", err)
	}

	if _, err := db.Exec(
		"DELETE FROM recipes_ingredients WHERE recipeID = $1", id); err != nil {
		log.Fatal(err)
		return
	}

	if _, err := db.Exec(
		"DELETE FROM recipes WHERE id = $1", id); err != nil {
		log.Fatal(err)
		return
	}
}

func searchRecipe(search string) []SimpleRecipe {
	//Connect to the 'recipes' db
	db, err := sql.Open("postgres", "postgresql://maxroach@localhost:26257/recipes?sslmode=disable")
	if err != nil {
		log.Fatal("Error connecting to the db: ", err)
	}

	rows, err := db.Query("SELECT id, name, difficulty, image, procedure FROM recipes WHERE name LIKE '%' || $1 || '%'", search)
	if err != nil {
		log.Fatal("Error querying products: ", err)
	}
	defer rows.Close()

	var (
		id         string
		name       string
		difficulty int
		image      string
		procedure  string
		results    []SimpleRecipe
	)

	for rows.Next() {
		err := rows.Scan(&id, &name, &difficulty, &image, &procedure)
		if err != nil {
			log.Fatal("Error scanning results: ", err)
		}
		results = append(results, SimpleRecipe{Id: id,
			Name:       name,
			Difficulty: difficulty,
			Image:      image,
			Procedure:  procedure,
		})
	}
	return results
}

func searchRecipeById(id string) Recipe {
	//Connect to the 'recipes' db
	db, err := sql.Open("postgres", "postgresql://maxroach@localhost:26257/recipes?sslmode=disable")
	if err != nil {
		log.Fatal("Error connecting to the db: ", err)
	}

	var (
		name        string
		difficulty  int
		image       string
		procedure   string
		ingredient  string
		ingredients []string
	)

	err = db.QueryRow("SELECT id, name, difficulty, image, procedure FROM recipes WHERE id = $1", id).Scan(&id, &name, &difficulty, &image, &procedure)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return Recipe{}
		}
		log.Println("Error querying products: ", err)
	}

	rows, err := db.Query("SELECT igredient FROM recipes_ingredients WHERE recipeID = $1", id)
	if err != nil {
		log.Fatal("Error querying ingredients: ", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ingredient)
		if err != nil {
			log.Fatal("Error scanning results: ", err)
		}
		ingredients = append(ingredients, ingredient)
	}

	return Recipe{
		Name:        name,
		Difficulty:  difficulty,
		Image:       image,
		Procedure:   procedure,
		Ingredients: ingredients,
	}
}

func recipesHandler(w http.ResponseWriter, r *http.Request) {

	//No CORS for development purposes
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type")

	//Set content type to json
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method == "POST" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
		}

		recipe := &Recipe{}

		err = json.Unmarshal([]byte(string(body)), recipe)

		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			fmt.Println("Error parsing body: ", err)
			return
		} else {
			fmt.Println(string(body))
		}

		insertRecipe(recipe)

		fmt.Fprint(w, "POST done:\n", recipe)
	}

	if r.Method == "GET" && r.URL.Query().Get("search") != "" {

		data := searchRecipe(r.URL.Query().Get("search"))

		resp, err := json.Marshal(data)

		if err != nil {
			http.Error(w, "Error converting respone", http.StatusInternalServerError)
			fmt.Println("Error converting body: ", err)
			return
		}

		fmt.Fprint(w, string(resp))
	}

	if r.Method == "GET" && r.URL.Query().Get("id") != "" {

		result := searchRecipeById(r.URL.Query().Get("id"))

		resp, err := json.Marshal(result)

		if err != nil {
			http.Error(w, "Error converting respone", http.StatusInternalServerError)
			fmt.Println("Error converting body: ", err)
			return
		}

		fmt.Fprint(w, string(resp))
	}

	if r.Method == "PUT" && r.URL.Query().Get("id") != "" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
		}

		recipe := &Recipe{}

		err = json.Unmarshal([]byte(string(body)), recipe)

		updateRecipe(recipe, r.URL.Query().Get("id"))

		fmt.Fprint(w, recipe)

	}

	if r.Method == "DELETE" && r.URL.Query().Get("id") != "" {
		deleteRecipe(r.URL.Query().Get("id"))
		fmt.Fprint(w, "{}")
	}

}

func allRecipes() []SimpleRecipe {
	//Connect to the 'recipes' db
	db, err := sql.Open("postgres", "postgresql://maxroach@localhost:26257/recipes?sslmode=disable")
	if err != nil {
		log.Fatal("Error connecting to the db: ", err)
	}

	rows, err := db.Query("SELECT id, name, difficulty, procedure, image FROM recipes")
	if err != nil {
		log.Fatal("Error querying products: ", err)
	}
	defer rows.Close()

	var (
		id         string
		name       string
		difficulty int
		procedure  string
		image      string
		results    []SimpleRecipe
	)

	for rows.Next() {
		err := rows.Scan(&id, &name, &difficulty, &procedure, &image)
		if err != nil {
			log.Fatal("Error scanning results: ", err)
		}
		results = append(results, SimpleRecipe{Id: id, Name: name, Difficulty: difficulty, Procedure: procedure, Image: image})
	}
	return results
}

func allRecipesHandler(w http.ResponseWriter, r *http.Request) {

	//No CORS for development purposes
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type")

	//Set content type to json
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method == "GET" {

		results := allRecipes()

		resp, err := json.Marshal(results)

		if err != nil {
			http.Error(w, "Error converting respone", http.StatusInternalServerError)
			fmt.Println("Error converting body: ", err)
			return
		}

		fmt.Fprint(w, string(resp))
	}
}

func main() {
	flag.Parse()
	//Drop past tables and create new ones
	prepareDB()

	//CRUD Handler for recipes
	http.HandleFunc("/recipe", recipesHandler)

	//CRUD Handler for all recipes
	http.HandleFunc("/recipes", allRecipesHandler)

	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
