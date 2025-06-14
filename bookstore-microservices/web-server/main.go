package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type BookStore struct {
	ID          string `json:"ID"`
	BookName    string `json:"BookName"`
	BookAuthor  string `json:"BookAuthor"`
	BookEdition string `json:"BookEdition"`
	BookPages   string `json:"BookPages"`
}

type BookResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Author  string `json:"author"`
	Pages   string `json:"pages"`
	Edition string `json:"edition"`
	Year    string `json:"year"`
}

type Template struct {
	tmpl *template.Template
}

func loadTemplates() *Template {
	return &Template{
		tmpl: template.Must(template.ParseGlob("views/*.html")),
	}
}

func (t *Template) Render(w io.Writer, name string, data interface{}, ctx echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

func getBooksFromAPI() ([]BookStore, error) {
	booksGetURL := os.Getenv("BOOKS_GET_URL")
	if booksGetURL == "" {
		booksGetURL = "http://books-get:8080"
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(booksGetURL + "/api/books")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var books []BookResponse
	if err := json.NewDecoder(resp.Body).Decode(&books); err != nil {
		return nil, err
	}

	// Convert BookResponse to BookStore for template compatibility
	var result []BookStore
	for _, book := range books {
		result = append(result, BookStore{
			ID:          book.ID,
			BookName:    book.Title,
			BookAuthor:  book.Author,
			BookEdition: book.Edition,
			BookPages:   book.Pages,
		})
	}

	return result, nil
}

func getAuthorsFromAPI() ([]map[string]interface{}, error) {
	books, err := getBooksFromAPI()
	if err != nil {
		return nil, err
	}

	// Extract unique authors
	authorsMap := make(map[string]bool)
	for _, book := range books {
		authorsMap[book.BookAuthor] = true
	}

	var authors []map[string]interface{}
	for author := range authorsMap {
		authors = append(authors, map[string]interface{}{
			"Author": author,
		})
	}

	return authors, nil
}

func getYearsFromAPI() ([]map[string]interface{}, error) {
	booksGetURL := os.Getenv("BOOKS_GET_URL")
	if booksGetURL == "" {
		booksGetURL = "http://books-get:8080"
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(booksGetURL + "/api/books")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var books []BookResponse
	if err := json.NewDecoder(resp.Body).Decode(&books); err != nil {
		return nil, err
	}

	// Extract unique years
	yearsMap := make(map[string]bool)
	for _, book := range books {
		yearsMap[book.Year] = true
	}

	var years []map[string]interface{}
	for year := range yearsMap {
		years = append(years, map[string]interface{}{
			"Year": year,
		})
	}

	return years, nil
}

func main() {
	// Wait for other services to be ready
	time.Sleep(10 * time.Second)

	e := echo.New()
	e.Renderer = loadTemplates()
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())
	e.Static("/css", "css")

	e.GET("/", func(c echo.Context) error {
		return c.Render(200, "index", nil)
	})

	e.GET("/books", func(c echo.Context) error {
		books, err := getBooksFromAPI()
		if err != nil {
			log.Printf("Error fetching books: %v", err)
			return c.Render(500, "index", map[string]string{"error": "Failed to fetch books"})
		}
		return c.Render(200, "book-table", books)
	})

	e.GET("/authors", func(c echo.Context) error {
		authors, err := getAuthorsFromAPI()
		if err != nil {
			log.Printf("Error fetching authors: %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		return c.Render(200, "authors-table", authors)
	})

	e.GET("/years", func(c echo.Context) error {
		years, err := getYearsFromAPI()
		if err != nil {
			log.Printf("Error fetching years: %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		return c.Render(200, "years-table", years)
	})

	e.GET("/search", func(c echo.Context) error {
		return c.Render(200, "search-bar", nil)
	})

	e.GET("/create", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	fmt.Println("Web server starting on port 8080")
	e.Logger.Fatal(e.Start(":8080"))
}