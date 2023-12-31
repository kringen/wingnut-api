package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	rmq "github.com/kringen/message-center/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Configuration struct {
	Mode      string `json:"mode"`
	Objective string `json:"objective"`
}

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	logger.Info(fmt.Sprintf("%s %s %s", r.Method, r.URL, r.RemoteAddr))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := []byte(`{"status":"ok"}`)
	w.Write(response)
}

func CreateConfiguration(w http.ResponseWriter, r *http.Request) {
	logger.Info(fmt.Sprintf("%s %s %s", r.Method, r.URL, r.RemoteAddr))
	var config Configuration
	json.NewDecoder(r.Body).Decode(&config)
	// Connect to the channel
	// Establish messaging connection
	messageCenter := rmq.MessageCenter{}
	// Define RabbitMQ server URL.
	messageCenter.ServerUrl = os.Getenv("RABBIT_URL")
	channelName := "wingnut"
	err := messageCenter.Connect(channelName, 5, 5)
	if err != nil {
		panic(err)
	}
	defer messageCenter.Connection.Close()
	defer messageCenter.Channel.Close()
	// Create a test message
	b, err := json.Marshal(config)
	if err != nil {
		panic(err)
	}
	publishMessage(&messageCenter, "config", b)
	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := []byte(`{"status":"ok"}`)
	w.Write(response)
}

func GetConfiguration(w http.ResponseWriter, r *http.Request) {
	logger.Info(fmt.Sprintf("%s %s %s", r.Method, r.URL, r.RemoteAddr))
	config := Configuration{
		Mode: "sleeping",
	}
	configData, err := json.Marshal(config)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(configData)

}

func failOnError(err error, msg string) {
	if err != nil {
		logger.Error(fmt.Sprintf("%s: %s", msg, err))
	}
}

func publishMessage(messageCenter *rmq.MessageCenter, q string, message []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var err error
	err = messageCenter.Channel.PublishWithContext(ctx,
		"",    // exchange
		q,     // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        message,
		})
	failOnError(err, "Failed to publish a message")
	logger.Info(fmt.Sprintf(" [x] Sent %s\n", message))
}

func ConsumeMessages(chanConsumeMessages chan string, messageCenter *rmq.MessageCenter, queue string) {
	// Subscribing to QueueService1 for getting messages.
	messages, err := messageCenter.Channel.Consume(
		queue, // queue name
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no local
		false, // no wait
		nil,   // arguments
	)
	if err != nil {
		logger.Error(err.Error())
	}

	// Build a welcome message.
	logger.Info("Successfully connected to RabbitMQ")
	logger.Info("Waiting for messages")
	for message := range messages {
		// For example, show received message in a console.
		logger.Info(fmt.Sprintf(" > Received message: %s\n", message.Body))
	}
}

func main() {

	//create a new router
	router := mux.NewRouter()

	// specify endpoints, handler functions and HTTP method
	router.HandleFunc("/healthz", HealthCheck).Methods("GET")
	router.HandleFunc("/api/v1/config", GetConfiguration).Methods("GET")
	router.HandleFunc("/api/v1/config", CreateConfiguration).Methods("POST")
	http.Handle("/", router)

	// start and listen to requests
	logger.Info("Listening on port 8080...")
	http.ListenAndServe(":8080", router)
	/*
		rabbitUrl, rabbitUrlExists := os.LookupEnv("RABBIT_URL")
		if !rabbitUrlExists {
			logger.Printf("RABBIT_URL not set, running api without rabbitmq connection.")
		} else {
			logger.Printf("RABBIT_URL: %s", rabbitUrl)
			// Create a connection
			conn, err := connect(rabbitUrl)
			failOnError(err, "Failed to connect to RabbitMQ")
			defer conn.Close()
			// Open Channel
			ch, err := conn.Channel()
			failOnError(err, "Failed to open a channel")
			defer ch.Close()
			// Create a queue
			logger.Printf("Creating queue...")
			//_, err = createQueue(ch, "mode", false, false, false, false, nil)
			//failOnError(err, "Failed to declare a queue")
			// Create a test message
			//queue := amqp.Queue{
			//	Name: "mode",
			//}
			/*
			publishMessage(ch, &queue, "This is only a test.")
			for i := 0; i < 100; i++ {
				publishMessage(ch, &queue, time.Now().Format("20060102150405"))
				time.Sleep(5 * time.Second)

			}

		}*/

}
