package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Configuration struct {
	Mode string `json:"mode"`
}

var logger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	logger.Printf("%s %s %s", r.Method, r.URL, r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := []byte(`{"status":"ok"}`)
	w.Write(response)
}

func GetConfiguration(w http.ResponseWriter, r *http.Request) {
	logger.Printf("%s %s %s", r.Method, r.URL, r.RemoteAddr)
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
		log.Panicf("%s: %s", msg, err)
	}
}

func initialValues(conn *amqp.Connection) error {

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body := "Hello World!"
	err = ch.PublishWithContext(ctx,
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent %s\n", body)

	return nil
}

func main() {

	rabbitUrl, rabbitUrlExists := os.LookupEnv("RABBIT_URL")
	if !rabbitUrlExists {
		logger.Printf("RABBIT_URL not set, running api without rabbitmq connection.")
	} else {
		logger.Printf("RABBIT_URL: %s", rabbitUrl)
		conn, err := amqp.Dial(rabbitUrl)
		failOnError(err, "Failed to connect to RabbitMQ")
		defer conn.Close()

		errValues := initialValues(conn)
		failOnError(errValues, "Failed to initialize values")

	}

	//create a new router
	router := mux.NewRouter()

	// specify endpoints, handler functions and HTTP method
	router.HandleFunc("/healthz", HealthCheck).Methods("GET")
	router.HandleFunc("/api/v1/config", GetConfiguration).Methods("GET")
	http.Handle("/", router)

	// start and listen to requests
	log.Printf("Listening on port 8080...")
	http.ListenAndServe(":8080", router)
}
