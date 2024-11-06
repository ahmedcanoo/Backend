package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type User struct {
	Name     string             `json:"name"`
	Email    string             `json:"email"`
	Phone    string             `json:"phone"`
	Password string             `json:"password"`
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
}

type Order struct {
	PickupLocation  string             `json:"pickupLocation"`
	DropOffLocation string             `json:"dropOffLocation"`
	PackageDetails  string             `json:"packageDetails"`
	DeliveryTime    string             `json:"deliveryTime"`
	Status          string             `json:"status"`
	UserID          primitive.ObjectID `bson:"userId" json:"userId"`
}

var client *mongo.Client

func main() {
	// Set up MongoDB connection
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	var err error
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/register", RegisterUser).Methods("POST")
	router.HandleFunc("/api/users", GetUsers).Methods("GET")
	router.HandleFunc("/api/login", LoginUser).Methods("POST")
	router.HandleFunc("/api/orders", CreateOrder).Methods("POST")
	router.HandleFunc("/api/orders", GetOrders).Methods("GET")
	router.HandleFunc("/api/orders/{id}", GetOrderDetails).Methods("GET")
	router.HandleFunc("/api/orders/{id}/cancel", CancelOrder).Methods("DELETE")

	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"http://localhost:3000"}),
		handlers.AllowedMethods([]string{"POST", "GET", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type"}),
		handlers.AllowedHeaders([]string{"Content-Type", "userId"}),
	)

	log.Fatal(http.ListenAndServe(":8001", corsHandler(router)))
}

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	collection := client.Database("myapp").Collection("users")

	// Check if user already exists
	var existingUser User
	err = collection.FindOne(context.TODO(), bson.M{"email": user.Email}).Decode(&existingUser)
	if err == nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	// Insert new user into the database
	_, err = collection.InsertOne(context.TODO(), user)
	if err != nil {
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode("User registered successfully")
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	collection := client.Database("myapp").Collection("users")

	cursor, err := collection.Find(context.TODO(), bson.D{}) // Retrieve all users
	if err != nil {
		http.Error(w, "Failed to retrieve users", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	var users []User
	if err = cursor.All(context.TODO(), &users); err != nil {
		http.Error(w, "Failed to decode users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func LoginUser(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	collection := client.Database("myapp").Collection("users")

	var existingUser User
	err = collection.FindOne(context.TODO(), bson.M{"email": user.Email}).Decode(&existingUser)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	if existingUser.Password != user.Password {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := map[string]string{
		"userId":  existingUser.ID.Hex(),
		"message": "Login successful!",
	}
	json.NewEncoder(w).Encode(response)
}

func CreateOrder(w http.ResponseWriter, r *http.Request) {
	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	userIDStr := r.Header.Get("userId")
	if userIDStr == "" {
		http.Error(w, "UserID is required", http.StatusBadRequest)
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		http.Error(w, "Invalid UserID format", http.StatusBadRequest)
		return
	}

	order.UserID = userID

	collection := client.Database("myapp").Collection("orders")

	// Insert order into the database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, order)
	if err != nil {
		http.Error(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "Order created successfully",
		"orderId": result.InsertedID,
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func GetOrders(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("userId")

	if userIDStr == "" {
		http.Error(w, "UserID is required", http.StatusBadRequest)
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		http.Error(w, "Invalid UserID format", http.StatusBadRequest)
		return
	}

	collection := client.Database("myapp").Collection("orders")

	filter := bson.M{"userId": userID}
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}

	var orders []Order
	if err := cursor.All(context.TODO(), &orders); err != nil {
		http.Error(w, "Error processing orders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}
func GetOrderDetails(w http.ResponseWriter, r *http.Request) {

	orderIDStr := mux.Vars(r)["id"]
	orderID, err := primitive.ObjectIDFromHex(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid OrderID format", http.StatusBadRequest)
		return
	}

	collection := client.Database("myapp").Collection("orders")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Retrieve the order details
	var order Order
	err = collection.FindOne(ctx, bson.M{"_id": orderID}).Decode(&order)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(order)
}

func CancelOrder(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	orderIDStr, exists := vars["id"]
	if !exists {
		http.Error(w, "OrderID is required", http.StatusBadRequest)
		return
	}

	orderID, err := primitive.ObjectIDFromHex(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid OrderID format", http.StatusBadRequest)
		return
	}

	collection := client.Database("myapp").Collection("orders")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deleteResult, err := collection.DeleteOne(ctx, bson.M{"_id": orderID})
	if err != nil || deleteResult.DeletedCount == 0 {
		http.Error(w, "Order not found or could not be deleted", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("Order canceled successfully")
}
