package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/go-resty/resty/v2"
	"gofr.dev/pkg/gofr"
	"gopkg.in/yaml.v3"

	"github.com/IBM/sarama"
)

type ConfigData struct {
	DataSourceConfig []DataSource `yaml:"SourceData" json:"SourceData"`
}

type DataSource struct {
	Source    int      `yaml:"Source" json:"Source"`
	InputName string   `yaml:"NAME" json:"NAME"`
	Config    []Config `yaml:"TYPEOF" json:"TYPEOF"`
}

type Config struct {
	Type       string `yaml:"TYPE" json:"TYPE"`
	DBType     string `yaml:"DB_TYPE" json:"DB_TYPE"`
	DBHost     string `yaml:"DB_HOST" json:"DB_HOST"`
	DBPort     int    `yaml:"DB_PORT" json:"DB_PORT"`
	DBUser     string `yaml:"DB_USER" json:"DB_USER"`
	DBPassword string `yaml:"DB_PASSWORD" json:"DB_PASSWORD"`
	DBName     string `yaml:"DB_NAME" json:"DB_NAME"`
	TableName  string `yaml:"DB_TABLE_NAME" json:"DB_TABLE_NAME"`
	Duration   string `yaml:"Duration" json:"Duration"`
	FilePath   string `yaml:"FILE_PATH" json:"FILE_PATH"`
	S3Bucket   string `yaml:"S3_BUCKET" json:"S3_BUCKET"`
	S3Region   string `yaml:"S3_REGION" json:"S3_REGION"`
	URL        string `yaml:"URL" json:"URL"`
	IP         string `yaml:"IP" json:"IP"`
	Port       string `yaml:"Port" json:"Port"`
	TopicName  string `yaml:"TopicName" json:"TopicName"`
}

type SaleRecord struct {
	ID           string  `json:"id"`
	CustomerName string  `json:"customer_name"`
	SaleAmount   float64 `json:"sale_amount"`
	SaleDate     string  `json:"sale_date"`
}

var client *resty.Client
var destinationConfig map[int][]Config

// main function initializes the GoFr app and sets up routes
func main() {
	client = resty.New()
	app := gofr.New()

	// Set up API routes
	app.POST("/createConfiguration", createConfiguration)
	app.GET("/loadConfiguration", loadConfiguration)
	app.POST("/deployConfiguration", deployConfiguration)
	app.GET("/health", healthCheckHandler)

	// Run the app
	app.Run()
}

// createConfiguration handles the creation of a configuration
func createConfiguration(c *gofr.Context) (interface{}, error) {
	defer panicRecoveryMiddleware()

	var (
		configType string
		inputData  ConfigData
	)

	configType = c.Param("configType")

	// Extract JSON body into the ConfigData struct
	err := c.Bind(&inputData)
	if err != nil {
		log.Println("Error binding data:", err)
		return nil, err
	}

	fileName := configType + ".yaml"
	if len(inputData.DataSourceConfig) > 0 {
		fileData, err := yaml.Marshal(inputData)
		if err != nil {
			log.Println("Error marshalling to YAML:", err)
			return nil, err
		}

		err = ioutil.WriteFile(fileName, fileData, 0644)
		if err != nil {
			log.Println("Error writing to file:", err)
			return nil, err
		} else {
			// Set permission to 771 (owner: rwx, group: rwx, others: x)
			err := os.Chmod(fileName, 0771)
			if err != nil {
				log.Println("Failed to change file permission: %v", err)
			}
		}
	} else {
		return nil, errors.New("empty data")
	}

	return "SUCCESSFUL", nil
}

// loadConfiguration loads a configuration file based on the config type
func loadConfiguration(c *gofr.Context) (interface{}, error) {
	defer panicRecoveryMiddleware()

	var inputData ConfigData

	configType := c.Param("configType")

	// Read YAML configuration file
	readData, err := ioutil.ReadFile(configType + ".yaml")
	if err != nil {
		log.Println("Failed to load config:", err)
		return nil, err
	}

	err = yaml.Unmarshal(readData, &inputData)
	if err != nil {
		log.Println("Error unmarshalling YAML: ", err)
	}

	fmt.Println("inputData = ", inputData)
	// Marshal the map to JSON
	jsonData, err := json.Marshal(inputData)
	if err != nil {
		log.Fatalf("Error marshalling to JSON: %v", err)
	}

	// Print the JSON output
	fmt.Println(string(jsonData))

	return string(jsonData), nil
}

