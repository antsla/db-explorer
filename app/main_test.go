package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type CR map[string]interface{}

type Case struct {
	Method string
	Path   string
	Query  string
	Status int
	Result interface{}
	Body   interface{}
}

var (
	client = &http.Client{Timeout: time.Second}
)

func PrepareTestApis(db *sql.DB) {
	qs := []string{
		`DROP TABLE IF EXISTS items;`,

		`CREATE TABLE items (
  			id int(11) NOT NULL AUTO_INCREMENT,
		  	title varchar(255) NOT NULL,
		  	description text NOT NULL,
		  	updated varchar(255) DEFAULT NULL,
		  	PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8;`,

		`INSERT INTO items (id, title, description, updated) VALUES
			(1,	'title 1', 'description 1',	'vantonyuk'),
			(2,	'title 2', 'description 2',	NULL);`,

		`DROP TABLE IF EXISTS users;`,

		`CREATE TABLE users (
			user_id int(11) NOT NULL AUTO_INCREMENT,
  			login varchar(255) NOT NULL,
	  		password varchar(255) NOT NULL,
	  		email varchar(255) NOT NULL,
  			info text NOT NULL,
  			updated varchar(255) DEFAULT NULL,
  			PRIMARY KEY (user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8;`,

		`INSERT INTO users (user_id, login, password, email, info, updated) VALUES
			(1,	'vantonyuk', '12345', 'vantonyuk@example.com', 'info',	NULL);`,
	}

	for _, q := range qs {
		_, err := db.Exec(q)
		if err != nil {
			panic(err)
		}
	}
}

func CleanupTestApis(db *sql.DB) {
	qs := []string{
		`DROP TABLE IF EXISTS items;`,
		`DROP TABLE IF EXISTS users;`,
	}
	for _, q := range qs {
		_, err := db.Exec(q)
		if err != nil {
			panic(err)
		}
	}
}

