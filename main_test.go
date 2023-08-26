package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func initMongoClient() (*mongo.Client, context.Context) {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return client, ctx
}
func TestServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintln(w, "Only GET requests are allowed")
			return
		}

		client, ctx := initMongoClient()
		defer client.Disconnect(ctx)

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
	}))
	defer ts.Close()

	testCases := []struct {
		desc           string
		query          string
		expectedStatus int
	}{
		{
			desc:           "Valid Request",
			query:          "?func=add&source=Facebook&receiver=123456789&info=code",
			expectedStatus: http.StatusOK,
		},
		{
			desc:           "Invalid Method",
			query:          "?func=add&source=Facebook&receiver=123456789&info=code",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			desc:           "Invalid Parameters",
			query:          "?func=invalid&source=123&receiver=abcdef&info=123",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			req, err := http.NewRequest("GET", ts.URL+tc.query, nil)
			if err != nil {
				t.Fatal(err)
			}

			resp := httptest.NewRecorder()
			ts.Config.Handler.ServeHTTP(resp, req)

			if resp.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, resp.Code)
			}
		})
	}
}
