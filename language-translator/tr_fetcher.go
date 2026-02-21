package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const (
	WorkerCount = 20
	TMDBURL     = "https://api.themoviedb.org/3/movie/%d?api_key=%s&language=tr-TR"
)

type TMDBFullResponse struct {
	Title      string  `json:"title"`
	Overview   string  `json:"overview"`
	Tagline    string  `json:"tagline"`
	PosterPath *string `json:"poster_path"`
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(".env yuklenemedi")
	}
}

func main() {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE"))

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(db)

	rows, err := db.Query("SELECT id, tmdb_id FROM movies WHERE tmdb_id IS NOT NULL AND overview_tr IS NULL ORDER BY popularity DESC")
	if err != nil {
		log.Fatal(err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(rows)

	jobs := make(chan [2]int, 100)
	var wg sync.WaitGroup

	for i := 0; i < WorkerCount; i++ {
		wg.Add(1)
		go worker(db, jobs, &wg)
	}

	for rows.Next() {
		var id, tmdbID int
		if err := rows.Scan(&id, &tmdbID); err != nil {
			continue
		}
		jobs <- [2]int{id, tmdbID}
	}

	close(jobs)
	wg.Wait()
	fmt.Println("Turkce veri senkronizasyonu tamamlandi.")
}

func worker(db *sql.DB, jobs <-chan [2]int, wg *sync.WaitGroup) {
	defer wg.Done()
	client := &http.Client{Timeout: 10 * time.Second}
	apiKey := os.Getenv("TMDB_API_KEY")

	for job := range jobs {
		dbID, tmdbID := job[0], job[1]
		data, err := fetchTRData(client, tmdbID, apiKey)
		if err != nil {
			continue
		}

		query := `
			UPDATE movies 
			SET title_tr = $1, overview_tr = $2, tagline_tr = $3, poster_path = COALESCE($4, poster_path)
			WHERE id = $5`

		_, err = db.Exec(query, data.Title, data.Overview, data.Tagline, data.PosterPath, dbID)
		if err != nil {
			log.Printf("ID %d Update Hatasi: %v", dbID, err)
		} else {
			fmt.Printf("Film %d (%s) Turkcelestirildi.\n", dbID, data.Title)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func fetchTRData(client *http.Client, tmdbID int, apiKey string) (*TMDBFullResponse, error) {
	url := fmt.Sprintf(TMDBURL, tmdbID, apiKey)
	resp, err := client.Get(url)
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
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var res TMDBFullResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}
