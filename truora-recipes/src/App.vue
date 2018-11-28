<template>
  <div id="app">
    <b-container fluid>
      <h1>Recetas</h1>
      <div class="top-container">
        <span class="search-text">Buscar receta</span>
        <div class="search-container">
          <b-form-input
            v-model="search"
            class="col-lg-4 col-sm-6 col-xs-11"
            type="text"
            placeholder="Pasta"
          ></b-form-input>
          <transition name="fade">
            <div v-if="fetching" class="lds-ripple col-1">
              <div></div>
              <div></div>
            </div>
          </transition>
        </div>
        <span class="text-danger">{{errorMsg}}</span>
      </div>
      <span v-if="!fetching && recipeData.length === 0">No encontramos ningún resultado :(</span>
      <b-card-group deck class="mb-4 mt-4">
        <b-card
          v-for="recipe in recipeData"
          :key="recipe.id"
          :title="recipe.name"
          :img-src="recipe.image"
          img-fluid
        >
          <p>Difficulty: {{recipe.difficulty}}</p>
          <p>{{shortPreparation(recipe.procedure)}}</p>
          <b-button size="sm" variant="primary">Ver receta completa</b-button>
        </b-card>
      </b-card-group>
    </b-container>
  </div>
</template>

<script>
import { debounce } from "debounce";

export default {
  name: "app",
  data() {
    return {
      search: "",
      initialData: [],
      fetchedData: [],
      fetching: false,
      errorMsg: ""
    };
  },
  created() {
    this.debouncedSearch = debounce(this.getSearch, 350);
    this.fetchInitialData();
  },
  watch: {
    search() {
      this.fetching = true;
      this.debouncedSearch();
    }
  },
  computed: {
    recipeData() {
      if (this.search && this.fetchedData) {
        return this.fetchedData;
      } else if (this.search && !this.fetchedData) {
        return [];
      } else {
        return this.initialData;
      }
    }
  },
  methods: {
    getSearch() {
      this.fetching = true;
      if (this.search) {
        fetch("http://localhost:8085/recipe?search=" + this.search, {
          method: "GET",
          /* mode: "no-cors", */
          headers: {
            Accept: "application/json",
            "Content-Type": "application/json; charset=utf-8"
          }
        })
          .then(resp => resp.json())
          .then(json => {
            console.log(json);

            this.fetching = false;
            this.fetchedData = json;
            this.errorMsg = "";
          })
          .catch(err => {
            console.log(err);
            this.errorMsg = "There was an error fetching data :(";
            this.fetching = false;
          });
      } else {
        this.fetching = false;
      }
    },
    fetchInitialData() {
      fetch("http://localhost:8085/recipes", {
        method: "GET",
        /* mode: "no-cors", */
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json; charset=utf-8"
        }
      })
        .then(resp => resp.json())
        .then(json => {
          console.log(json);

          this.initialData = json;
        })
        .catch(err => {
          console.log(err);
          this.err = "There was an error fetching data :(";
          this.err = err;
        });
    },
    shortPreparation(preparation) {
      if (preparation.length > 50) {
        return preparation.substring(0, 47) + "...";
      } else {
        return preparation;
      }
    }
  }
};
</script>

<style>
#app {
  font-family: "Avenir", Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  color: #2c3e50;
  margin: 16px;
}

.top-container {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: flex-start;
}

.search-container {
  display: flex;
  flex-direction: row;
  justify-content: flex-start;
  align-items: center;
  width: 100%;
}

.card .card-img {
  max-height: 184px;
  max-width: 244px;
}

/*Spinner css*/

html #app .lds-ripple {
  display: inline-block;
  position: relative;
  width: 2.25rem;
  height: 2.25rem;
}
html #app .lds-ripple div {
  position: absolute;
  border: 4px solid rgb(0, 110, 236);
  opacity: 1;
  border-radius: 50%;
  animation: lds-ripple 1s cubic-bezier(0, 0.2, 0.8, 1) infinite;
}
html #app .lds-ripple div:nth-child(2) {
  animation-delay: -0.5s;
}

@keyframes lds-ripple {
  0% {
    top: calc(2.25rem / 2);
    left: calc(2.25rem / 2);
    width: 0;
    height: 0;
    opacity: 1;
  }
  100% {
    top: -1px;
    left: -1px;
    width: 2.25rem;
    height: 2.25rem;
    opacity: 0;
  }
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.5s;
}
.fade-enter, .fade-leave-to /* .fade-leave-active below version 2.1.8 */ {
  opacity: 0;
}
</style>
