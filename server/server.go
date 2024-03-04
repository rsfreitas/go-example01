package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type ExternalQuotation struct {
	Usdbrl Usdbrl `json:"USDBRL"`
}

type Usdbrl struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

type QuotationResponse struct {
	BID string `json:"bid"`
}

func main() {
	db, err := gorm.Open(sqlite.Open("cotacao.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	if err := db.AutoMigrate(&Usdbrl{}); err != nil {
		panic(err)
	}

	http.Handle("/cotacao", &QuotationHandler{db: db})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

type QuotationHandler struct {
	db *gorm.DB
}

func (c *QuotationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	quotation, err := makeExternalRequest()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if result := c.db.WithContext(ctx).Create(quotation.Usdbrl); result.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := QuotationResponse{
		BID: quotation.Usdbrl.Bid,
	}

	body, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func makeExternalRequest() (*ExternalQuotation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	client := &http.Client{}
	request, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var quotation ExternalQuotation
	if err := json.Unmarshal(body, &quotation); err != nil {
		return nil, err
	}

	return &quotation, nil
}
