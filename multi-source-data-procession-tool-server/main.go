package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/PaesslerAG/gval"
	"github.com/go-resty/resty/v2"
	"gofr.dev/pkg/gofr"
	"gopkg.in/yaml.v3"

	"github.com/IBM/sarama"
)

// Global map to store stop channels for each goroutine (using name as key)
var stopChannels = make(map[string]chan bool)
var mu sync.Mutex

// PartitionConsumer is a global variable to store the partition consumer
var PartitionConsumer sarama.PartitionConsumer

// Rule structure defines rules for transformation or filtering
type Rule struct {
	Key   string      // The key in the data
	Value interface{} // The value to compare against
	Op    string      // The operation (e.g., "==", ">", "<")
}

// Global variable for final output data after transformation
var finalOutputData = make(map[string]interface{})

// Structure for the final output data in JSON format
type finalOutputDataJSON struct {
	RuleType     string                `yaml:"RuleType" json:"RuleType"`
	OutputFormat []outputRuleStructure `yaml:"OutputFormat" json:"OutputFormat"`
}

// Structure for output rule with fields for key, display name, key type, rule, and nested structure
type outputRuleStructure struct {
	Key         string                `yaml:"Key" json:"Key"`
	DisplayName string                `yaml:"DisplayName" json:"DisplayName"`
	KeyType     string                `yaml:"KeyType" json:"KeyType"`
	Rule        string                `yaml:"Rule" json:"Rule"`
	Structure   []outputRuleStructure `yaml:"Structure" json:"Structure"`
}

// Configuration data structure with a list of data sources
type ConfigData struct {
	DataSourceConfig []DataSource `yaml:"SourceData" json:"SourceData"`
}

// Output format structure for customer and product
type OutputFormat struct {
	CustomerName string `json:"CName"`
	ProductID    string `json:"PID"`
}

// Data source structure with source, input name, config, and transformation config
type DataSource struct {
	Source               int                 `yaml:"Source" json:"Source"`
	InputName            string              `yaml:"NAME" json:"NAME"`
	Config               []Config            `yaml:"TYPEOF" json:"TYPEOF"`
	TransformationConfig finalOutputDataJSON `yaml:"TransformationConfig" json:"TransformationConfig"`
}

// Configuration structure for various data sources
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

// Sale record structure for customer sales data
type SaleRecord struct {
	ID           string  `json:"id"`
	CustomerName string  `json:"customer_name"`
	SaleAmount   float64 `json:"sale_amount"`
	SaleDate     string  `json:"sale_date"`
}

// Global variables for HTTP client and destination configuration
var client *resty.Client
var destinationConfig = make(map[int]DataSource)

// main function initializes the GoFr app and sets up routes
func main() {
	client = resty.New()
	app := gofr.New()

	processConfiguration("destinationConfig")
	processConfiguration("sourceConfig")

	// Set up API routes
	app.POST("/updateConfiguration", updateConfiguration)
	app.GET("/loadConfiguration", loadConfiguration)
	app.POST("/refreshConfiguration", refreshConfiguration)
	app.GET("/health", healthCheckHandler)

	// Run the app
	app.Run()
}

