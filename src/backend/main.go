package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

func main() {
	/*
	 * 必須情報を環境変数から取得
	 */
	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		log.Fatal("no PROJECT_ID")
	}
	bucketName := os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		log.Fatal("no BUCKET_NAME")
	}
	cred := os.Getenv("SA_CREDENTIALS")
	if cred == "" {
		log.Fatal("no SA_CREDENTIALS")
	}

	/*
	 * GCSの署名付きURL生成関数実行用の設定
	 */
	conf, err := google.JWTConfigFromJSON([]byte(cred), storage.ScopeReadOnly)
	if err != nil {
		log.Fatal(err)
	}
	opts := &storage.SignedURLOptions{
		GoogleAccessID: conf.Email,
		PrivateKey:     conf.PrivateKey,
		Method:         http.MethodGet,
	}
	signedURLFunc := func(fileName string, expires time.Time) (string, error) {
		opts.Expires = expires
		url, err := storage.SignedURL(bucketName, fileName, opts)
		if err != nil {
			fmt.Println(err)
			return "", err
		}
		fmt.Printf("signedURL:%s\n", url)

		return url, nil
	}

	ctx := context.Background()

	/*
	 * GCSアクセス用クライアント生成
	 */
	storageCli, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(cred)))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if storageCli != nil {
			if err := storageCli.Close(); err != nil {
				fmt.Println(err)
			}
		}
	}()
	uploadGCSObjectFunc := func(ctx context.Context, objectName string, reader io.Reader) error {
		writer := storageCli.Bucket(bucketName).Object(objectName).NewWriter(ctx)
		defer func() {
			if writer != nil {
				if err := writer.Close(); err != nil {
					fmt.Println(err)
				}
			}
		}()
		writer.ContentType = "image/png"
		if _, err = io.Copy(writer, reader); err != nil {
			return fmt.Errorf("io.Copy: %v", err)
		}
		return nil
	}

	/*
	 * Firestoreアクセス用クライアント生成
	 */
	firestoreCli, err := firestore.NewClient(ctx, projectID, option.WithCredentialsJSON([]byte(cred)))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if firestoreCli != nil {
			if err := firestoreCli.Close(); err != nil {
				fmt.Println(err)
			}
		}
	}()

	/*
	 * Web APIサーバーとしての設定
	 */
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/*", static())
	e.GET("/api/list", list(firestoreCli, signedURLFunc))
	e.POST("/api/addImage", addImage(firestoreCli, uploadGCSObjectFunc))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		fmt.Printf("Defaulting to port %s\n", port)
	}

	if err := e.Start(":" + port); err != nil {
		log.Fatal(err)
	}
}

// 静的ルート用
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

func addImage(firestoreCli *firestore.Client, uploadGCSObjectFunc uploadGCSObjectFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.FormValue("name")
		fmt.Printf("name:%s\n", name)

		imageFile, err := c.FormFile("imageFile")
		if err != nil {
			fmt.Println(err)
			return c.String(http.StatusInternalServerError, err.Error())
		}
		f, err := imageFile.Open()
		if err != nil {
			fmt.Println(err)
			return c.String(http.StatusInternalServerError, err.Error())
		}
		fmt.Printf("imageFile.Filename:%s\n", imageFile.Filename)

		uid := uuid.New()

		objectName := uid.String() + filepath.Ext(imageFile.Filename)
		fmt.Printf("objectName:%s\n", objectName)

		if err := uploadGCSObjectFunc(c.Request().Context(), objectName, f); err != nil {
			fmt.Println(err)
			return c.String(http.StatusInternalServerError, err.Error())
		}

		_, err = firestoreCli.Collection("image").Doc(uid.String()).Set(c.Request().Context(),
			map[string]interface{}{
				"id":   uid.String(),
				"name": name,
				"date": time.Now().Format("2006-01-02"),
				"path": objectName,
			},
		)
		if err != nil {
			fmt.Println(err)
			return c.String(http.StatusInternalServerError, err.Error())
		}

		return nil
	}
}

func list(firestoreCli *firestore.Client, signedURLFunc signedURLFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		iter := firestoreCli.Collection("image").Documents(c.Request().Context())
		var images []*Image
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			var image *Image
			if err := doc.DataTo(&image); err != nil {
				fmt.Println(err)
				return c.String(http.StatusInternalServerError, err.Error())
			}
			url, err := signedURLFunc(image.Path, time.Now().Add(30*time.Minute))
			if err != nil {
				fmt.Println(err)
				return c.String(http.StatusInternalServerError, err.Error())
			}
			image.Path = url
			images = append(images, image)
		}
		return c.JSON(http.StatusOK, images)
	}
}

// GCSオブジェクトアップロード用関数
type uploadGCSObjectFunc func(ctx context.Context, objectName string, reader io.Reader) error

// 署名付きURL生成用関数
type signedURLFunc func(fileName string, expires time.Time) (string, error)

type Image struct {
	Date string `json:"date"`
	ID   string `json:"id"`
	Path string `json:"path"`
	Name string `json:"name"`
}
