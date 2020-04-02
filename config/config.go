// Package config provides access to run-time configuration.
package config

import (
	"context"
	"os"

	"cloud.google.com/go/datastore"
)

type credentials struct {
	SecretKey   string `datastore:",noindex"`
	AccessToken string `datastore:",noindex"`
}

var cachedCreds *credentials

func loadCreds(ctx context.Context) error {
	if cachedCreds != nil {
		return nil
	}

	client, err := datastore.NewClient(ctx, datastore.DetectProjectID)
	if err != nil {
		return err
	}

	var c credentials
	if err := client.Get(ctx, datastore.NameKey("credentials", "singleton", nil), &c); err != nil {
		return err
	}

	cachedCreds = &c
	return nil
}

// SecretKey returns the shared secret used to verify the signature on Github events.
func SecretKey(ctx context.Context) ([]byte, error) {
	if err := loadCreds(ctx); err != nil {
		return nil, err
	}

	return []byte(cachedCreds.SecretKey), nil
}

// AccessToken returns the access token used to authenticate requests to Github.
func AccessToken(ctx context.Context) (string, error) {
	if err := loadCreds(ctx); err != nil {
		return "", err
	}

	return cachedCreds.AccessToken, nil
}
