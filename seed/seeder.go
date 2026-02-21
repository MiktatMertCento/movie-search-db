package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env dosyasi yuklenemedi")
	}
}

func main() {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE"))

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("DB baglanti hatasi: %v", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(db)

	if err := db.Ping(); err != nil {
		log.Fatalf("DB erisim hatasi: %v", err)
	}

	// Tablo oluşturma bloğu
	createTableSQL := `
    CREATE EXTENSION IF NOT EXISTS vector;
    CREATE TABLE IF NOT EXISTS movies (
        id SERIAL PRIMARY KEY,
        tmdb_id INTEGER UNIQUE,
        title TEXT,
        title_tr TEXT,
        tagline TEXT,
        tagline_tr TEXT,
        overview TEXT,
        overview_tr TEXT,
        genres JSONB,
        keywords JSONB,
        cast_list JSONB,
        director TEXT,
        release_date DATE,
        popularity DOUBLE PRECISION,
        vote_average DOUBLE PRECISION,
        vote_count INTEGER,
        original_language TEXT,
        poster_path TEXT,
        embedding vector(1024)
    );`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Tablo olusturma hatasi: %v", err)
	}

	keywordsMap := loadKeywords()
	castMap, directorsMap := loadCredits()

	file, err := os.Open("datas/movies_metadata.csv")
	if err != nil {
		log.Fatalf("CSV acilamadi: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(file)

	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	header, err := reader.Read()
	if err != nil {
		log.Fatalf("Header okunamadi: %v", err)
	}

	colMap := make(map[string]int)
	for i, name := range header {
		colMap[name] = i
	}

	query := `
       INSERT INTO movies (tmdb_id, title, tagline, overview, genres, keywords, cast_list, director, release_date, popularity, vote_average, original_language, vote_count) 
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) 
       ON CONFLICT (tmdb_id) DO UPDATE SET 
          popularity = EXCLUDED.popularity,
          vote_average = EXCLUDED.vote_average,
          vote_count = EXCLUDED.vote_count,
          tagline = CASE WHEN movies.tagline IS NULL OR movies.tagline = '' THEN EXCLUDED.tagline ELSE movies.tagline END,
          overview = CASE WHEN movies.overview IS NULL OR movies.overview = '' THEN EXCLUDED.overview ELSE movies.overview END
    `
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Fatalf("Statement hatasi: %v", err)
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(stmt)

	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		tmdbID, _ := strconv.Atoi(record[colMap["id"]])
		if tmdbID == 0 {
			continue
		}

		pop, _ := strconv.ParseFloat(record[colMap["popularity"]], 64)
		vote, _ := strconv.ParseFloat(record[colMap["vote_average"]], 64)
		vCount, _ := strconv.Atoi(record[colMap["vote_count"]])

		var releaseDate interface{}
		if t, err := time.Parse("2006-01-02", record[colMap["release_date"]]); err == nil {
			releaseDate = t
		}

		genresRaw := strings.ReplaceAll(record[colMap["genres"]], "'", "\"")
		var genres []map[string]interface{}
		_ = json.Unmarshal([]byte(genresRaw), &genres)
		genresJSON, _ := json.Marshal(genres)

		kJSON, _ := json.Marshal(keywordsMap[tmdbID])
		cJSON, _ := json.Marshal(castMap[tmdbID])
		director := directorsMap[tmdbID]

		_, err = stmt.Exec(tmdbID, record[colMap["title"]], record[colMap["tagline"]], record[colMap["overview"]],
			genresJSON, kJSON, cJSON, director, releaseDate, pop, vote, record[colMap["original_language"]], vCount)

		if err != nil {
			log.Printf("ID %d yazma hatasi: %v", tmdbID, err)
			continue
		}

		count++
		if count%2000 == 0 {
			fmt.Printf("%d film verisi islendi\n", count)
		}
	}
	fmt.Printf("Islem tamamlandi. Toplam %d film güncellendi/eklendi.\n", count)
}

func loadKeywords() map[int][]string {
	m := make(map[int][]string)
	f, err := os.Open("datas/keywords.csv")
	if err != nil {
		return m
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(f)

	r := csv.NewReader(f)
	_, err = r.Read()
	if err != nil {
		return nil
	}
	for {
		rec, err := r.Read()
		if err != nil {
			break
		}
		id, _ := strconv.Atoi(rec[0])
		var kw []map[string]interface{}
		cleanJSON := strings.ReplaceAll(rec[1], "'", "\"")
		if err := json.Unmarshal([]byte(cleanJSON), &kw); err == nil {
			for _, k := range kw {
				if name, ok := k["name"].(string); ok {
					m[id] = append(m[id], name)
				}
			}
		}
	}
	return m
}

func loadCredits() (map[int][]string, map[int]string) {
	castM := make(map[int][]string)
	dirM := make(map[int]string)
	f, err := os.Open("datas/credits.csv")
	if err != nil {
		return castM, dirM
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(f)

	r := csv.NewReader(f)
	_, err = r.Read()
	if err != nil {
		return nil, nil
	}
	for {
		rec, err := r.Read()
		if err != nil {
			break
		}
		id, _ := strconv.Atoi(rec[2])

		var castRaw []map[string]interface{}
		cleanCast := strings.ReplaceAll(rec[0], "'", "\"")
		if err := json.Unmarshal([]byte(cleanCast), &castRaw); err == nil {
			for i, c := range castRaw {
				if i > 4 {
					break
				}
				if name, ok := c["name"].(string); ok {
					castM[id] = append(castM[id], name)
				}
			}
		}

		var crewRaw []map[string]interface{}
		cleanCrew := strings.ReplaceAll(rec[1], "'", "\"")
		if err := json.Unmarshal([]byte(cleanCrew), &crewRaw); err == nil {
			for _, cr := range crewRaw {
				if cr["job"] == "Director" {
					if name, ok := cr["name"].(string); ok {
						dirM[id] = name
						break
					}
				}
			}
		}
	}
	return castM, dirM
}
