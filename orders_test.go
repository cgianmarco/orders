package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func GivenARunningApp(t *testing.T) (*httptest.Server, *sql.DB) {
	config := Config{
		DB_HOST:     os.Getenv("TEST_DB_HOST"),
		DB_PORT:     os.Getenv("TEST_DB_PORT"),
		DB_USER:     os.Getenv("TEST_DB_USER"),
		DB_PASSWORD: os.Getenv("TEST_DB_PASSWORD"),
		DB_NAME:     os.Getenv("TEST_DB_NAME"),
	}

	var db, err = sql.Open(
		"postgres",
		fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			config.DB_HOST,
			config.DB_PORT,
			config.DB_USER,
			config.DB_PASSWORD,
			config.DB_NAME,
		),
	)

	if err != nil {
		log.Println("Error connecting to database:", err)
	}

	// Setup db
	_, err = db.Exec(`
        TRUNCATE TABLE order_items, orders, items RESTART IDENTITY CASCADE
    `)
	if err != nil {
		t.Fatalf("Failed to clean tables: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO vat_categories (rate, name) VALUES
			(22, 'standard rate'),
			(10, 'reduced rate'),
			(5, 'special reduced rate'),
			(4, 'super reduced rate')
		ON CONFLICT (rate) DO NOTHING;
	`)
	if err != nil {
		t.Fatalf("Failed to insert VAT categories: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO items (name, quantityInStock, priceCents, vatCategoryId) VALUES
			('Laptop', 10, 99999, 1),
			('Mouse', 10, 2550, 1),
			('Keyboard', 10, 7500, 1),
			('Monitor', 10, 29999, 1),
			('Webcam', 10, 8999, 1),
			('Headphones', 10, 14999, 1),
			('USB Cable', 10, 1299, 1),
			('External SSD', 10, 17999, 1),
			('Desk Lamp', 10, 4550, 1),
			('Phone Stand', 10, 1999, 1);
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	handler := getHandler(db)
	server := httptest.NewServer(handler)

	return server, db
}

func Test_ShowsProducts(t *testing.T) {

	server, db := GivenARunningApp(t)
	defer server.Close()
	defer db.Close()

	resp, err := http.Get(server.URL + "/products")

	if err != nil {
		t.Fatalf("Failed to get products: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK, got %v", resp.Status)
	}

	var productsPage ProductsPage
	if err := json.NewDecoder(resp.Body).Decode(&productsPage); err != nil {
		t.Fatalf("Failed to decode products: %v", err)
	}

	if len(productsPage.Products) == 0 {
		t.Fatalf("Expected non-empty products list")
	}
}

func Test_PlaceOrder_Success(t *testing.T) {

	server, db := GivenARunningApp(t)
	defer server.Close()
	defer db.Close()

	items := []OrderItem{
		{ID: 1, Quantity: 2},
		{ID: 2, Quantity: 1},
	}

	placeOrderRequest := OrderToPlace{
		Items: items,
	}

	reqBody, err := json.Marshal(placeOrderRequest)
	if err != nil {
		t.Fatalf("Failed to encode order request: %v", err)
	}

	resp, err := http.Post(server.URL+"/orders", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK, got %v", resp.Status)
	}

	var orderResponse PlacedOrder
	if err := json.NewDecoder(resp.Body).Decode(&orderResponse); err != nil {
		t.Fatalf("Failed to decode order response: %v", err)
	}

	// Check product stock level
	resp, err = http.Get(server.URL + "/products")

	if err != nil {
		t.Fatalf("Failed to get products: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK, got %v", resp.Status)
	}

	var productsPage ProductsPage
	if err := json.NewDecoder(resp.Body).Decode(&productsPage); err != nil {
		t.Fatalf("Failed to decode products: %v", err)
	}

	updatedItems := make(map[int]Product)
	for _, p := range productsPage.Products {
		updatedItems[p.ID] = p
	}

	for _, orderedItem := range items {
		updatedProduct, exists := updatedItems[orderedItem.ID]
		if !exists {
			t.Fatalf("Ordered item with ID %d not found in products", orderedItem.ID)
		}
		expectedStock := 10 - orderedItem.Quantity
		if updatedProduct.QuantityInStock != expectedStock {
			t.Fatalf("Expected stock for item ID %d to be %d, got %d", orderedItem.ID, expectedStock, updatedProduct.QuantityInStock)
		}
	}
}

func Test_PlaceOrder_InsufficientStock(t *testing.T) {

	server, db := GivenARunningApp(t)
	defer server.Close()
	defer db.Close()

	items := []OrderItem{
		{ID: 1, Quantity: 20},
	}

	placeOrderRequest := OrderToPlace{
		Items: items,
	}

	reqBody, err := json.Marshal(placeOrderRequest)
	if err != nil {
		t.Fatalf("Failed to encode order request: %v", err)
	}

	resp, err := http.Post(server.URL+"/orders", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status BadRequest, got %v", resp.Status)
	}
}

func Test_PlaceOrder_ItemNotFound(t *testing.T) {
	server, db := GivenARunningApp(t)
	defer server.Close()
	defer db.Close()

	items := []OrderItem{
		{ID: 999, Quantity: 1},
	}

	placeOrderRequest := OrderToPlace{
		Items: items,
	}

	reqBody, err := json.Marshal(placeOrderRequest)
	if err != nil {
		t.Fatalf("Failed to encode order request: %v", err)
	}

	resp, err := http.Post(server.URL+"/orders", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status BadRequest, got %v", resp.Status)
	}
}

func Test_PlaceOrder_EmptyItems(t *testing.T) {
	server, db := GivenARunningApp(t)
	defer server.Close()
	defer db.Close()

	items := []OrderItem{}

	placeOrderRequest := OrderToPlace{
		Items: items,
	}

	reqBody, err := json.Marshal(placeOrderRequest)
	if err != nil {
		t.Fatalf("Failed to encode order request: %v", err)
	}

	resp, err := http.Post(server.URL+"/orders", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status BadRequest, got %v", resp.Status)
	}
}

func Test_PlaceOrder_RepeatedItem(t *testing.T) {
	server, db := GivenARunningApp(t)
	defer server.Close()
	defer db.Close()

	items := []OrderItem{
		{ID: 1, Quantity: 2},
		{ID: 1, Quantity: 4},
	}

	placeOrderRequest := OrderToPlace{
		Items: items,
	}

	reqBody, err := json.Marshal(placeOrderRequest)
	if err != nil {
		t.Fatalf("Failed to encode order request: %v", err)
	}

	resp, err := http.Post(server.URL+"/orders", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status BadRequest, got %v", resp.Status)
	}
}

func Test_PlaceOrder_ZeroQuantity(t *testing.T) {
	server, db := GivenARunningApp(t)
	defer server.Close()
	defer db.Close()

	items := []OrderItem{
		{ID: 1, Quantity: 0},
	}

	placeOrderRequest := OrderToPlace{
		Items: items,
	}

	reqBody, err := json.Marshal(placeOrderRequest)
	if err != nil {
		t.Fatalf("Failed to encode order request: %v", err)
	}

	resp, err := http.Post(server.URL+"/orders", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status BadRequest, got %v", resp.Status)
	}
}

func Test_PlaceOrder_NegativeQuantity(t *testing.T) {
	server, db := GivenARunningApp(t)
	defer server.Close()
	defer db.Close()

	items := []OrderItem{
		{ID: 1, Quantity: -5},
	}

	placeOrderRequest := OrderToPlace{
		Items: items,
	}

	reqBody, err := json.Marshal(placeOrderRequest)
	if err != nil {
		t.Fatalf("Failed to encode order request: %v", err)
	}

	resp, err := http.Post(server.URL+"/orders", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status BadRequest, got %v", resp.Status)
	}
}
