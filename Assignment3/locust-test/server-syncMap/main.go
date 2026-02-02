package main

import (
    "net/http"
    "sync"

    "github.com/gin-gonic/gin"
)

type album struct {
    ID     string  `json:"id"`
    Title  string  `json:"title"`
    Artist string  `json:"artist"`
    Price  float64 `json:"price"`
}

var albums sync.Map

func init() {
    // Initialize with default albums
    albums.Store("1", album{ID: "1", Title: "Blue Train", Artist: "John Coltrane", Price: 56.99})
    albums.Store("2", album{ID: "2", Title: "Jeru", Artist: "Gerry Mulligan", Price: 17.99})
    albums.Store("3", album{ID: "3", Title: "Sarah Vaughan and Clifford Brown", Artist: "Sarah Vaughan", Price: 39.99})
}

func main() {
    router := gin.Default()
    router.GET("/albums", getAlbums)
    router.POST("/albums", postAlbums)
    router.GET("/albums/:id", getAlbumByID)
    router.Run(":8080")
}

func getAlbums(c *gin.Context) {
    var result []album
    albums.Range(func(key, value interface{}) bool {
        result = append(result, value.(album))
        return true
    })
    c.IndentedJSON(http.StatusOK, result)
}

func postAlbums(c *gin.Context) {
    var newAlbum album
    if err := c.BindJSON(&newAlbum); err != nil {
        return
    }

    albums.Store(newAlbum.ID, newAlbum)
    c.IndentedJSON(http.StatusCreated, newAlbum)
}

func getAlbumByID(c *gin.Context) {
    id := c.Param("id")

    if value, ok := albums.Load(id); ok {
        c.IndentedJSON(http.StatusOK, value.(album))
        return
    }
    c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
}