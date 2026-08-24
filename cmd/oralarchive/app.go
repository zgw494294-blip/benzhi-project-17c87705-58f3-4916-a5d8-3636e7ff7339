package main

import (
	"context"
	"net/http"

	"oralarchive/internal/application"
	"oralarchive/internal/httpapi"
	"oralarchive/internal/storage"
	"oralarchive/internal/web"
)

type app struct {
	store   *storage.Store
	handler http.Handler
}

func buildApp(ctx context.Context, dataDir string) (*app, error) {
	store, err := storage.Open(ctx, dataDir)
	if err != nil {
		return nil, err
	}
	service := application.New(store)
	api := httpapi.New(service)
	mux := http.NewServeMux()
	mux.Handle("/api/", api.Handler())
	mux.Handle("/", web.Handler())
	return &app{store: store, handler: mux}, nil
}
func (a *app) Close() error { return a.store.Close() }
