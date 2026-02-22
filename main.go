package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type SearchRequest struct {
	Query        string `json:"query"`
	CaptchaToken string `json:"captchaToken"`
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

type RecaptchaResponse struct {
	Success bool    `json:"success"`
	Score   float64 `json:"score"`
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
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(db)

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Minute * 5)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: false,
		ReadTimeout:           10 * time.Second,
	})

	app.Use(cors.New())

	app.Post("/api/search", handleSearch)

	log.Fatal(app.Listen(":8080"))
}

func handleSearch(c *fiber.Ctx) error {
	var req SearchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid_request"})
	}

	if req.CaptchaToken == "" {
		return c.Status(400).JSON(fiber.Map{"error": "captcha_required"})
	}

	if req.Query == "" {
		return c.Status(400).JSON(fiber.Map{"error": "query_required"})
	}

	valid, err := verifyRecaptcha(req.CaptchaToken)
	if err != nil || !valid {
		return c.Status(403).JSON(fiber.Map{"error": "bot_detected"})
	}

	embedding, err := getEmbedding(req.Query)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "embedding_failed"})
	}

	vectorJSON, _ := json.Marshal(embedding)

	const query = `WITH MatchData AS (
    SELECT 
        id, 
        tmdb_id, 
        COALESCE(NULLIF(TRIM(title_tr), ''), title) AS title,
        COALESCE(NULLIF(TRIM(tagline_tr), ''), tagline) AS tagline,
        COALESCE(NULLIF(TRIM(overview_tr), ''), overview) AS overview,
        poster_path, 
        vote_average,
        popularity,
        (1 - (embedding <=> $1)) AS sim
    FROM movies
    WHERE embedding IS NOT NULL 
      AND vote_count > 10
)
SELECT 
    id, 
    tmdb_id, 
    title, 
    tagline, 
    overview, 
    poster_path, 
    vote_average,
    sim,
    ((sim * 0.85) + ((vote_average / 10.0) * 0.10) + (LOG(GREATEST(popularity, 1.0)) / 10.0 * 0.05)) AS score
FROM MatchData
WHERE sim > 0.35
ORDER BY score DESC
LIMIT 12;`

	rows, err := db.Query(query, string(vectorJSON))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "database_error"})
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(rows)

	results := make([]MovieResponse, 0)
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
		return c.Status(404).JSON(fiber.Map{"message": "no_results", "results": []MovieResponse{}})
	}

	return c.JSON(results)
}

func verifyRecaptcha(token string) (bool, error) {
	secret := os.Getenv("RECAPTCHA_PRIVATE_KEY")
	data := url.Values{
		"secret":   {secret},
		"response": {token},
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.PostForm("https://www.google.com/recaptcha/api/siteverify", data)
	if err != nil {
		return false, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	var res RecaptchaResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return false, err
	}

	return res.Success && res.Score >= 0.5, nil
}

func getEmbedding(text string) ([]float32, error) {
	ollamaUrl := fmt.Sprintf("%s/api/embed", os.Getenv("OLLAMA_BASE_URL"))
	reqBody, _ := json.Marshal(map[string]string{
		"model": "bge-m3",
		"input": text,
	})

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(ollamaUrl, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama_status_%d", resp.StatusCode)
	}

	var res struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	if len(res.Embeddings) == 0 {
		return nil, fmt.Errorf("empty_embedding")
	}

	return res.Embeddings[0], nil
}