// updateConfiguration handles the creation of a configuration
func updateConfiguration(c *gofr.Context) (interface{}, error) {
	defer panicRecoveryMiddleware()

	var configType string
	var inputData ConfigData

	// Extract JSON body into the ConfigData struct
	configType = c.Param("configType")
	err := c.Bind(&inputData)
	if err != nil {
		log.Println("Error binding data:", err)
		return nil, err
	}

	log.Println("Received data:", inputData)

	// Define the file name based on the config type
	fileName := configType + ".yaml"
	if len(inputData.DataSourceConfig) > 0 {
		// Marshal the data into YAML format
		fileData, err := yaml.Marshal(inputData)
		if err != nil {
			log.Println("Error marshalling to YAML:", err)
			return nil, err
		}

		// Write the YAML data to a file
		err = ioutil.WriteFile(fileName, fileData, 0644)
		if err != nil {
			log.Println("Error writing to file:", err)
			return nil, err
		} else {
			// Set file permissions
			err := os.Chmod(fileName, 0771)
			if err != nil {
				log.Println("Failed to change file permission:", err)
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

	// Read the configuration file
	readData, err := ioutil.ReadFile(configType + ".yaml")
	if err != nil {
		log.Println("Failed to load config:", err)
		return nil, err
	}

	// Unmarshal YAML data into the ConfigData struct
	err = yaml.Unmarshal(readData, &inputData)
	if err != nil {
		log.Println("Error unmarshalling YAML:", err)
	}

	// Marshal the map to JSON
	jsonData, err := json.Marshal(inputData)
	if err != nil {
		log.Fatalf("Error marshalling to JSON: %v", err)
	}

	// Return the JSON as a string
	return string(jsonData), nil
}

// refreshConfiguration deploys the configuration based on the provided config type
func refreshConfiguration(c *gofr.Context) (interface{}, error) {
	defer panicRecoveryMiddleware()

	var configType string

	configType = c.Param("configType")
	log.Println("Config type:", configType)

	processConfiguration(configType)

	return "SUCCESSFUL", nil
}

func processConfiguration(configType string) {
	// Process the configuration data based on the config type
	// For example, if the config type is "database", you might create a database connection
	var incomingData ConfigData
	// Read the configuration file
	readDataFromPath, err := ioutil.ReadFile(configType + ".yaml")
	if err != nil {
		log.Println("Failed to load config:", err)
	}

	// Unmarshal YAML data into the ConfigData struct
	err = yaml.Unmarshal(readDataFromPath, &incomingData)
	if err != nil {
		log.Println("Error unmarshalling YAML:", err)
	}
	// Handling sourceConfig and destinationConfig
	if configType == "sourceConfig" {

		for _, sourceConfig := range incomingData.DataSourceConfig {
			for _, config := range sourceConfig.Config {
				stopWorker(config.Type)

				// Create a stop channel for this worker
				stopChan := make(chan bool)

				// Store the stop channel globally so it can be used to kill the worker
				mu.Lock()
				stopChannels["API"] = stopChan
				mu.Unlock()
				// Handle different config types (HTTP, DB, Kafka, etc.)
				switch config.Type {
				case "API":
					log.Println("HTTP input handler")
					go handleAPIInput(sourceConfig.Source, config, stopChan)
				case "KAFKA":
					log.Println("Kafka input handler")
					go startKafkaSubscription(sourceConfig.Source, config.IP, config.Port, config.TopicName, stopChan)
				default:
					// CSV
					go readInputFile(sourceConfig.Source, config.FilePath)
					log.Println("Unknown configuration type")
				}
			}
		}
	} else if configType == "destinationConfig" {

		log.Println("Destination configuration", incomingData)
		for _, sourceConfig := range incomingData.DataSourceConfig {
			// Store the source configuration in the global map
			destinationConfig[sourceConfig.Source] = sourceConfig
		}
	}

}

// startKafkaSubscription starts consuming messages from a Kafka topic
func startKafkaSubscription(sourceID int, ip string, port string, topicName string, stopChan chan bool) {
	// Set up Kafka consumer
	consumer, err := sarama.NewConsumer([]string{ip + ":" + port}, nil)
	if err != nil {
		log.Fatal("Failed to start Kafka consumer:", err)
	}
	defer consumer.Close()

	// Start consuming from the Kafka topic
	PartitionConsumer, err = consumer.ConsumePartition(topicName, 0, sarama.OffsetNewest)
	if err != nil {
		log.Fatal("Failed to start partition consumer:", err)
	}
	defer PartitionConsumer.Close()

	// Consume messages
	for message := range PartitionConsumer.Messages() {

		if config, ok := destinationConfig[sourceID]; ok {
			outputInHighLevelTransform(config.TransformationConfig.OutputFormat)
			respd, _ := TransformationINHighLevel(message.Value, finalOutputData, config.TransformationConfig.RuleType)
			log.Println("Request succeeded ", string(respd))
			go processData(sourceID, respd)
		}
	}
}

// processData processes incoming data from Kafka or other sources
func processData(sourceID int, data []byte) {
	log.Println("Processing data:", string(data))

	if config, ok := destinationConfig[sourceID]; ok {
		log.Println("Sending data to destination service")
		for _, cfg := range config.Config {
			switch cfg.Type {
			case "API":
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
				publishDataToAPIs(cfg.URL, data)
				log.Println("Unknown output type")
			}
		}
	}
}

func readInputFile(sourceID int, fileName string) {
	if config, ok := destinationConfig[sourceID]; ok {
		outputInHighLevelTransform(config.TransformationConfig.OutputFormat)

	}
	fmt.Println("Reading input file:", fileName)
	file, err := os.Open(fileName)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("Error reading CSV: %v", err)
		return
	}
	header := records[0]
	data := records[1:]
	fmt.Println("fgdfsgh", header)
	fmt.Println("fgdfsgh", data)
	var finalDataOutputList []map[string]interface{}
	for _, value := range data {
		reslutData := finalOutputData
		for keyData, _key := range reslutData {
			for headerIndex, record := range header {

				if strings.Contains(_key.(string), record) {

					reslutData[keyData] = value[headerIndex]
				}

			}
		}
		finalDataOutputList = append(finalDataOutputList, reslutData)
	}
	fmt.Println("fgdfsgh", finalDataOutputList)

	// Marshal the data into JSON format
	jsonData, err := json.Marshal(finalDataOutputList)
	if err != nil {
		fmt.Println(err)
	}
	go processData(sourceID, jsonData)
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
func handleDatabaseFetchData(Source int, config Config) {
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
	// Fetch rows
	var allRows []map[string]interface{}
	// Print rows
	for rows.Next() {
		columns, err := rows.Columns()
		if err != nil {
			log.Println("Error getting columns:", err)
		}

		// Make a slice for the values
		values := make([]sql.RawBytes, len(columns))

		// rows.Scan wants pointers to values, so we must copy
		// the references into such a slice
		// See http://go-database-sql.org/retrieving.html
		scanArgs := make([]interface{}, len(values))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		for rows.Next() {
			err = rows.Scan(scanArgs...)
			if err != nil {
				log.Println("Error scanning row:", err)
			}

			row := make(map[string]interface{})
			for i, col := range columns {
				row[col] = string(values[i])
			}
			allRows = append(allRows, row)
		}

	}
	// Marshal the data into JSON format
	jsonData, err := json.Marshal(allRows)
	if err != nil {
		fmt.Println(err)
	}
	finalOutputData = make(map[string]interface{})

	sourceConfig := destinationConfig[Source]
	outputInHighLevelTransform(sourceConfig.TransformationConfig.OutputFormat)
	respd, _ := TransformationINHighLevel(jsonData, finalOutputData, sourceConfig.TransformationConfig.RuleType)
	log.Println("Request succeeded with status 200", string(respd))
	go processData(Source, respd)
}

// handleAPIInput makes an HTTP request based on the provided configuration
func handleAPIInput(sourceID int, config Config, stopChan chan bool) {

	// Parse the duration
	duration, err := parseDuration(config.Duration)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		resp, err := client.R().Get(config.URL)
		if err != nil {
			log.Println("Error occurred:", err)
			return
		}

		log.Println("Response:", string(resp.Body()))

		finalOutputData = make(map[string]interface{})

		if resp.StatusCode() == 200 {

			if config, ok := destinationConfig[sourceID]; ok {
				outputInHighLevelTransform(config.TransformationConfig.OutputFormat)
				respd, _ := TransformationINHighLevel(resp.Body(), finalOutputData, config.TransformationConfig.RuleType)
				log.Println("Request succeeded with status 200", string(respd))
				go processData(sourceID, respd)
			}
		}
		select {
		case <-time.After(duration):

			fmt.Println("API is running...")
		case <-stopChan:
			// Stop signal received, exit the loop and terminate the goroutine
			fmt.Printf("%s is stopping...\n", "API")
			return
		}

	}
}

// publishDataToAPIs sends transformed data to an exrnal HTTP API
func publishDataToAPIs(url string, data []byte) error {

	// Make an HTTP POST request to the external API
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
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

func TransformationINHighLevel(input interface{}, output interface{}, ruleType string) ([]byte, error) {
	// Check the type of the input
	switch v := input.(type) {
	case string:
		// If the input is a string, check if it's valid JSON
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(v), &jsonData); err == nil {
			// If it's valid JSON, return as it is
			return json.Marshal(jsonData)
		}
		// If it's not valid JSON, return it as plain text in JSON
		return json.Marshal(map[string]interface{}{
			"data": v,
		})

	case []byte:
		var err error
		var data []map[string]interface{}
		if err = json.Unmarshal(v, &data); err == nil {
			fmt.Println("Input data as a csv")
			header := data[0]
			userData := data[1:]
			// If the input is a JSON array, check if it's valid JSON
			finalDataOutputList := make([]map[string]interface{}, 0)
			// value, err := gval.Evaluate(ruleType,
			// 	item)
			// if err != nil {
			// 	fmt.Println(err)
			// }
			if ruleType == "" {

				for _, value := range userData {
					reslutData := finalOutputData
					for keyData, _key := range reslutData {

						for index, _ := range header {
							if strings.Contains(_key.(string), "") {

								reslutData[keyData] = value[index]
							}
						}

					}
					finalDataOutputList = append(finalDataOutputList, reslutData)
				}
				// If it's valid JSON, return as it is
				return json.Marshal(finalDataOutputList)
			}
		}
		fmt.Println(err, "hdjfh")

		// If the input is a string, check if it's valid JSON
		var jsonData map[string]interface{}

		if err = json.Unmarshal([]byte(v), &jsonData); err == nil {

			if ruleType == "" {
				reslutData := finalOutputData
				for keyData, valueType := range reslutData {

					for key, value := range jsonData {

						if strings.Contains(valueType.(string), key) {
							reslutData[keyData] = value
						}
						fmt.Println("key", key, value)
					}
				}
				// If it's valid JSON, return as it is
				return json.Marshal(reslutData)
			}
		}
		if strings.Contains(err.Error(), "cannot unmarshal array") {
			// If it's not valid JSON, return it as plain text in JSON
			var jsonArrayData []map[string]interface{}
			if err = json.Unmarshal([]byte(v), &jsonArrayData); err == nil {
				var finalDataOutputList []map[string]interface{}

				for _, item := range jsonArrayData {
					value, err := gval.Evaluate(ruleType,
						item)
					if err != nil {
						fmt.Println(err)
					}
					if value == true || ruleType == "" {
						reslutData := finalOutputData
						for keyData, valueType := range reslutData {

							for key, value := range item {
								if strings.Contains(valueType.(string), key) {

									reslutData[keyData] = value

								}
								fmt.Println("key", key, value)
							}
							finalDataOutputList = append(finalDataOutputList, reslutData)
						}
					}

				}
				return json.Marshal(finalDataOutputList)
			}
		}
		fmt.Println("JSON Output (Binary Data Input):", err)
		return nil, nil

	default:
		// If the input is another type, try converting it into a JSON-friendly format
		return json.Marshal(map[string]interface{}{
			"data": fmt.Sprintf("Unsupported type: %v", reflect.TypeOf(input)),
		})
	}
	return nil, nil
}

func outputInHighLevelTransform(outputData []outputRuleStructure) map[string]interface{} {

	fmt.Println(outputData, "outputData")
	for _, data := range outputData {
		switch data.KeyType {
		case "STRING":
			finalOutputData[data.DisplayName] = data.Key
		case "INT":
			finalOutputData[data.DisplayName] = data.Key
		case "ARRAY_STRING":
			finalOutputData[data.DisplayName] = data.Key
		case "ARRAY_INT":
			finalOutputData[data.DisplayName] = data.Key
		case "ARRAY_STRUCT":
			var aryaInputMap []map[string]interface{}
			aryaInputMap = append(aryaInputMap, outputInHighLevelTransform(data.Structure))
			finalOutputData[data.DisplayName] = data.Key
		case "STRUCT":
			finalOutputData[data.DisplayName] = data.Key
		}
	}
	return finalOutputData
}
func evaluateComplexRule(data map[string]interface{}, expression string) (bool, error) {
	// Split the expression into individual conditions (for simplicity)
	conditions := strings.Split(expression, "&&")

	// Process each condition
	for _, condition := range conditions {
		condition = strings.TrimSpace(condition)
		// Each condition will be in the form of "key operator value"
		parts := strings.Fields(condition)
		if len(parts) != 3 {
			return false, fmt.Errorf("invalid condition: %s", condition)
		}

		key, operator, value := parts[0], parts[1], parts[2]
		rule := Rule{Key: key, Op: operator, Value: value}

		// Evaluate the individual rule
		result, err := evaluateRule(data, rule)
		if err != nil {
			return false, err
		}

		// If any condition is false, the entire expression is false
		if !result {
			return false, nil
		}
	}

	return true, nil
}

// Function to evaluate rules
func evaluateRule(data map[string]interface{}, rule Rule) (bool, error) {
	// Get the value from the data map
	value, exists := data[rule.Key]
	if !exists {
		return false, fmt.Errorf("key %s not found in data", rule.Key)
	}

	// Apply the operator
	switch rule.Op {
	case "==":
		// Check equality
		return value == rule.Value, nil
	case ">":
		// Check greater than for numeric values
		if v, ok := value.(float64); ok {
			if val, ok := rule.Value.(float64); ok {
				return v > val, nil
			}
		}
		return false, fmt.Errorf("invalid comparison for >")
	case "<":
		// Check less than for numeric values
		if v, ok := value.(float64); ok {
			if val, ok := rule.Value.(float64); ok {
				return v < val, nil
			}
		}
		return false, fmt.Errorf("invalid comparison for <")
	case "contains":
		// Check if a string contains another string
		if v, ok := value.(string); ok {
			if val, ok := rule.Value.(string); ok {
				return strings.Contains(v, val), nil
			}
		}
		return false, fmt.Errorf("invalid comparison for contains")
	default:
		return false, fmt.Errorf("unsupported operator %s", rule.Op)
	}
}

// Function to parse the time flag (e.g., "10m" for 10 minutes, "1h" for 1 hour)
func parseDuration(flag string) (time.Duration, error) {
	// Parse the time string into a time.Duration
	duration, err := time.ParseDuration(flag)
	if err != nil {
		return 0, fmt.Errorf("invalid time duration format: %v", err)
	}
	return duration, nil
}

// Function to stop a worker by name (closes the stop channel)
func stopWorker(name string) {
	mu.Lock()
	defer mu.Unlock()
	if stopChan, exists := stopChannels[name]; exists {
		close(stopChan)
		delete(stopChannels, name)
	}
}
