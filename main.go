package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type user struct {
	ID       primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name     string             `json:"name"`
	Email    string             `json:"email"`
	Password []byte             `json:"password"`
}

type post struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	UID       primitive.ObjectID `json:"uid,omitempty" bson:"uid,omitempty"`
	Caption   string             `json:"caption"`
	Image     string             `json:"image_url"`
	TimeStamp string             `json:"timestamp"`
}

func GetMongoDbConnection() (*mongo.Client, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}
	return client, nil
}

func getMongoDbCollection(DbName string, CollectionName string) (*mongo.Collection, error) {
	client, err := GetMongoDbConnection()
	if err != nil {
		return nil, err
	}
	collection := client.Database(DbName).Collection(CollectionName)
	return collection, nil
}

const dbName = "Appointy"
const userCollectionName = "users"
const postCollectionName = "posts"

func createUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	if r.Method == "POST" {
		var u user
		_ = json.NewDecoder(r.Body).Decode(&u)
		collection, _ := getMongoDbCollection(dbName, userCollectionName)
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		result, _ := collection.InsertOne(ctx, u)
		json.NewEncoder(w).Encode(result)
	}
}

func createPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	if r.Method == "POST" {
		var postTemp post
		_ = json.NewDecoder(r.Body).Decode(&postTemp)
		collection, _ := getMongoDbCollection(dbName, postCollectionName)
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		result, _ := collection.InsertOne(ctx, postTemp)
		json.NewEncoder(w).Encode(result)
	}
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("content-type", "application/json")
		name := strings.Replace(r.URL.Path, "/users/", "", 1)
		id, _ := primitive.ObjectIDFromHex(name)
		var userTemp user
		collection, _ := getMongoDbCollection(dbName, userCollectionName)
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&userTemp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{ "message": "` + err.Error() + `" }` + id.String()))
			return
		}
		json.NewEncoder(w).Encode(userTemp)
	}
}

func getPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("content-type", "application/json")
		name := strings.Replace(r.URL.Path, "/posts/", "", 1)
		id, _ := primitive.ObjectIDFromHex(name)
		var postTemp post
		collection, _ := getMongoDbCollection(dbName, postCollectionName)
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&postTemp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{ "message": "` + err.Error() + `" }` + id.String()))
			return
		}
		json.NewEncoder(w).Encode(postTemp)
	}
}

func getPostsByID(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("content-type", "application/json")
		name := strings.Replace(r.URL.Path, "/posts/users/", "", 1)
		id, _ := primitive.ObjectIDFromHex(name)
		var postTemp []post
		collection, _ := getMongoDbCollection(dbName, postCollectionName)
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		cursor, err := collection.Find(ctx, bson.M{"uid": id})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{ "message": "` + err.Error() + `" }` + id.String()))
			return
		}
		for cursor.Next(ctx) {
			var p post
			cursor.Decode(&p)
			postTemp = append(postTemp, p)
		}
		if err := cursor.Err(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{ "message": "` + err.Error() + `" }`))
			return
		}
		json.NewEncoder(w).Encode(postTemp)
	}
}

func createHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

func encrypt(data []byte, passphrase string) []byte {
	block, _ := aes.NewCipher([]byte(createHash(passphrase)))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext
}

func handleRequests() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNotFound)

		io.WriteString(w, "Not found\n")
	})
	http.HandleFunc("/users/", getUsers)
	http.HandleFunc("/users", createUser)
	http.HandleFunc("/posts/", getPosts)
	http.HandleFunc("/posts/users/", getPostsByID)
	http.HandleFunc("/posts", createPost)
	log.Fatal(http.ListenAndServe(":1000", nil))
}

func main() {
	handleRequests()
}
