package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type ProductsPage struct {
	Products   []Product `json:"products"`
	NextCursor string    `json:"nextCursor,omitempty"`
}

type Product struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	QuantityInStock int    `json:"quantityInStock"`
	PriceCents      int64  `json:"priceCents"`
	VATRate         int    `json:"vatRate"`
}

type Cursor struct {
	ID int `json:"id"`
}

func EncodeCursor(cursor Cursor) (string, error) {
	b, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func DecodeCursor(encoded string) (Cursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Cursor{}, err
	}
	var cursor Cursor
	err = json.Unmarshal(b, &cursor)
	return cursor, err
}

func GetProductsHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		v, err := strconv.Atoi(l)
		if err != nil || v <= 0 {
			http.Error(w, "Invalid limit", http.StatusBadRequest)
			return
		}
		limit = v
	}

	cursor := r.URL.Query().Get("cursor")
	var rows *sql.Rows
	var err error
	if cursor != "" {
		cursor, err := DecodeCursor(cursor)
		if err != nil {
			http.Error(w, "Invalid cursor", http.StatusBadRequest)
			return
		}
		rows, err = db.Query(`
		SELECT
			items.id,
			items.name,
			items.quantityInStock,
			items.priceCents,
			vat_categories.rate
		FROM items
		JOIN vat_categories
			ON items.vatCategoryId = vat_categories.id
		WHERE items.id > $1
		ORDER BY items.id ASC
		LIMIT $2
		`, cursor.ID, limit)
	} else {
		rows, err = db.Query(`
		SELECT
			items.id,
			items.name,
			items.quantityInStock,
			items.priceCents,
			vat_categories.rate
		FROM items
		JOIN vat_categories
			ON items.vatCategoryId = vat_categories.id
		ORDER BY items.id ASC
		LIMIT $1
		`, limit)
	}

	if err != nil {
		log.Printf("Error retrieving products: %v\n", err)
		http.Error(w, "Failed to retrieve products", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var productsPage ProductsPage

	for rows.Next() {
		var product Product

		err = rows.Scan(
			&product.ID,
			&product.Name,
			&product.QuantityInStock,
			&product.PriceCents,
			&product.VATRate,
		)

		if err != nil {
			log.Printf("Error retrieving products: %v\n", err)
			http.Error(w, "Failed to retrieve products", http.StatusInternalServerError)
			return
		}

		productsPage.Products = append(productsPage.Products, product)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error retrieving products: %v\n", err)
		http.Error(w, "Failed to retrieve products", http.StatusInternalServerError)
		return
	}

	if len(productsPage.Products) == limit {
		lastProduct := productsPage.Products[len(productsPage.Products)-1]
		cursor := Cursor{ID: lastProduct.ID}
		productsPage.NextCursor, err = EncodeCursor(cursor)
		if err != nil {
			log.Printf("Error encoding cursor: %v\n", err)
			http.Error(w, "Failed to retrieve products", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(productsPage)
}
