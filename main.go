package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/types"
)

func main() {
	app := pocketbase.New()

	var queryTimeout int
	app.RootCmd.PersistentFlags().IntVar(
		&queryTimeout,
		"queryTimeout",
		30,
		"the default SELECT queries timeout in seconds",
	)
	app.RootCmd.ParseFlags(os.Args[1:])

	app.OnAfterBootstrap().PreAdd(func(e *core.BootstrapEvent) error {
		app.Dao().ModelQueryTimeout = time.Duration(queryTimeout) * time.Second
		return nil
	})

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method: http.MethodGet,
			Path:   "/articles",
			Handler: func(c echo.Context) error {

				collection, err := app.Dao().FindCollectionByNameOrId("articles")
				if err != nil {
					collection = &models.Collection{
						Name:       "articles",
						Type:       models.CollectionTypeBase,
						ListRule:   nil,
						ViewRule:   types.Pointer("@request.auth.id != ''"),
						CreateRule: types.Pointer(""),
						UpdateRule: types.Pointer("@request.auth.id != ''"),
						DeleteRule: nil,
						Schema: schema.NewSchema(
							&schema.SchemaField{
								Name:     "title",
								Type:     schema.FieldTypeText,
								Required: true,
								Options: &schema.TextOptions{
									Max: types.Pointer(10),
								},
							},
						),
					}
					if err := app.Dao().SaveCollection(collection); err != nil {
						return err
					}
				}
				record := models.NewRecord(collection)

				form := forms.NewRecordUpsert(app, record)
				input, err := readJsonInput()
				if err != nil {
					return c.String(http.StatusBadRequest, err.Error())

				}

				if err := form.LoadData(input); err != nil {
					return c.String(http.StatusInternalServerError, err.Error())

				}

				// validate and submit (internally it calls app.Dao().SaveRecord(record) in a transaction)
				if err := form.Submit(); err != nil {
					return c.String(http.StatusInternalServerError, err.Error())
				}
				return c.String(http.StatusOK, "success")
			},
			Middlewares: []echo.MiddlewareFunc{
				apis.ActivityLogger(app),
			},
		})
		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func readJsonInput() (map[string]interface{}, error) {
	fileContent, err := os.Open("input.json")
	if err != nil {
		return nil, err
	}

	defer fileContent.Close()

	byteResult, err := io.ReadAll(fileContent)

	var res map[string]interface{}
	json.Unmarshal([]byte(byteResult), &res)

	return res, err
}
