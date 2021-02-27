package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/oauth2/google"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}
	bucketName := os.Getenv("BUCKET_NAME")

	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, storage.ScopeReadOnly)
	if err != nil {
		log.Fatal(err)
	}
	conf, err := google.JWTConfigFromJSON(creds.JSON, storage.ScopeReadOnly)
	if err != nil {
		log.Fatal(err)
	}
	opts := &storage.SignedURLOptions{
		GoogleAccessID: conf.Email,
		PrivateKey:     conf.PrivateKey,
		Method:         http.MethodGet,
	}
	fn := func(fileName string, expires time.Time) (string, error) {
		opts.Expires = expires
		url, err := storage.SignedURL(bucketName, fileName, opts)
		if err != nil {
			return "", err
		}
		return url, nil
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/*", static())
	e.GET("/api/list", list(fn))

	if err := e.Start(":" + port); err != nil {
		log.Fatal(err)
	}
}

func static() echo.HandlerFunc {
	return func(c echo.Context) error {
		wd, err := os.Getwd()
		if err != nil {
			log.Println(err)
			return err
		}
		fs := http.FileServer(http.Dir(filepath.Join(wd, "view")))
		fs.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

func list(fn signedURLFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		url, err := fn("drink.JPG", time.Now().Add(30*time.Minute))
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		return c.String(http.StatusOK, url)
	}
}

type signedURLFunc func(fileName string, expires time.Time) (string, error)