// deployConfiguration deploys the configuration based on the provided config type
func deployConfiguration(c *gofr.Context) (interface{}, error) {
	defer panicRecoveryMiddleware()

	var (
		configType   string
		incomingData ConfigData
	)

	configType = c.Param("configType")
	log.Println("Config type:", configType)

	// Extract JSON body into the ConfigData struct
	err := c.Bind(&incomingData)
	if err != nil {
		return nil, err
	}

	if configType == "sourceConfig" {
		for _, sourceConfig := range incomingData.DataSourceConfig {
			for _, config := range sourceConfig.Config {
				switch config.Type {
				case "HTTP":
					log.Println("HTTP input handler")
					handleAPIInput(config)
				case "FILE":
					log.Println("File input handler")
				case "DB":
					log.Println("Database input handler")
					handleDatabaseFetchData(config)
				case "KAFKA":
					log.Println("Kafka input handler")
					startKafkaSubscription(sourceConfig.Source, config.IP, config.Port, config.TopicName)
				default:
					log.Println("Unknown configuration type")
				}
			}
		}
	} else if configType == "destinationConfig" {
		log.Println("Destination configuration")
		for _, sourceConfig := range incomingData.DataSourceConfig {
			destinationConfig[sourceConfig.Source] = sourceConfig.Config
		}
	}

	return "SUCCESSFUL", nil
}

// startKafkaSubscription starts consuming messages from a Kafka topic
func startKafkaSubscription(sourceID int, ip string, port string, topicName string) {
	// Set up Kafka consumer
	consumer, err := sarama.NewConsumer([]string{ip + ":" + port}, nil)
	if err != nil {
		log.Fatal("Failed to start Kafka consumer:", err)
	}
	defer consumer.Close()

	// Start consuming from the Kafka topic
	partitionConsumer, err := consumer.ConsumePartition(topicName, 0, sarama.OffsetNewest)
	if err != nil {
		log.Fatal("Failed to start partition consumer:", err)
	}
	defer partitionConsumer.Close()

	// Consume messages
	for message := range partitionConsumer.Messages() {
		go processData(sourceID, message.Value)
	}
}

// processData processes incoming data from Kafka or other sources
func processData(sourceID int, data []byte) {
	log.Println("Processing data:", string(data))

	if config, ok := destinationConfig[sourceID]; ok {
		log.Println("Sending data to destination service")
		for _, cfg := range config {
			switch cfg.Type {
			case "HTTP":
				log.Println("HTTP output handler")
				publishDataToAPIs(cfg.URL, data)
			case "FILE":
				log.Println("File output handler")
			case "DB":
				log.Println("Database output handler")
			case "KAFKA":
				log.Println("Kafka output handler")
				publishDataToKafka(cfg.IP, cfg.Port, cfg.TopicName, string(data))
			default:
				log.Println("Unknown output type")
			}
		}
	}
}

// publishDataToKafka sends data to a Kafka topic
func publishDataToKafka(ip string, port string, topicName string, data string) {
	producer, err := sarama.NewSyncProducer([]string{ip + ":" + port}, nil)
	if err != nil {
		log.Fatal("Failed to start Kafka producer:", err)
	}
	defer producer.Close()

	// Create a Kafka message
	message := &sarama.ProducerMessage{
		Topic: topicName,
		Value: sarama.StringEncoder(data),
	}

	// Send the message to Kafka
	partition, offset, err := producer.SendMessage(message)
	if err != nil {
		log.Fatal("Failed to send message:", err)
	}

	log.Printf("Message sent to partition %d with offset %d", partition, offset)
}

// handleDatabaseFetchData fetches data from a database based on configuration
func handleDatabaseFetchData(config Config) {
	defer panicRecoveryMiddleware()

	// Create DSN string for MySQL connection
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", config.DBUser, config.DBPassword, config.DBHost, config.DBPort, config.DBName)

	// Open the database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Println("Error opening database:", err)
		return
	}
	defer db.Close()

	// Test the database connection
	err = db.Ping()
	if err != nil {
		log.Println("Error pinging database:", err)
		return
	}

	log.Println("Successfully connected to the MySQL database!")

	// Query the database
	var version string
	err = db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		log.Println("Error querying database:", err)
		return
	}
	log.Printf("MySQL version: %s", version)

	rows, err := db.Query("SELECT * FROM " + config.TableName)
	if err != nil {
		log.Println("Error querying database:", err)
		return
	}
	defer rows.Close()

	// Print rows
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			log.Println("Error scanning row:", err)
		}
		log.Printf("User: %d, Name: %s", id, name)
	}
}

// handleAPIInput makes an HTTP request based on the provided configuration
func handleAPIInput(config Config) {
	resp, err := client.R().Get(config.URL)
	if err != nil {
		log.Println("Error occurred:", err)
		return
	}

	log.Println("Response:", resp.Body())

	if resp.StatusCode() == 200 {
		log.Println("Request succeeded with status 200")
	}
}

// publishDataToAPIs sends transformed data to an external HTTP API
func publishDataToAPIs(url string, data []byte) error {
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

// healthCheckHandler is a basic health check route
func healthCheckHandler(c *gofr.Context) (interface{}, error) {
	return "Service is up!", nil
}

// panicRecoveryMiddleware recovers from panics and logs the error
func panicRecoveryMiddleware() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
		}
	}()
}
