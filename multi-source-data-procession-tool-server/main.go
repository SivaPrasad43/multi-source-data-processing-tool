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
	"strconv"

	"github.com/go-resty/resty/v2"
	"gofr.dev/pkg/gofr"
	"gopkg.in/yaml.v3"

	"github.com/IBM/sarama"
)

type configDataStruct struct {
	DataSourceConfig []DataSource `yaml:"SourceData" json:"SourceData"`
}

type DataSource struct {
	Source int      `yaml:"Source" json:"Source"`
	Config []Config `yaml:"TYPEOF" json:"TYPEOF"`
}

type Config struct {
	TYPE       string `yaml:"TYPE" json:"TYPE"`
	DBTYPE     string `yaml:"DB_TYPE" json:"DB_TYPE"`
	DBHOST     string `yaml:"DB_HOST" json:"DB_HOST"`
	DBPORT     int    `yaml:"DB_PORT" json:"DB_PORT"`
	DBUSER     string `yaml:"DB_USER" json:"DB_USER"`
	DBPASSWORD string `yaml:"DB_PASSWORD" json:"DB_PASSWORD"`
	DBNAME     string `yaml:"DB_NAME" json:"DB_NAME"`
	TableName  string `yaml:"DB_TABLE_NAME" json:"DB_TABLE_NAME"`
	Duration   string `yaml:"Duration" json:"Duration"`
	FILEPATH   string `yaml:"FILE_PATH" json:"FILE_PATH"`
	S3BUCKET   string `yaml:"S3_BUCKET" json:"S3_BUCKET"`
	S3REGION   string `yaml:"S3_REGION" json:"S3_REGION"`
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

// main function initializes the GoFr app and sets up routes
func main() {

	// Create a new Resty client
	client = resty.New()

	app := gofr.New()

	app.POST("/createConfiguration", CreateConfiguration)

	app.GET("/loadConfiguration", LoadConfiguration)

	app.POST("/deployConfigData", DeployConfigData)

	// Health check route
	app.GET("/health", HealthCheckHandler)

	app.Run()
}

func HttpDataCall(config Config) {

	// Make an HTTP GET request
	resp, err := client.R().Get(config.URL) // A URL that will return 500 error
	if err != nil {
		fmt.Println("Error occurred:", err)
		return
	}

	fmt.Println("resp = ", resp.Body())

	if resp.StatusCode() == 200 {

	}

}

func CreateConfiguration(c *gofr.Context) (interface{}, error) {
	defer PanicRecoveryMiddleware()

	var CreateData configDataStruct
	// Extract JSON body into the 'Person' struct
	err := c.Bind(&CreateData)
	if err != nil {
		fmt.Print("errror = ", err)
		return nil, err
	}

	fmt.Println("CreateData = ", CreateData)

	if len(CreateData.DataSourceConfig) > 0 {
		Createdata, err := yaml.Marshal(CreateData)

		if err != nil {
			fmt.Println("error in yaml marshal = ", err)
		}

		err = ioutil.WriteFile("createdata.yaml", Createdata, 0644)
		if err != nil {
			fmt.Println("error in writing = ", err)
		}
	} else {
		return nil, errors.New("Empty data")
	}

	return "SUSSUSEFUL", err

}

func LoadConfiguration(c *gofr.Context) (interface{}, error) {

	defer PanicRecoveryMiddleware()

	readData, err := ioutil.ReadFile("createdata.yaml")
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		return nil, err
	}

	return string(readData), nil

}

func DeployConfigData(c *gofr.Context) (interface{}, error) {

	defer PanicRecoveryMiddleware()

	var CreateData configDataStruct
	// Extract JSON body into the 'Person' struct
	err := c.Bind(&CreateData)
	if err != nil {
		return nil, err
	}

	for _, conf := range CreateData.DataSourceConfig {

		for _, typeof := range conf.Config {
			switch typeof.TYPE {
			case "HTTP":
				fmt.Println("HTTP")
				HttpDataCall(typeof)
			case "FILE":
				fmt.Println("FILE")
			case "DB":
				fmt.Println("DB")
				DataBaseFetchData(typeof)
			case "KAFKA":
				fmt.Println("KAFKA")
				// RunningThreads[topic] = make(chan bool) // Create a stop channel for each topic
				startkafkaSubscription(conf.Source, typeof.IP, typeof.Port, typeof.TopicName)
			default:
				fmt.Println("Unknown")

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
			log.Printf("Recovered from panic: ", r)
		}
	}()

}

func startkafkaSubscription(sourceID int, ip string, port string, topicName string) {
	// Set up Kafka consumer
	consumer, err := sarama.NewConsumer([]string{ip + ":" + port}, nil)
	if err != nil {
		log.Fatal("Failed to start Kafka consumer:", err)
	}
	defer consumer.Close()

	// Start consuming from the "my-topic" topic
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

func processData(sourceID int, data []byte) {
	// Process the data here
	// For example, you can send it to a service using HTTP
	// or you can store it in a database
	// For this example, we'll just print it
	log.Println(string(data))

}
func publisDataToKafka(ip string, port string, topicName string, data string) {

	// Set up Kafka producer
	producer, err := sarama.NewSyncProducer([]string{ip + ":" + port}, nil)
	if err != nil {
		log.Fatal("Failed to start Kafka producer:", err)
	}
	defer producer.Close()

	// Create a message
	message := &sarama.ProducerMessage{
		Topic: topicName,
		Value: sarama.StringEncoder(data),
	}

	// Send the message to Kafka
	partition, offset, err := producer.SendMessage(message)
	if err != nil {
		log.Fatal("Failed to send message:", err)
	}

	fmt.Printf("Message sent to partition %d with offset %d\n", partition, offset)

}

func DataBaseFetchData(config Config) {
	// Panic handler
	defer PanicRecoveryMiddleware()

	// Define the MySQL data source name (DSN)
	// Format: username:password@tcp(hostname:port)/dbname
	dsn := config.DBUSER + ":" + config.DBPASSWORD + "@tcp(" + config.DBHOST + ":" + strconv.Itoa(config.DBPORT) + ")/" + config.DBNAME

	// Open the connection to the MySQL database
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println("Error opening database: ", err)
	}
	defer db.Close()

	// Test the connection to ensure everything is set up correctly
	err = db.Ping()
	if err != nil {
		fmt.Println("Error pinging database: ", err)
	}

	fmt.Println("Successfully connected to the MySQL database!")

	// Example of a simple SQL query
	var version string
	err = db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		fmt.Println("Error querying database: ", err)
	}
	fmt.Printf("MySQL version: %s\n", version)

	rows, err := db.Query("SELECT * FROM " + config.TableName)
	if err != nil {
		fmt.Println("Error querying users: ", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			fmt.Println("Error scanning row: ", err)
		}
		fmt.Printf("User: %d, Name: %s\n", id, name)
	}

}

