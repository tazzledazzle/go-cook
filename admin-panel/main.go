package main

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type FeatureFlag struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
}

var (
	mu    sync.Mutex
	flags = map[string]*FeatureFlag{
		"new-checkout": {Key: "new-checkout", Enabled: false},
	}
)

func main() {

}
