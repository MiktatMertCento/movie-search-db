package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const (
	WorkerCount = 15
	TMDBURL     = "https://api.themoviedb.org/3/movie/%d?api_key=%s&language=tr-TR&append_to_response=credits,keywords"
)

type TMDBFullResponse struct {
	Title            string  `json:"title"`
	Overview         string  `json:"overview"`
	Tagline          string  `json:"tagline"`
	PosterPath       *string `json:"poster_path"`
	ReleaseDate      string  `json:"release_date"`
	Popularity       float64 `json:"popularity"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
	OriginalLanguage string  `json:"original_language"`
	Genres           []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"genres"`
	Keywords struct {
		Keywords []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"keywords"`
	} `json:"keywords"`
	Credits struct {
		Cast []struct {
			Name      string `json:"name"`
			Character string `json:"character"`
			Order     int    `json:"order"`
		} `json:"cast"`
		Crew []struct {
			Name string `json:"name"`
			Job  string `json:"job"`
		} `json:"crew"`
	} `json:"credits"`
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

	db.SetMaxOpenConns(WorkerCount + 5)
	db.SetMaxIdleConns(WorkerCount)

	if err := prepareDatabase(db); err != nil {
		log.Fatalf("DB Hazirlik Hatasi: %v", err)
	}

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

	rowCount := 0
	for rows.Next() {
		var id, tmdbID int
		if err := rows.Scan(&id, &tmdbID); err != nil {
			continue
		}
		jobs <- [2]int{id, tmdbID}
		rowCount++
	}

	fmt.Printf("%d film için güncelleme işlemi başlatıldı...\n", rowCount)
	close(jobs)
	wg.Wait()
	fmt.Println("\nSenkronizasyon tamamlandı.")
}

func prepareDatabase(db *sql.DB) error {
	query := `
	ALTER TABLE movies ADD COLUMN IF NOT EXISTS genres JSONB;
	ALTER TABLE movies ADD COLUMN IF NOT EXISTS keywords JSONB;
	ALTER TABLE movies ADD COLUMN IF NOT EXISTS cast_list JSONB;
	ALTER TABLE movies ADD COLUMN IF NOT EXISTS director TEXT;
	`
	_, err := db.Exec(query)
	return err
}

func worker(db *sql.DB, jobs <-chan [2]int, wg *sync.WaitGroup) {
	defer wg.Done()
	client := &http.Client{Timeout: 15 * time.Second}
	apiKey := os.Getenv("TMDB_API_KEY")

	for job := range jobs {
		dbID, tmdbID := job[0], job[1]

		data, err := fetchTMDBData(client, tmdbID, apiKey)
		if err != nil {
			log.Printf("[Hata] TMDB ID %d: %v", tmdbID, err)
			continue
		}

		if err := performUpdate(db, dbID, data); err != nil {
			log.Printf("[Hata] DB Update ID %d: %v", dbID, err)
		} else {
			fmt.Printf("[OK] %s guncellendi.\n", data.Title)
		}

		time.Sleep(40 * time.Millisecond)
	}
}

func performUpdate(db *sql.DB, dbID int, data *TMDBFullResponse) error {
	var directors []string
	for _, member := range data.Credits.Crew {
		if member.Job == "Director" {
			directors = append(directors, member.Name)
		}
	}

	genresJSON, _ := json.Marshal(data.Genres)
	keywordsJSON, _ := json.Marshal(data.Keywords.Keywords)
	castJSON, _ := json.Marshal(data.Credits.Cast)

	query := `
		UPDATE movies 
		SET title_tr = $1, 
		    overview_tr = $2, 
		    tagline_tr = $3, 
		    poster_path = COALESCE($4, poster_path),
		    director = $5,
		    genres = $6,
		    keywords = $7,
		    cast_list = $8,
		    release_date = NULLIF($9, '')::DATE,
		    popularity = $10,
		    vote_average = $11,
		    vote_count = $12,
		    original_language = $13
		WHERE id = $14`

	_, err := db.Exec(query,
		data.Title, data.Overview, data.Tagline, data.PosterPath,
		strings.Join(directors, ", "),
		genresJSON, keywordsJSON, castJSON,
		data.ReleaseDate, data.Popularity, data.VoteAverage, data.VoteCount,
		data.OriginalLanguage, dbID,
	)
	return err
}

func fetchTMDBData(client *http.Client, tmdbID int, apiKey string) (*TMDBFullResponse, error) {
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
