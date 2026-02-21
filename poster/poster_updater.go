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
	TMDBBaseURL = "https://api.themoviedb.org/3/movie/%d?api_key=%s"
)

type TMDBMovieResponse struct {
	PosterPath string `json:"poster_path"`
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(".env dosyasi yuklenemedi")
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

	rows, err := db.Query("SELECT id, tmdb_id FROM movies WHERE tmdb_id IS NOT NULL ORDER BY popularity DESC")
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
	fmt.Println("Poster guncelleme islemi bitti.")
}

func worker(db *sql.DB, jobs <-chan [2]int, wg *sync.WaitGroup) {
	defer wg.Done()
	client := &http.Client{Timeout: 10 * time.Second}
	apiKey := os.Getenv("TMDB_API_KEY")

	for job := range jobs {
		dbID := job[0]
		tmdbID := job[1]

		newPath, err := fetchCurrentPoster(client, tmdbID, apiKey)
		if err != nil {
			log.Printf("TMDB ID %d hatasi: %v", tmdbID, err)
			continue
		}

		if newPath != "" {
			_, err = db.Exec("UPDATE movies SET poster_path = $1 WHERE id = $2", newPath, dbID)
			if err != nil {
				log.Printf("DB update hatasi (ID %d): %v", dbID, err)
			} else {
				fmt.Printf("Film %d guncellendi: %s\n", dbID, newPath)
			}
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func fetchCurrentPoster(client *http.Client, tmdbID int, apiKey string) (string, error) {
	url := fmt.Sprintf(TMDBBaseURL, tmdbID, apiKey)
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var res TMDBMovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.PosterPath, nil
}
