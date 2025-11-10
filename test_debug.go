package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"rocksdb-cli/internal/api/handlers"
	"rocksdb-cli/internal/service"
)

func TestDebug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	manager := service.NewDBManager()
	handler := handlers.NewDBManagerHandler(manager)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/databases/list", nil)
	
	handler.ListAvailable(c)
	
	fmt.Println("Status:", w.Code)
	fmt.Println("Body:", w.Body.String())
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		fmt.Println("JSON Error:", err)
		return
	}
	
	fmt.Printf("Response: %+v\n", response)
	if data, ok := response["data"]; ok {
		fmt.Printf("Data: %+v\n", data)
		if dataMap, ok := data.(map[string]interface{}); ok {
			fmt.Printf("Databases: %+v\n", dataMap["databases"])
			fmt.Printf("MountPoints: %+v\n", dataMap["mountPoints"])
		}
	}
}
