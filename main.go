package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Data struct {
	Func     string `json:"func"`
	Source   string `json:"source"`
	Receiver string `json:"receiver"`
	Info     string `json:"info"`
}

var (
	mongoURI       = "mongodb://localhost:27017"
	databaseName   = "sms"
	collectionName = "sms-dumped"
)

func main() {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Only GET requests are allowed", http.StatusMethodNotAllowed)
			return
		}

		funcName := r.URL.Query().Get("func")
		source := r.URL.Query().Get("source")
		receiver := r.URL.Query().Get("receiver")
		info := r.URL.Query().Get("info")

		if funcName != "add" {
			http.Error(w, "Invalid 'func' parameter", http.StatusBadRequest)
			return
		}

		// Validate source, receiver, and info parameters using regular expressions
		validSource := regexp.MustCompile(`^[a-zA-Z\s]+$`)
		validReceiver := regexp.MustCompile(`^\d+$`)
		validInfo := regexp.MustCompile(`^[a-zA-Z\s]+$`)

		if !validSource.MatchString(source) || !validReceiver.MatchString(receiver) || !validInfo.MatchString(info) {
			http.Error(w, "Invalid parameter format", http.StatusBadRequest)
			return
		}

		data := Data{
			Func:     funcName,
			Source:   source,
			Receiver: receiver,
			Info:     info,
		}

		collection := client.Database(databaseName).Collection(collectionName)
		_, err := collection.InsertOne(ctx, data)
		if err != nil {
			http.Error(w, "Error inserting data into MongoDB", http.StatusInternalServerError)
			log.Println("Error inserting data into MongoDB:", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Data inserted into MongoDB")
	})

	fmt.Println("Server started on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
