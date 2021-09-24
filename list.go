package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

type ItemModel struct {
	Id          int `gorm:primary_key`
	Description string
	Completed   bool
}

var db, _ = gorm.Open(mysql.New(mysql.Config{
	DSN:                       "root:@tcp(127.0.0.1:3306)/go_lists?charset=utf8mb4&parseTime=True&loc=Local", // data source name
	DefaultStringSize:         256,                                                                           // default size for string fields
	DisableDatetimePrecision:  true,                                                                          // disable datetime precision, which not supported before MySQL 5.6
	DontSupportRenameIndex:    true,                                                                          // drop & create when rename index, rename index not supported before MySQL 5.7, MariaDB
	DontSupportRenameColumn:   true,                                                                          // `change` when rename column, rename column not supported before MySQL 8, MariaDB
	SkipInitializeWithVersion: false,                                                                         // auto configure based on currently MySQL version
}), &gorm.Config{})

func HealthZ(w http.ResponseWriter, r *http.Request) {
	logrus.Info("API Health is OK")
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{ "status": "success", "alive": "true" }
	json.NewEncoder(w).Encode(response)
}

func CreateItem(w http.ResponseWriter, r *http.Request) {
	description := r.FormValue("description")
	logrus.WithFields(logrus.Fields{"description": description}).Info("Add new Item. Saving to database.")
	item := &ItemModel{Description: description, Completed: false}
	db.Create(&item)
	result := db.Last(&item)
	response := map[string]interface{}{ "status": "success", "data": result }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func UpdateItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	err := GetItemById(id)
	w.Header().Set("Content-Type", "application/json")
	if err == false {
		json.NewEncoder(w).Encode(`{"updated": false, "error": "Record Not found"}`)
	} else {
		completed, _ := strconv.ParseBool(r.FormValue("completed"))
		description := r.FormValue("description")
		logrus.WithFields(logrus.Fields{"Id": id, "Completed": completed, "Description": description}).Info("Updating item")
		item := &ItemModel{}
		db.First(&item, id)
		if completed {
			item.Completed = completed
		} else {
			item.Completed = true
		}
		if description != "" {
			item.Description = description
		}
		db.Save(&item)
		response := map[string]interface{}{ "status": "success", "data": item }
		json.NewEncoder(w).Encode(response)
	}
}

func DeleteItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	err := GetItemById(id)
	w.Header().Set("Content-Type", "application/json")
	if err == false {
		response := map[string]interface{}{ "status": "failed", "error": "Record not found" }
		json.NewEncoder(w).Encode(response)
	} else {
		logrus.WithFields(logrus.Fields{"Id": id}).Info("Deleting item")
		item := &ItemModel{}
		db.First(&item, id)
		db.Delete(&item)
		response := map[string]interface{}{ "status": "success" }
		json.NewEncoder(w).Encode(response)
	}
}

func GetItemById(id int) bool {
	item := db.First(&ItemModel{}, id)
	if item.Error != nil {
		logrus.Warning(fmt.Sprintf(`Item %d not found in the database`, id))
		return false
	}
	return true
}

func GetCompletedItems(w http.ResponseWriter, r *http.Request) {
	logrus.Info("Get all incomplete items")
	Items := &ItemModel{}
	items := db.Where("completed = ?", 1).Find(Items)
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{ "status": "success", "data": items }
	json.NewEncoder(w).Encode(response)
}

func GetIncompletedItems(w http.ResponseWriter, r *http.Request) {
	logrus.Info("Get all incomplete items")
	items := db.Where("completed = ?", 0).Find(&ItemModel{})
	response := map[string]interface{}{ "status": "success", "data": items }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
func GetAllItems(w http.ResponseWriter, r *http.Request) {
	logrus.Info("Get all incomplete items")
	items := db.Find(&ItemModel{})
	response := map[string]interface{}{ "status": "success", "data": items }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
func GetItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	err := GetItemById(id)
	w.Header().Set("Content-Type", "application/json")
	if err == false {
		json.NewEncoder(w).Encode(`{"status": failed, "error": "Record Not found"}`)
	} else {
		logrus.WithFields(logrus.Fields{"Id": id}).Info("Deleting item")
		item := &ItemModel{}
		db.First(&item, id)
		response := map[string]interface{}{ "status": "success", "data": item }
		json.NewEncoder(w).Encode(response)
	}
}

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{})
	logrus.SetReportCaller(true)
}

func main() {

	//defer db.Close()

	if !db.Debug().Migrator().HasTable(&ItemModel{}) {
		err := db.Debug().Migrator().CreateTable(&ItemModel{})
		if err != nil {
			return
		}
	}

	logrus.Info("Starting TODO list server")
	router := mux.NewRouter()
	router.HandleFunc("/healthz", HealthZ).Methods("GET")
	router.HandleFunc("/items-incompleted", GetIncompletedItems).Methods("GET")
	router.HandleFunc("/items-completed", GetCompletedItems).Methods("GET")
	router.HandleFunc("/items", GetAllItems).Methods("GET")
	router.HandleFunc("/item", CreateItem).Methods("POST")
	router.HandleFunc("/item/{id}", GetItem).Methods("GET")
	router.HandleFunc("/item/{id}", UpdateItem).Methods("POST")
	router.HandleFunc("/item/{id}", DeleteItem).Methods("DELETE")
	http.ListenAndServe(":8000", router)
}
