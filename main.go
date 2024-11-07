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
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PickupLocation  string             `json:"pickupLocation"`
	DropOffLocation string             `json:"dropOffLocation"`
	PackageDetails  string             `json:"packageDetails"`
	DeliveryTime    string             `json:"deliveryTime"`
	Status          string             `json:"status"`
	UserID          primitive.ObjectID `bson:"userId" json:"userId"`

	UserName        string             `json:"userName,omitempty"`
	CourierEmail    string             `json:"courierEmail,omitempty" bson:"courierEmail,omitempty"`
	CourierPhone    string             `json:"courierPhone,omitempty" bson:"courierPhone,omitempty"`
	CourierName     string             `json:"courierName,omitempty" bson:"courierName,omitempty"`
}
type Courier struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Email       string             `json:"email" bson:"email"`
	Phone       string             `json:"phone" bson:"phone"`
	Password    string             `json:"password" bson:"password"`
	VehicleType string             `json:"vehicleType" bson:"vehicleType"`
	PlateNumber string             `json:"plateNumber" bson:"plateNumber"`
}
type Admin struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name     string             `json:"name" bson:"name"`
	Email    string             `json:"email" bson:"email"`
	Password string             `json:"password" bson:"password"`
	Role     string             `json:"role" bson:"role"`

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
	//InsertAdminUser()
	router := mux.NewRouter()
	//User
	router.HandleFunc("/api/register", RegisterUser).Methods("POST")
	router.HandleFunc("/api/users", GetUsers).Methods("GET")
	router.HandleFunc("/api/login", LoginUser).Methods("POST")
	router.HandleFunc("/api/orders", CreateOrder).Methods("POST")
	router.HandleFunc("/api/orders", GetOrders).Methods("GET")
	router.HandleFunc("/api/orders/{id}", GetOrderDetails).Methods("GET")
	router.HandleFunc("/api/orders/{id}/cancel", CancelOrder).Methods("DELETE")

	//Courier
	router.HandleFunc("/api/register-courier", RegisterCourier).Methods("POST")
	router.HandleFunc("/api/login-courier", LoginCourier).Methods("POST")
	router.HandleFunc("/api/courier/orders", GetOrdersAssignedToCourier).Methods("GET")
	//Admin
	router.HandleFunc("/api/admin/login", LoginAdmin).Methods("POST")
	router.HandleFunc("/api/admin/orders", GetAllOrders).Methods("GET")
	router.HandleFunc("/api/admin/orders/{id}/status", UpdateOrderStatus).Methods("PUT")
	router.HandleFunc("/api/admin/orders/{id}", DeleteOrder).Methods("DELETE")
	router.HandleFunc("/api/admin/orders/{orderId}/assign-courier", AssignCourierToOrder).Methods("POST")
	router.HandleFunc("/api/admin/orders/{orderId}/reassign-courier", ReassignCourierToOrder).Methods("PUT")

	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"http://localhost:3000"}),
		handlers.AllowedMethods([]string{"POST", "GET", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type"}),
		handlers.AllowedHeaders([]string{"Content-Type", "userId"}),
	)

	log.Fatal(http.ListenAndServe(":8001", corsHandler(router)))
}
func InsertAdminUser() {
	admin := Admin{
		Name:     "admin",
		Email:    "admin@yahoo.com",
		Password: "admin",
		Role:     "admin",
	}

	collection := client.Database("myapp").Collection("users")
	_, err := collection.InsertOne(context.TODO(), admin)
	if err != nil {
		log.Fatal("Failed to insert admin user:", err)
	} else {
		log.Println("Admin user inserted successfully")
	}
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

	// Convert userID from string to ObjectID
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		http.Error(w, "Invalid UserID format", http.StatusBadRequest)
		return
	}

	// Fetch orders for the user
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

	// Fetch the user's name from the "users" collection
	userCollection := client.Database("myapp").Collection("users")
	var user User
	err = userCollection.FindOne(context.TODO(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Add user's name to each order
	for i := range orders {
		orders[i].UserName = user.Name // Add user name to each order
	}

	// Fetch courier information for each order
	for i := range orders {
		if orders[i].CourierEmail != "" {
			courierCollection := client.Database("myapp").Collection("couriers")
			var courier Courier
			err := courierCollection.FindOne(context.TODO(), bson.M{"email": orders[i].CourierEmail}).Decode(&courier)
			if err == nil {
				// Add courier details to the order if the courier exists
				orders[i].CourierName = courier.Name
				orders[i].CourierPhone = courier.Phone
			}
		}
	}

	// Send the orders response with the user's and courier's names included
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

	// Fetch order from database
	collection := client.Database("myapp").Collection("orders")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
	orderIDStr, exists := mux.Vars(r)["id"]
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

// Courier Featuers

func RegisterCourier(w http.ResponseWriter, r *http.Request) {
	var courier Courier
	err := json.NewDecoder(r.Body).Decode(&courier)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	collection := client.Database("myapp").Collection("couriers")

	var existingCourier Courier
	err = collection.FindOne(context.TODO(), bson.M{"email": courier.Email}).Decode(&existingCourier)
	if err == nil {
		http.Error(w, "Courier already exists", http.StatusConflict)
		return
	}

	_, err = collection.InsertOne(context.TODO(), courier)
	if err != nil {
		http.Error(w, "Failed to register courier", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode("Courier registered successfully")
}

func LoginCourier(w http.ResponseWriter, r *http.Request) {
	var courier Courier
	err := json.NewDecoder(r.Body).Decode(&courier)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	collection := client.Database("myapp").Collection("couriers")

	var existingCourier Courier
	err = collection.FindOne(context.TODO(), bson.M{"email": courier.Email}).Decode(&existingCourier)
	if err != nil {
		http.Error(w, "Courier not found", http.StatusUnauthorized)
		return
	}

	if existingCourier.Password != courier.Password {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := map[string]string{
		"message":  "Login successful!",
		"username": existingCourier.Name,
	}
	json.NewEncoder(w).Encode(response)
}
func AcceptOrder(w http.ResponseWriter, r *http.Request) {
	orderIDStr := mux.Vars(r)["orderId"]
	orderID, err := primitive.ObjectIDFromHex(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid OrderID format", http.StatusBadRequest)
		return
	}

	var request struct {
		Email string `json:"email"`
	}

	// Decode request body to get courier email
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Fetch courier by email
	var courier Courier
	collection := client.Database("myapp").Collection("couriers")
	err = collection.FindOne(context.TODO(), bson.M{"email": request.Email}).Decode(&courier)
	if err != nil {
		http.Error(w, "Courier not found", http.StatusNotFound)
		return
	}

	// Fetch the order
	orderCollection := client.Database("myapp").Collection("orders")
	var order Order
	err = orderCollection.FindOne(context.TODO(), bson.M{"_id": orderID}).Decode(&order)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Check if the order is already accepted
	if order.Status == "Accepted" {
		http.Error(w, "Order is already accepted", http.StatusConflict)
		return
	}

	// Update order with courier's details and change the status
	filter := bson.M{"_id": orderID}
	update := bson.M{
		"$set": bson.M{
			"courierId":    courier.ID,
			"courierEmail": courier.Email,
			"courierPhone": courier.Phone,
			"courierName":  courier.Name,
			"status":       "Accepted", // Change status to Accepted
		},
	}

	// Perform the update in the database
	_, err = orderCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		http.Error(w, "Failed to accept the order", http.StatusInternalServerError)
		return
	}

	// Respond with success
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("Order accepted successfully")
}
func DeclineOrder(w http.ResponseWriter, r *http.Request) {
	orderIDStr := mux.Vars(r)["orderId"]
	orderID, err := primitive.ObjectIDFromHex(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid OrderID format", http.StatusBadRequest)
		return
	}

	var request struct {
		Email string `json:"email"`
	}

	// Decode request body to get courier email
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Fetch courier by email
	var courier Courier
	collection := client.Database("myapp").Collection("couriers")
	err = collection.FindOne(context.TODO(), bson.M{"email": request.Email}).Decode(&courier)
	if err != nil {
		http.Error(w, "Courier not found", http.StatusNotFound)
		return
	}

	// Fetch the order
	orderCollection := client.Database("myapp").Collection("orders")
	var order Order
	err = orderCollection.FindOne(context.TODO(), bson.M{"_id": orderID}).Decode(&order)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Update the order status to "Declined"
	filter := bson.M{"_id": orderID}
	update := bson.M{
		"$set": bson.M{
			"status": "Declined", // Change status to Declined
		},
	}

	// Perform the update in the database
	_, err = orderCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		http.Error(w, "Failed to decline the order", http.StatusInternalServerError)
		return
	}

	// Respond with success
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("Order declined successfully")
}

// ADMIN Features
func LoginAdmin(w http.ResponseWriter, r *http.Request) {
	var admin User
	err := json.NewDecoder(r.Body).Decode(&admin)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	collection := client.Database("myapp").Collection("users")

	var existingAdmin User
	err = collection.FindOne(context.TODO(), bson.M{"email": admin.Email, "role": "admin"}).Decode(&existingAdmin)
	if err != nil {
		http.Error(w, "Admin not found", http.StatusUnauthorized)
		return
	}

	if existingAdmin.Password != admin.Password {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := map[string]string{
		"adminId": existingAdmin.ID.Hex(),
		"message": "Admin login successful!",
	}
	json.NewEncoder(w).Encode(response)
}
func GetAllOrders(w http.ResponseWriter, r *http.Request) {
	collection := client.Database("myapp").Collection("orders")

	cursor, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		http.Error(w, "Failed to retrieve orders", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	var orders []Order
	if err := cursor.All(context.TODO(), &orders); err != nil {
		http.Error(w, "Failed to decode orders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(orders)
}

func UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderIDStr := mux.Vars(r)["id"]
	orderID, err := primitive.ObjectIDFromHex(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid OrderID format", http.StatusBadRequest)
		return
	}

	var update struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	collection := client.Database("myapp").Collection("orders")
	filter := bson.M{"_id": orderID}
	updateData := bson.M{"$set": bson.M{"status": update.Status}}

	_, err = collection.UpdateOne(context.TODO(), filter, updateData)
	if err != nil {
		http.Error(w, "Failed to update order", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("Order status updated successfully")
}

func DeleteOrder(w http.ResponseWriter, r *http.Request) {
	orderIDStr := mux.Vars(r)["id"]
	orderID, err := primitive.ObjectIDFromHex(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid OrderID format", http.StatusBadRequest)
		return
	}

	collection := client.Database("myapp").Collection("orders")

	result, err := collection.DeleteOne(context.TODO(), bson.M{"_id": orderID})
	if err != nil {
		http.Error(w, "Failed to delete order", http.StatusInternalServerError)
		return
	}

	if result.DeletedCount == 0 {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("Order deleted successfully")
}
func AssignCourierToOrder(w http.ResponseWriter, r *http.Request) {
	orderIDStr := mux.Vars(r)["orderId"]
	orderID, err := primitive.ObjectIDFromHex(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid OrderID format", http.StatusBadRequest)
		return
	}

	var request struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	var courier Courier
	collection := client.Database("myapp").Collection("couriers")
	err = collection.FindOne(context.TODO(), bson.M{"email": request.Email}).Decode(&courier)
	if err != nil {
		http.Error(w, "Courier not found", http.StatusNotFound)
		return
	}

	orderCollection := client.Database("myapp").Collection("orders")
	var order Order
	err = orderCollection.FindOne(context.TODO(), bson.M{"_id": orderID}).Decode(&order)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	if order.CourierEmail != "" {
		http.Error(w, "Order is already assigned to a courier", http.StatusConflict)
		return
	}

	filter := bson.M{"_id": orderID}
	update := bson.M{
		"$set": bson.M{
			"courierId":    courier.ID,
			"courierEmail": courier.Email,
			"courierPhone": courier.Phone,
			"courierName":  courier.Name,
			"reassigned":   false,
		},
	}

	_, err = orderCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		http.Error(w, "Failed to assign courier to order", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("Courier assigned successfully")
}

func ReassignCourierToOrder(w http.ResponseWriter, r *http.Request) {
	orderIDStr := mux.Vars(r)["orderId"]
	orderID, err := primitive.ObjectIDFromHex(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid OrderID format", http.StatusBadRequest)
		return
	}

	var request struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	var courier Courier
	collection := client.Database("myapp").Collection("couriers")
	err = collection.FindOne(context.TODO(), bson.M{"email": request.Email}).Decode(&courier)
	if err != nil {
		http.Error(w, "Courier not found", http.StatusNotFound)
		return
	}

	// Get the order collection and check if the order is already assigned
	orderCollection := client.Database("myapp").Collection("orders")
	var order Order
	err = orderCollection.FindOne(context.TODO(), bson.M{"_id": orderID}).Decode(&order)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Check if the order is already assigned to a courier
	if order.CourierEmail != "" {
		filter := bson.M{"_id": orderID}
		update := bson.M{
			"$set": bson.M{
				"courierId":    courier.ID,
				"courierEmail": courier.Email,
				"courierPhone": courier.Phone,
				"courierName":  courier.Name,
				"reassigned":   true,
			},
		}

		_, err = orderCollection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			http.Error(w, "Failed to reassign courier to order", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("Courier reassigned successfully")
	} else {
		http.Error(w, "Order has not been assigned to a courier yet", http.StatusConflict)
		return
	}
}

func GetOrdersAssignedToCourier(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if request.Email == "" {
		http.Error(w, "Courier email is required", http.StatusBadRequest)
		return
	}

	var courier Courier
	collection := client.Database("myapp").Collection("couriers")
	err := collection.FindOne(context.TODO(), bson.M{"email": request.Email}).Decode(&courier)
	if err != nil {
		http.Error(w, "Courier not found", http.StatusNotFound)
		return
	}

	orderCollection := client.Database("myapp").Collection("orders")
	filter := bson.M{"courierId": courier.ID}

	cursor, err := orderCollection.Find(context.TODO(), filter)
	if err != nil {
		http.Error(w, "Failed to retrieve assigned orders", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	var orders []Order
	if err := cursor.All(context.TODO(), &orders); err != nil {
		http.Error(w, "Failed to decode orders", http.StatusInternalServerError)
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