func TestApis(t *testing.T) {
	db, err := sql.Open("mysql", DSN)
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	PrepareTestApis(db)

	// defer CleanupTestApis(db)

	handler, err := NewDbExplorer(db)
	if err != nil {
		panic(err)
	}

	ts := httptest.NewServer(handler)

	cases := []Case{
		Case{
			Path: "/",
			Result: CR{
				"response": CR{
					"tables": []string{"items", "users"},
				},
			},
		},
		Case{
			Path:   "/unknown_table",
			Status: http.StatusNotFound,
			Result: CR{
				"error": "unknown table",
			},
		},
		Case{
			Path: "/items",
			Result: CR{
				"response": CR{
					"records": []CR{
						CR{
							"id":          1,
							"title":       "title 1",
							"description": "description 1",
							"updated":     "vantonyuk",
						},
						CR{
							"id":          2,
							"title":       "title 2",
							"description": "description 2",
							"updated":     nil,
						},
					},
				},
			},
		},
		Case{
			Path:  "/items",
			Query: "limit=1",
			Result: CR{
				"response": CR{
					"records": []CR{
						CR{
							"id":          1,
							"title":       "title 1",
							"description": "description 1",
							"updated":     "vantonyuk",
						},
					},
				},
			},
		},
		Case{
			Path:  "/items",
			Query: "limit=1&offset=1",
			Result: CR{
				"response": CR{
					"records": []CR{
						CR{
							"id":          2,
							"title":       "title 2",
							"description": "description 2",
							"updated":     nil,
						},
					},
				},
			},
		},
		Case{
			Path: "/items/1",
			Result: CR{
				"response": CR{
					"record": CR{
						"id":          1,
						"title":       "title 1",
						"description": "description 1",
						"updated":     "vantonyuk",
					},
				},
			},
		},
		Case{
			Path:   "/items/100500",
			Status: http.StatusNotFound,
			Result: CR{
				"error": "record not found",
			},
		},
		Case{
			Path:   "/items/",
			Method: http.MethodPut,
			Body: CR{
				"id":          42,
				"title":       "title 3",
				"description": "",
			},
			Result: CR{
				"response": CR{
					"id": 3,
				},
			},
		},
		Case{
			Path: "/items/3",
			Result: CR{
				"response": CR{
					"record": CR{
						"id":          3,
						"title":       "title 3",
						"description": "",
						"updated":     nil,
					},
				},
			},
		},
		Case{
			Path:   "/items/3",
			Method: http.MethodPost,
			Body: CR{
				"description": "description 3",
			},
			Result: CR{
				"response": CR{
					"updated": 1,
				},
			},
		},
		Case{
			Path: "/items/3",
			Result: CR{
				"response": CR{
					"record": CR{
						"id":          3,
						"title":       "title 3",
						"description": "description 3",
						"updated":     nil,
					},
				},
			},
		},
		Case{
			Path:   "/items/3",
			Method: http.MethodPost,
			Body: CR{
				"updated": "autotests",
			},
			Result: CR{
				"response": CR{
					"updated": 1,
				},
			},
		},
		Case{
			Path: "/items/3",
			Result: CR{
				"response": CR{
					"record": CR{
						"id":          3,
						"title":       "title 3",
						"description": "description 3",
						"updated":     "autotests",
					},
				},
			},
		},
		Case{
			Path:   "/items/3",
			Method: http.MethodPost,
			Body: CR{
				"updated": nil,
			},
			Result: CR{
				"response": CR{
					"updated": 1,
				},
			},
		},
		Case{
			Path: "/items/3",
			Result: CR{
				"response": CR{
					"record": CR{
						"id":          3,
						"title":       "title 3",
						"description": "description 3",
						"updated":     nil,
					},
				},
			},
		},
		Case{
			Path:   "/items/3",
			Method: http.MethodPost,
			Status: http.StatusBadRequest,
			Body: CR{
				"id": 4,
			},
			Result: CR{
				"error": "field id have invalid type",
			},
		},
		Case{
			Path:   "/items/3",
			Method: http.MethodPost,
			Status: http.StatusBadRequest,
			Body: CR{
				"title": 42,
			},
			Result: CR{
				"error": "field title have invalid type",
			},
		},
		Case{
			Path:   "/items/3",
			Method: http.MethodPost,
			Status: http.StatusBadRequest,
			Body: CR{
				"title": nil,
			},
			Result: CR{
				"error": "field title have invalid type",
			},
		},
		Case{
			Path:   "/items/3",
			Method: http.MethodPost,
			Status: http.StatusBadRequest,
			Body: CR{
				"updated": 42,
			},
			Result: CR{
				"error": "field updated have invalid type",
			},
		},
		Case{
			Path:   "/items/3",
			Method: http.MethodDelete,
			Result: CR{
				"response": CR{
					"deleted": 1,
				},
			},
		},
		Case{
			Path:   "/items/3",
			Method: http.MethodDelete,
			Result: CR{
				"response": CR{
					"deleted": 0,
				},
			},
		},
		Case{
			Path:   "/items/3",
			Status: http.StatusNotFound,
			Result: CR{
				"error": "record not found",
			},
		},
		Case{
			Path: "/users/1",
			Result: CR{
				"response": CR{
					"record": CR{
						"user_id":  1,
						"login":    "vantonyuk",
						"password": "12345",
						"email":    "vantonyuk@example.com",
						"info":     "info",
						"updated":  nil,
					},
				},
			},
		},
		Case{
			Path:   "/users/1",
			Method: http.MethodPost,
			Body: CR{
				"info":    "try update",
				"updated": "now",
			},
			Result: CR{
				"response": CR{
					"updated": 1,
				},
			},
		},
		Case{
			Path: "/users/1",
			Result: CR{
				"response": CR{
					"record": CR{
						"user_id":  1,
						"login":    "vantonyuk",
						"password": "12345",
						"email":    "vantonyuk@example.com",
						"info":     "try update",
						"updated":  "now",
					},
				},
			},
		},
		Case{
			Path:   "/users/1",
			Method: http.MethodPost,
			Status: http.StatusBadRequest,
			Body: CR{
				"user_id": 1,
			},
			Result: CR{
				"error": "field user_id have invalid type",
			},
		},
		Case{
			Path:   "/users/",
			Method: http.MethodPut,
			Body: CR{
				"user_id":    2,
				"login":      "qwerty'",
				"password":   "12345\"",
				"unkn_field": "info",
			},
			Result: CR{
				"response": CR{
					"user_id": 2,
				},
			},
		},
		Case{
			Path: "/users/2",
			Result: CR{
				"response": CR{
					"record": CR{
						"user_id":  2,
						"login":    "qwerty'",
						"password": "12345\"",
						"email":    "",
						"info":     "",
						"updated":  nil,
					},
				},
			},
		},
		Case{
			Path:  "/users",
			Query: "limit=1'&offset=1\"",
			Result: CR{
				"response": CR{
					"records": []CR{
						CR{
							"user_id":  1,
							"login":    "vantonyuk",
							"password": "12345",
							"email":    "vantonyuk@example.com",
							"info":     "try update",
							"updated":  "now",
						},
						CR{
							"user_id":  2,
							"login":    "qwerty'",
							"password": "12345\"",
							"email":    "",
							"info":     "",
							"updated":  nil,
						},
					},
				},
			},
		},
	}

	runCases(t, ts, db, cases)
}

func runCases(t *testing.T, ts *httptest.Server, db *sql.DB, cases []Case) {
	for idx, item := range cases {
		var (
			err      error
			result   interface{}
			expected interface{}
			req      *http.Request
		)

		caseName := fmt.Sprintf("case %d: [%s] %s %s", idx, item.Method, item.Path, item.Query)

		if db.Stats().OpenConnections != 1 {
			t.Fatalf("[%s] you have %d open connections, must be 1", caseName, db.Stats().OpenConnections)
		}

		if item.Method == "" || item.Method == http.MethodGet {
			req, err = http.NewRequest(item.Method, ts.URL+item.Path+"?"+item.Query, nil)
		} else {
			data, err := json.Marshal(item.Body)
			if err != nil {
				panic(err)
			}
			reqBody := bytes.NewReader(data)
			req, err = http.NewRequest(item.Method, ts.URL+item.Path, reqBody)
			req.Header.Add("Content-Type", "application/json")
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("[%s] request error: %v", caseName, err)
			continue
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		if item.Status == 0 {
			item.Status = http.StatusOK
		}

		if resp.StatusCode != item.Status {
			t.Fatalf("[%s] expected http status %v, got %v", caseName, item.Status, resp.StatusCode)
			continue
		}

		err = json.Unmarshal(body, &result)
		if err != nil {
			t.Fatalf("[%s] cant unpack json: %v", caseName, err)
			continue
		}

		data, err := json.Marshal(item.Result)
		json.Unmarshal(data, &expected)

		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("[%s] results not match\nGot : %#v\nWant: %#v", caseName, result, expected)
			continue
		}
	}

}
