package main

import (
	"bytes"
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

const WorkerCount = 20

type MovieJob struct {
	ID         int
	Title      string
	TitleTR    string
	Tagline    string
	TaglineTR  string
	Overview   string
	OverviewTR string
	Director   string
	Genres     string
	Keywords   string
	Cast       string
	Year       string
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(".env error")
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

	query := `
		SELECT id, title, COALESCE(title_tr, '') as title_tr, 
		       tagline, COALESCE(tagline_tr, '') as tagline_tr, 
		       overview, COALESCE(overview_tr, '') as overview_tr, 
		       director, release_date,
		COALESCE((SELECT string_agg(val->>'name', ', ') FROM jsonb_array_elements(CASE WHEN jsonb_typeof(genres) = 'array' THEN genres ELSE '[]'::jsonb END) val), '') as genres_list,
		COALESCE((SELECT string_agg(elem, ', ') FROM jsonb_array_elements_text(CASE WHEN jsonb_typeof(keywords) = 'array' THEN keywords ELSE '[]'::jsonb END) elem), '') as keywords_list,
		COALESCE((SELECT string_agg(elem, ', ') FROM jsonb_array_elements_text(CASE WHEN jsonb_typeof(cast_list) = 'array' THEN cast_list ELSE '[]'::jsonb END) elem), '') as cast_list_text
		FROM movies 
		WHERE embedding IS NULL AND overview_tr IS NOT NULL
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(rows)

	jobs := make(chan MovieJob, 100)
	var wg sync.WaitGroup

	for i := 0; i < WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{Timeout: 60 * time.Second}
			for j := range jobs {
				combinedText := fmt.Sprintf(
					"Represent this movie for retrieval: "+
						"Titles: [EN: %s | TR: %s]. Year: %s. Director: %s. "+
						"Metadata: {Genres: %s. Keywords: %s. Cast: %s}. "+
						"EN_Context: %s %s. TR_Baglam: %s %s.",
					j.Title, j.TitleTR, j.Year, j.Director, j.Genres, j.Keywords, j.Cast,
					j.Tagline, j.Overview, j.TaglineTR, j.OverviewTR,
				)

				emb, err := getEmbedding(combinedText, client)
				if err != nil {
					log.Printf("ID %d Error: %v", j.ID, err)
					continue
				}

				embJSON, _ := json.Marshal(emb)
				_, err = db.Exec("UPDATE movies SET embedding = $1 WHERE id = $2", string(embJSON), j.ID)
				if err == nil {
					fmt.Printf("Vektör Kaydedildi: %d\n", j.ID)
				}
			}
		}()
	}

	for rows.Next() {
		var j MovieJob
		var t, ttr, tg, tgtr, ov, ovtr, dir, rd, gn, kw, cs sql.NullString
		if err := rows.Scan(&j.ID, &t, &ttr, &tg, &tgtr, &ov, &ovtr, &dir, &rd, &gn, &kw, &cs); err != nil {
			continue
		}
		j.Title, j.TitleTR, j.Tagline, j.TaglineTR = t.String, ttr.String, tg.String, tgtr.String
		j.Overview, j.OverviewTR, j.Director = ov.String, ovtr.String, dir.String
		j.Year = "N/A"
		if rd.Valid && len(rd.String) >= 4 {
			j.Year = rd.String[:4]
		}
		j.Genres, j.Keywords, j.Cast = gn.String, kw.String, cs.String
		jobs <- j
	}

	close(jobs)
	wg.Wait()
}

func getEmbedding(input string, client *http.Client) ([]float32, error) {
	url := fmt.Sprintf("%s/api/embed", os.Getenv("OLLAMA_BASE_URL"))
	body, _ := json.Marshal(map[string]string{"model": "bge-m3", "input": input})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)
	var res struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, err
	}
	return res.Embeddings[0], nil
}
