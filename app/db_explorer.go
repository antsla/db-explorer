package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"strings"
)

const (
	errorUnknownTable   = "unknown table"
	errorRecordNotFound = "record not found"
)

type Explorer struct {
	db *sql.DB
}

type Field struct {
	Name      string
	IsPrimary bool
	Type      string
	Required  bool
}

type Table struct {
	Name   string
	IdName string
}

func (t *Table) GetFields(e Explorer) ([]Field, error) {
	fieldRows, err := e.db.Query(`
		SELECT column_name, 
			IF(COLUMN_KEY='PRI', true, false) AS is_primary, 
			data_type, 
			IF(IS_NULLABLE='YES', false, true) AS required
		FROM information_schema.columns 
		WHERE table_name = ? 
			AND table_schema = database()`, t.Name)

	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}
	defer fieldRows.Close()

	fields := make([]Field, 0)
	for fieldRows.Next() {
		field := &Field{}
		fieldRows.Scan(&field.Name, &field.IsPrimary, &field.Type, &field.Required)
		fields = append(fields, *field)
		if field.IsPrimary {
			t.IdName = field.Name
		}
	}

	return fields, nil
}

func (t *Table) CreateTemplate(fields []Field) []interface{} {
	recordTemplate := make([]interface{}, len(fields))
	for index, field := range fields {
		switch field.Type {
		case "int":
			recordTemplate[index] = new(sql.NullInt64)
		case "text", "varchar":
			recordTemplate[index] = new(sql.NullString)
		}

	}
	return recordTemplate
}

func (t *Table) FillField(fieldTemplate interface{}) interface{} {
	switch fieldTemplate.(type) {
	case *sql.NullString:
		if value, ok := fieldTemplate.(*sql.NullString); ok {
			if value.Valid {
				return value.String
			}
		}
	case *sql.NullInt64:
		if value, ok := fieldTemplate.(*sql.NullInt64); ok {
			if value.Valid {
				return value.Int64
			}
		}
	}
	return nil
}

func (t *Table) SetDefault(field Field) interface{} {
	if field.Required {
		switch field.Type {
		case "text", "varchar":
			return ""
		case "int":
			return 0
		}
	}
	return nil
}

func (t *Table) ValidateField(field Field, value interface{}) error {
	if field.IsPrimary {
		return fmt.Errorf("field %s have invalid type", t.IdName)
	}
	if value == nil && field.Required {
		return fmt.Errorf("field %s have invalid type", field.Name)
	}
	switch value.(type) {
	case float64:
		if field.Type != "int" {
			return fmt.Errorf("field %s have invalid type", field.Name)
		}
	case string:
		if field.Type != "varchar" && field.Type != "text" {
			return fmt.Errorf("field %s have invalid type", field.Name)
		}
	}
	return nil
}

func ResponseWriter(w http.ResponseWriter, responseCode int, errorMessage string, response interface{}) {
	result := make(map[string]interface{})
	if errorMessage != "" {
		result["error"] = errorMessage
	}
	if response != nil {
		result["response"] = response
	}
	out, _ := json.Marshal(result)
	w.WriteHeader(responseCode)
	w.Write(out)
}

