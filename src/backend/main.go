package main

import (
	"fmt"
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
		fmt.Printf("Defaulting to port %s\n", port)
	}
	bucketName := os.Getenv("BUCKET_NAME")
	cred := os.Getenv("GCS_SA_CREDENTIALS")

	conf, err := google.JWTConfigFromJSON([]byte(cred), storage.ScopeReadOnly)
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
			fmt.Println(err)
			return "", err
		}
		fmt.Printf("signedURL:%s\n", url)

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
			fmt.Println(err)
			return c.String(http.StatusInternalServerError, err.Error())
		}
		fmt.Println(url)

		var imgs []*Image
		imgs = append(imgs, &Image{
			Title: "床下パントリー",
			Date:  "2021-02-28",
			URL:   url,
		})

		return c.JSON(http.StatusOK, imgs)
	}
}

type signedURLFunc func(fileName string, expires time.Time) (string, error)

type Image struct {
	Title, Date, URL string
}
