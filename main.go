package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/go-resty/resty/v2"
	"gofr.dev/pkg/gofr"
	"gopkg.in/yaml.v3"
)

type SourceDataStruct struct {
	SourceData []SourceData `yaml:"SourceData" json:"SourceData"`
}

type TYPEOF struct {
	TYPE       string `yaml:"TYPE" json:"TYPE"`
	DBTYPE     string `yaml:"DB_TYPE" json:"DB_TYPE"`
	DBHOST     string `yaml:"DB_HOST" json:"DB_HOST"`
	DBPORT     int    `yaml:"DB_PORT" json:"DB_PORT"`
	DBUSER     string `yaml:"DB_USER" json:"DB_USER"`
	DBPASSWORD string `yaml:"DB_PASSWORD" json:"DB_PASSWORD"`
	DBNAME     string `yaml:"DB_NAME" json:"DB_NAME"`
	Duration   string `yaml:"Duration" json:"Duration"`
	FILEPATH   string `yaml:"FILE_PATH" json:"FILE_PATH"`
	S3BUCKET   string `yaml:"S3_BUCKET" json:"S3_BUCKET"`
	S3REGION   string `yaml:"S3_REGION" json:"S3_REGION"`
	URL        string `yaml:"URL" json:"URL"`
}
type SourceData struct {
	Source int      `yaml:"Source" json:"Source"`
	TYPEOF []TYPEOF `yaml:"TYPEOF" json:"TYPEOF"`
}

type SaleRecord struct {
	ID           string  `json:"id"`
	CustomerName string  `json:"customer_name"`
	SaleAmount   float64 `json:"sale_amount"`
	SaleDate     string  `json:"sale_date"`
}

var client *resty.Client

// main function initializes the GoFr app and sets up routes
func main() {
	app := gofr.New()

	// Create a new Resty client
	client = resty.New()

	app.POST("/createConfiguration", CreateConfiguration)

	app.GET("/readConfigyrationFile", ReadConfigyrationFile)

	app.POST("/deployConfigData", DeployConfigData)

	// Health check route
	app.GET("/health", HealthCheckHandler)

	app.Run()
}

func ConfigFileReading(c *gofr.Context) {

	var data SourceDataStruct
	config, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	err = yaml.Unmarshal(config, &data)
	if err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	fmt.Println("data = %v", data)

	for _, conf := range data.SourceData {

		if conf.Source == 1 {
			for _, typeof := range conf.TYPEOF {
				if typeof.TYPE == "HTTP" {
					HttpDataCall(typeof)
				}
			}
		}
	}

}

func HttpDataCall(config TYPEOF) {

	// Make an HTTP GET request
	resp, err := client.R().Get(config.URL) // A URL that will return 500 error
	if err != nil {
		fmt.Println("Error occurred:", err)
		return
	}

	fmt.Println("resp = %v", resp.Body())

	if resp.StatusCode() == 200 {

	}

}

func CreateConfiguration(c *gofr.Context) (interface{}, error) {
	defer PanicRecoveryMiddleware()

	var CreateData SourceDataStruct
	// Extract JSON body into the 'Person' struct
	err := c.Bind(&CreateData)
	if err != nil {
		fmt.Print("errror = ", err)
		return nil, err
	}

	fmt.Println("CreateData = %v", CreateData)

	if len(CreateData.SourceData) > 0 {
		Createdata, err := yaml.Marshal(CreateData)

		if err != nil {
			log.Fatalf("error in yaml marshal = %v", err)
		}

		err = ioutil.WriteFile("createdata.yaml", Createdata, 066)
		if err != nil {
			log.Fatalf("error in writing = %v", err)
		}
	}

	return "SUSSUSEFUL", err

}

func ReadConfigyrationFile(c *gofr.Context) (interface{}, error) {

	defer PanicRecoveryMiddleware()

	readData, err := ioutil.ReadFile("createdata.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	return string(readData), nil

}

func DeployConfigData(c *gofr.Context) (interface{}, error) {

	defer PanicRecoveryMiddleware()

	var CreateData SourceDataStruct
	// Extract JSON body into the 'Person' struct
	err := c.Bind(&CreateData)
	if err != nil {
		return nil, err
	}

	fmt.Println("CreateData = %v", CreateData)

	for _, conf := range CreateData.SourceData {

		if conf.Source == 1 {
			for _, typeof := range conf.TYPEOF {
				if typeof.TYPE == "HTTP" {
					HttpDataCall(typeof)
				}
			}
		}
	}

	return "SUSSUSEFUL", err

}

// sendToOutput sends transformed data to an external HTTP API
func sendToOutput(url string, data []SaleRecord) error {
	// Marshal the data into JSON format
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Make an HTTP POST request to the external API
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Error response from API: %s", string(bodyBytes))
		return err
	}

	return nil
}

// HealthCheckHandler is a basic route for checking the service status
func HealthCheckHandler(c *gofr.Context) (interface{}, error) {
	return "Service is up!", nil
}

// PanicRecoveryMiddleware handles panic and recovers gracefully
func PanicRecoveryMiddleware() {

	// Use defer to recover from panics
	defer func() {
		if r := recover(); r != nil {
			// Log the panic details
			log.Printf("Recovered from panic: %v", r)
		}
	}()

}