func (e *Explorer) TablesList(w http.ResponseWriter, r *http.Request) {
	rows, err := e.db.Query("SHOW TABLES")
	defer rows.Close()

	if err != nil {
		ResponseWriter(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	tables := make([]string, 0)
	var table string
	for rows.Next() {
		rows.Scan(&table)
		tables = append(tables, table)
	}

	ResponseWriter(w, http.StatusOK, "", map[string]interface{}{"tables": tables})
}

func (e *Explorer) RecordsList(w http.ResponseWriter, r *http.Request) {
	variables := mux.Vars(r)

	// get fields
	table := &Table{Name: variables["table"]}
	fields, err := table.GetFields(*e)
	if err != nil {
		ResponseWriter(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// get record list
	limit, _ := strconv.Atoi(r.FormValue("limit"))
	if limit == 0 {
		limit = 5
	}
	offset, _ := strconv.Atoi(r.FormValue("offset"))

	rows, err := e.db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", variables["table"], limit, offset))
	if err != nil {
		ResponseWriter(w, http.StatusNotFound, errorUnknownTable, nil)
		return
	}
	defer rows.Close()

	records := []interface{}{}
	recordTemplate := table.CreateTemplate(fields)
	for rows.Next() {
		rows.Scan(recordTemplate...)
		record := make(map[string]interface{})
		for index, fieldTemplate := range recordTemplate {
			record[fields[index].Name] = table.FillField(fieldTemplate)
		}
		records = append(records, record)
	}

	ResponseWriter(w, http.StatusOK, "", map[string]interface{}{"records": records})
}

func (e *Explorer) RecordOne(w http.ResponseWriter, r *http.Request) {
	variables := mux.Vars(r)

	// get fields
	table := &Table{Name: variables["table"]}
	fields, err := table.GetFields(*e)
	if err != nil {
		ResponseWriter(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// get record
	id, _ := strconv.Atoi(variables["id"])
	row := e.db.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", table.Name, table.IdName), id)
	if err != nil {
		ResponseWriter(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	recordTemplate := table.CreateTemplate(fields)
	err = row.Scan(recordTemplate...)
	if err != nil {
		ResponseWriter(w, http.StatusNotFound, errorRecordNotFound, nil)
		return
	}
	record := make(map[string]interface{})
	for index, fieldTemplate := range recordTemplate {
		record[fields[index].Name] = table.FillField(fieldTemplate)
	}

	ResponseWriter(w, http.StatusOK, "", map[string]interface{}{"record": record})
}

func (e *Explorer) CreateRecord(w http.ResponseWriter, r *http.Request) {
	variables := mux.Vars(r)

	// get fields
	table := &Table{Name: variables["table"]}
	fields, err := table.GetFields(*e)
	if err != nil {
		ResponseWriter(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// prepare query
	decoder := json.NewDecoder(r.Body)
	requestParams := make(map[string]interface{})
	decoder.Decode(&requestParams)

	queryFields := make([]string, 0)
	placeholders := make([]string, 0)
	values := make([]interface{}, 0)
	for _, field := range fields {
		if field.IsPrimary {
			continue
		}
		queryFields = append(queryFields, field.Name)
		placeholders = append(placeholders, "?")
		if value, ok := requestParams[field.Name]; ok {
			err := table.ValidateField(field, value)
			if err != nil {
				ResponseWriter(w, http.StatusBadRequest, err.Error(), nil)
				return
			}
			values = append(values, value)
			continue
		}
		values = append(values, table.SetDefault(field))
	}

	// processing result
	res, err := e.db.Exec(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table.Name, strings.Join(queryFields, ","), strings.Join(placeholders, ",")), values...)
	if err != nil {
		ResponseWriter(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	id, _ := res.LastInsertId()
	ResponseWriter(w, http.StatusOK, "", map[string]interface{}{fmt.Sprintf("%s", table.IdName): id})
}

func (e *Explorer) UpdateURecord(w http.ResponseWriter, r *http.Request) {
	variables := mux.Vars(r)

	// get fields
	table := &Table{Name: variables["table"]}
	fields, err := table.GetFields(*e)
	if err != nil {
		ResponseWriter(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	decoder := json.NewDecoder(r.Body)
	requestParams := make(map[string]interface{})
	decoder.Decode(&requestParams)

	updatePlaceholder := make([]string, 0)
	values := make([]interface{}, 0)
	for _, field := range fields {
		if value, ok := requestParams[field.Name]; ok {
			err := table.ValidateField(field, value)
			if err != nil {
				ResponseWriter(w, http.StatusBadRequest, err.Error(), nil)
				return
			}
			updatePlaceholder = append(updatePlaceholder, fmt.Sprintf("%s = ?", field.Name))
			values = append(values, value)
		}
	}

	row, err := e.db.Exec(fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s", table.Name, strings.Join(updatePlaceholder, ","), table.IdName, variables["id"]), values...)
	if err != nil {
		ResponseWriter(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	id, _ := row.RowsAffected()
	ResponseWriter(w, http.StatusOK, "", map[string]interface{}{"updated": id})
}

func (e *Explorer) DeleteRecord(w http.ResponseWriter, r *http.Request) {
	variables := mux.Vars(r)
	table := &Table{Name: variables["table"]}

	row, err := e.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", table.Name), variables["id"])
	if err != nil {
		ResponseWriter(w, http.StatusInternalServerError, err.Error(), nil)
	}

	id, _ := row.RowsAffected()
	ResponseWriter(w, http.StatusOK, "", map[string]interface{}{"deleted": id})
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	explorer := &Explorer{db: db}
	serverMux := mux.NewRouter()

	serverMux.HandleFunc("/", explorer.TablesList).Methods("GET")
	serverMux.HandleFunc("/{table:[_a-z]+}", explorer.RecordsList).Methods("GET")
	serverMux.HandleFunc("/{table:[_a-z]+}/{id:[0-9]+}", explorer.RecordOne).Methods("GET")
	serverMux.HandleFunc("/{table:[_a-z]+}/", explorer.CreateRecord).Methods("PUT")
	serverMux.HandleFunc("/{table:[_a-z]+}/{id:[0-9]+}", explorer.UpdateURecord).Methods("POST")
	serverMux.HandleFunc("/{table:[_a-z]+}/{id:[0-9]+}", explorer.DeleteRecord).Methods("DELETE")

	return serverMux, nil
}
