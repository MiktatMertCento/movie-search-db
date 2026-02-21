package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type SearchRequest struct {
	Query string `json:"query"`
}

type MovieResponse struct {
	ID     int     `json:"ID"`
	TmdbID int     `json:"TmdbID"`
	Title  string  `json:"Title"`
	Tag    string  `json:"Tag"`
	Ov     string  `json:"Ov"`
	Post   string  `json:"Post"`
	Vote   float64 `json:"Vote"`
	Sim    float64 `json:"Sim"`
	Score  float64 `json:"Score"`
}

var db *sql.DB

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env bulunamadı")
	}
}

func main() {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE"))

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	app := fiber.New(fiber.Config{
		DisableStartupMessage: false,
	})
	app.Use(cors.New())

	app.Post("/api/search", handleSearch)

	log.Fatal(app.Listen(":8080"))
}

func handleSearch(c *fiber.Ctx) error {
	var req SearchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "geçersiz istek"})
	}

	if req.Query == "" {
		return c.Status(400).JSON(fiber.Map{"error": "sorgu boş olamaz"})
	}

	embedding, err := getEmbedding(req.Query)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "AI servisi hatası"})
	}

	vectorJSON, _ := json.Marshal(embedding)

	query := `
		SELECT id, tmdb_id, title, tagline, overview, poster_path, vote_average,
		       (1 - (embedding <=> $1)) as sim,
		       (((1 - (embedding <=> $1)) * 0.85) + ((vote_average / 10) * 0.10) + (LOG(GREATEST(popularity, 1)) / 10 * 0.05)) as score
		FROM movies
		WHERE embedding IS NOT NULL 
		  AND vote_count > 10
		  AND (1 - (embedding <=> $1)) > 0.35
		ORDER BY score DESC
		LIMIT 12;
	`

	rows, err := db.Query(query, string(vectorJSON))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	var results []MovieResponse
	for rows.Next() {
		var m MovieResponse
		var t, tg, ov, p sql.NullString
		if err := rows.Scan(&m.ID, &m.TmdbID, &t, &tg, &ov, &p, &m.Vote, &m.Sim, &m.Score); err != nil {
			continue
		}
		m.Title = t.String
		m.Tag = tg.String
		m.Ov = ov.String
		m.Post = p.String
		results = append(results, m)
	}

	if len(results) == 0 {
		return c.Status(404).JSON(fiber.Map{
			"message": "aradığınız kriterlere uygun sonuç bulunamadı",
			"results": []interface{}{},
		})
	}

	return c.JSON(results)
}

func getEmbedding(text string) ([]float32, error) {
	url := fmt.Sprintf("%s/api/embed", os.Getenv("OLLAMA_BASE_URL"))
	reqBody, _ := json.Marshal(map[string]string{
		"model": "bge-m3",
		"input": text,
	})

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama hatası: %d", resp.StatusCode)
	}

	var res struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	if len(res.Embeddings) == 0 {
		return nil, fmt.Errorf("embedding üretilemedi")
	}

	return res.Embeddings[0], nil
}
