package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

type OrderItem struct {
	ID       int `json:"id"`
	Quantity int `json:"quantity"`
}

type OrderToPlace struct {
	Items []OrderItem `json:"items"`
}

type SelectedItem struct {
	ID         int   `json:"id"`
	PriceCents int64 `json:"priceCents"`
	VATCents   int64 `json:"vatCents"`
	Quantity   int   `json:"quantity"`
}

type PlacedOrder struct {
	ID              int            `json:"id"`
	TotalPriceCents int64          `json:"totalPriceCents"`
	TotalVATCents   int64          `json:"totalVATCents"`
	Items           []SelectedItem `json:"items"`
}

func PlaceOrderHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Decode the request
	var request OrderToPlace
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	placedOrder, err := PlaceOrder(ctx, db, request)

	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyOrderItems):
			http.Error(w, "Order must contain at least one item", http.StatusBadRequest)

		case errors.Is(err, ErrInvalidItemQuantity):
			http.Error(w, "Order contains item with invalid quantity", http.StatusBadRequest)

		case errors.Is(err, ErrDuplicateOrderItem):
			http.Error(w, "Order contains duplicate items", http.StatusBadRequest)

		case errors.Is(err, ErrItemNotFound):
			http.Error(w, "One or more items in the order were not found", http.StatusBadRequest)

		case errors.Is(err, ErrInsufficientStock):
			http.Error(w, "One or more items in the order have insufficient stock", http.StatusBadRequest)

		default:
			http.Error(w, "Failed to place order", http.StatusInternalServerError)
		}
		log.Printf("Failed to place order: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(placedOrder)
}

var (
	ErrEmptyOrderItems     = errors.New("No items in order")
	ErrInvalidItemQuantity = errors.New("Invalid quantity for item")
	ErrDuplicateOrderItem  = errors.New("Duplicate item in order")
	ErrItemNotFound        = errors.New("Item not found")
	ErrInsufficientStock   = errors.New("Insufficient stock for item")
	ErrFailedToPlaceOrder  = errors.New("Failed to place order")
)

func PlaceOrder(ctx context.Context, db *sql.DB, orderToPlace OrderToPlace) (*PlacedOrder, error) {

	// Validate the request
	if len(orderToPlace.Items) == 0 {
		return nil, ErrEmptyOrderItems
	}

	seen := make(map[int]bool)

	for _, item := range orderToPlace.Items {
		if item.Quantity < 1 {
			return nil, ErrInvalidItemQuantity
		}
		if seen[item.ID] {
			return nil, ErrDuplicateOrderItem
		}

		seen[item.ID] = true
	}

	// Run order placement transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToPlaceOrder, err)
	}
	defer tx.Rollback()

	var placedOrder PlacedOrder

	for _, item := range orderToPlace.Items {
		var quantityInStock int
		var priceCents int64
		var vatRate int

		err = tx.QueryRow(`
			SELECT i.quantityInStock, i.priceCents, vc.rate 
			FROM items i
			JOIN vat_categories vc ON i.vatCategoryId = vc.id
			WHERE i.id = $1 
			FOR NO KEY UPDATE of i
			`, item.ID).Scan(&quantityInStock, &priceCents, &vatRate)

		if err != nil {
			if err == sql.ErrNoRows {
				return nil, ErrItemNotFound
			}

			return nil, fmt.Errorf("%w: %w", ErrFailedToPlaceOrder, err)
		}

		if quantityInStock < item.Quantity {
			return nil, fmt.Errorf("%w: item ID %d", ErrInsufficientStock, item.ID)
		}

		_, err = tx.Exec(`
			UPDATE items 
			SET quantityInStock = quantityInStock - $1 
			WHERE id = $2
		`, item.Quantity, item.ID)

		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToPlaceOrder, err)
		}

		VATCents := (priceCents*int64(vatRate) + 50) / 100

		placedOrder.TotalPriceCents += int64(item.Quantity) * priceCents
		placedOrder.TotalVATCents += int64(item.Quantity) * VATCents

		placedOrder.Items = append(placedOrder.Items, SelectedItem{
			ID:         item.ID,
			PriceCents: priceCents,
			VATCents:   VATCents,
			Quantity:   item.Quantity,
		})
	}

	err = tx.QueryRow(`INSERT INTO orders DEFAULT VALUES RETURNING id`).Scan(&placedOrder.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToPlaceOrder, err)
	}

	for _, item := range placedOrder.Items {
		_, err = tx.Exec(`
			INSERT INTO order_items (orderId, itemId, quantity, priceCents, vatCents) 
			VALUES ($1, $2, $3, $4, $5)
		`, placedOrder.ID, item.ID, item.Quantity, item.PriceCents, item.VATCents)

		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToPlaceOrder, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToPlaceOrder, err)
	}

	return &placedOrder, nil
}
